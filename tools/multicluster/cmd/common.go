package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type clusterType string

// This tool handles the creation of ServiceAccounts and roles across multiple clusters.
// Service Accounts, Roles and RoleBindings are created in all the member clusters and the central cluster.
// The Service Account token secrets from the member clusters are merged into a KubeConfig file which is then
// created in the central cluster.

var memberClusters string
var memberClustersApiServers string

const (
	kubeConfigEnv              = "KUBECONFIG"
	centralCluster clusterType = "CENTRAL"
	memberCluster  clusterType = "MEMBER"
)

// flags holds all the fields provided by the user.
type flags struct {
	memberClusters              []string
	memberClusterApiServerUrls  []string
	serviceAccount              string
	centralCluster              string
	memberClusterNamespace      string
	centralClusterNamespace     string
	cleanup                     bool
	clusterScoped               bool
	installDatabaseRoles        bool
	operatorName                string
	sourceCluster               string
	createServiceAccountSecrets bool
}

const (
	kubeConfigSecretName         = "mongodb-enterprise-operator-multi-cluster-kubeconfig"
	kubeConfigSecretKey          = "kubeconfig"
	appdbServiceAccount          = "mongodb-enterprise-appdb"
	databasePodsServiceAccount   = "mongodb-enterprise-database-pods"
	opsManagerServiceAccount     = "mongodb-enterprise-ops-manager"
	appdbRole                    = "mongodb-enterprise-appdb"
	appdbRoleBinding             = "mongodb-enterprise-appdb"
	defaultOperatorName          = "mongodb-enterprise-operator"
	defaultOperatorConfigMapName = defaultOperatorName + "-member-list"
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// anyAreEmpty returns true if any of the given strings have the zero value.
func anyAreEmpty(values ...string) bool {
	for _, v := range values {
		if v == "" {
			return true
		}
	}
	return false
}

// getMemberClusterApiServerUrls returns the slice of member cluster api urls that should be used.
func getMemberClusterApiServerUrls(kubeconfig *clientcmdapi.Config, clusterNames []string) ([]string, error) {
	var urls []string
	for _, name := range clusterNames {
		if cluster := kubeconfig.Clusters[name]; cluster != nil {
			urls = append(urls, cluster.Server)
		} else {
			return nil, xerrors.Errorf("cluster '%s' not found in kubeconfig", name)
		}
	}
	return urls, nil
}

// KubeConfigFile represents the contents of a KubeConfig file.
type KubeConfigFile struct {
	ApiVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Clusters   []KubeConfigClusterItem `json:"clusters"`
	Contexts   []KubeConfigContextItem `json:"contexts"`
	Users      []KubeConfigUserItem    `json:"users"`
}

type KubeConfigClusterItem struct {
	Name    string            `json:"name"`
	Cluster KubeConfigCluster `json:"cluster"`
}

type KubeConfigCluster struct {
	CertificateAuthorityData []byte `json:"certificate-authority-data"`
	Server                   string `json:"server"`
}

type KubeConfigContextItem struct {
	Name    string            `json:"name"`
	Context KubeConfigContext `json:"context"`
}

type KubeConfigContext struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	User      string `json:"user"`
}

type KubeConfigUserItem struct {
	Name string         `json:"name"`
	User KubeConfigUser `json:"user"`
}

type KubeConfigUser struct {
	Token string `json:"token"`
}

// multiClusterLabels the labels that will be applied to every resource created by this tool.
func multiClusterLabels() map[string]string {
	return map[string]string{
		"multi-cluster": "true",
	}
}

// createClientMap crates a map of all MultiClusterClient for every member cluster, and the operator cluster.
func createClientMap(memberClusters []string, operatorCluster, kubeConfigPath string, getClient func(clusterName string, kubeConfigPath string) (kubernetes.Interface, error)) (map[string]kubernetes.Interface, error) {
	clientMap := map[string]kubernetes.Interface{}
	for _, c := range memberClusters {
		clientset, err := getClient(c, kubeConfigPath)
		if err != nil {
			return nil, xerrors.Errorf("failed to create clientset map: %w", err)
		}
		clientMap[c] = clientset
	}

	clientset, err := getClient(operatorCluster, kubeConfigPath)
	if err != nil {
		return nil, xerrors.Errorf("failed to create clientset map: %w", err)
	}
	clientMap[operatorCluster] = clientset
	return clientMap, nil
}

// loadKubeConfigFilePath returns the path of the local KubeConfig file.
func loadKubeConfigFilePath() string {
	env := os.Getenv(kubeConfigEnv)
	if env != "" {
		return env
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

// getKubernetesClient returns a kubernetes.Clientset using the given context from the
// specified KubeConfig filepath.
func getKubernetesClient(context, kubeConfigPath string) (kubernetes.Interface, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()

	if err != nil {
		return nil, xerrors.Errorf("failed to create client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, xerrors.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return clientset, nil
}

// performCleanup cleans up all of the resources that were created by this script in the past.
func performCleanup(ctx context.Context, clientMap map[string]kubernetes.Interface, flags flags) error {
	for _, cluster := range flags.memberClusters {
		c := clientMap[cluster]
		if err := cleanupClusterResources(ctx, c, cluster, flags.memberClusterNamespace); err != nil {
			return xerrors.Errorf("failed cleaning up cluster %s namespace %s: %w", cluster, flags.memberClusterNamespace, err)
		}
	}
	c := clientMap[flags.centralCluster]
	if err := cleanupClusterResources(ctx, c, flags.centralCluster, flags.centralClusterNamespace); err != nil {
		return xerrors.Errorf("failed cleaning up cluster %s namespace %s: %w", flags.centralCluster, flags.centralClusterNamespace, err)
	}
	return nil
}

// cleanupClusterResources cleans up all the resources created by this tool in a given namespace.
func cleanupClusterResources(ctx context.Context, clientset kubernetes.Interface, clusterName, namespace string) error {
	listOpts := metav1.ListOptions{
		LabelSelector: "multi-cluster=true",
	}

	// clean up secrets
	secretList, err := clientset.CoreV1().Secrets(namespace).List(ctx, listOpts)

	if err != nil {
		return err
	}

	if secretList != nil {
		for _, s := range secretList.Items {
			fmt.Printf("Deleting Secret: %s in cluster %s\n", s.Name, clusterName)
			if err := clientset.CoreV1().Secrets(namespace).Delete(ctx, s.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	// clean up service accounts
	serviceAccountList, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, listOpts)

	if err != nil {
		return err
	}

	if serviceAccountList != nil {
		for _, sa := range serviceAccountList.Items {
			fmt.Printf("Deleting ServiceAccount: %s in cluster %s\n", sa.Name, clusterName)
			if err := clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, sa.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	// clean up roles
	roleList, err := clientset.RbacV1().Roles(namespace).List(ctx, listOpts)
	if err != nil {
		return err
	}

	for _, r := range roleList.Items {
		fmt.Printf("Deleting Role: %s in cluster %s\n", r.Name, clusterName)
		if err := clientset.RbacV1().Roles(namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	// clean up roles
	roles, err := clientset.RbacV1().Roles(namespace).List(ctx, listOpts)
	if err != nil {
		return err
	}

	if roles != nil {
		for _, r := range roles.Items {
			fmt.Printf("Deleting Role: %s in cluster %s\n", r.Name, clusterName)
			if err := clientset.RbacV1().Roles(namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	// clean up role bindings
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, listOpts)
	if !errors.IsNotFound(err) && err != nil {
		return err
	}

	if roleBindings != nil {
		for _, crb := range roleBindings.Items {
			fmt.Printf("Deleting RoleBinding: %s in cluster %s\n", crb.Name, clusterName)
			if err := clientset.RbacV1().RoleBindings(namespace).Delete(ctx, crb.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	// clean up cluster role bindings
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, listOpts)
	if !errors.IsNotFound(err) && err != nil {
		return err
	}

	if clusterRoleBindings != nil {
		for _, crb := range clusterRoleBindings.Items {
			fmt.Printf("Deleting ClusterRoleBinding: %s in cluster %s\n", crb.Name, clusterName)
			if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	// clean up cluster roles
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, listOpts)
	if !errors.IsNotFound(err) && err != nil {
		return err
	}

	if clusterRoles != nil {
		for _, cr := range clusterRoles.Items {
			fmt.Printf("Deleting ClusterRole: %s in cluster %s\n", cr.Name, clusterName)
			if err := clientset.RbacV1().ClusterRoles().Delete(ctx, cr.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	return nil
}

// ensureNamespace creates the namespace with the given clientset.
func ensureNamespace(ctx context.Context, clientSet kubernetes.Interface, nsName string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: multiClusterLabels(),
		},
	}
	_, err := clientSet.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("failed to create namespace %s: %w", ns.Name, err)
	}

	return nil
}

// ensureAllClusterNamespacesExist makes sure the namespace we will be creating exists in all clusters.
func ensureAllClusterNamespacesExist(ctx context.Context, clientSets map[string]kubernetes.Interface, f flags) error {
	for _, clusterName := range f.memberClusters {
		if err := ensureNamespace(ctx, clientSets[clusterName], f.memberClusterNamespace); err != nil {
			return xerrors.Errorf("failed to ensure namespace %s in member cluster %s: %w", f.memberClusterNamespace, clusterName, err)
		}
	}
	if err := ensureNamespace(ctx, clientSets[f.centralCluster], f.centralClusterNamespace); err != nil {
		return xerrors.Errorf("failed to ensure namespace %s in central cluster %s: %w", f.centralClusterNamespace, f.centralCluster, err)
	}
	return nil
}

// ensureMultiClusterResources copies the ServiceAccount Secret tokens from the specified
// member clusters, merges them into a KubeConfig file and creates a Secret in the central cluster
// with the contents.
func ensureMultiClusterResources(ctx context.Context, flags flags, clientMap map[string]kubernetes.Interface) error {
	if flags.cleanup {
		if err := performCleanup(ctx, clientMap, flags); err != nil {
			return xerrors.Errorf("failed performing cleanup of resources: %w", err)
		}
	}

	if err := ensureAllClusterNamespacesExist(ctx, clientMap, flags); err != nil {
		return xerrors.Errorf("failed ensuring namespaces: %w", err)
	}
	fmt.Println("Ensured namespaces exist in all clusters.")

	if err := createServiceAccountsAndRoles(ctx, clientMap, flags); err != nil {
		return xerrors.Errorf("failed creating service accounts and roles in all clusters: %w", err)
	}
	fmt.Println("Ensured ServiceAccounts and Roles.")

	secrets, err := getAllWorkerClusterServiceAccountSecretTokens(ctx, clientMap, flags)
	if err != nil {
		return xerrors.Errorf("failed to get service account secret tokens: %w", err)
	}

	if len(secrets) != len(flags.memberClusters) {
		return xerrors.Errorf("required %d serviceaccount tokens but found only %d\n", len(flags.memberClusters), len(secrets))
	}

	kubeConfig, err := createKubeConfigFromServiceAccountTokens(secrets, flags)
	if err != nil {
		return xerrors.Errorf("failed to create kube config from service account tokens: %w", err)
	}

	kubeConfigBytes, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return xerrors.Errorf("failed to marshal kubeconfig: %w", err)
	}

	centralClusterClient := clientMap[flags.centralCluster]
	if err != nil {
		return xerrors.Errorf("failed to get central cluster clientset: %w", err)
	}

	if err := createKubeConfigSecret(ctx, centralClusterClient, kubeConfigBytes, flags); err != nil {
		return xerrors.Errorf("failed creating KubeConfig secret: %w", err)
	}

	if flags.sourceCluster != "" {
		if err := setupDatabaseRoles(ctx, clientMap, flags); err != nil {
			return xerrors.Errorf("failed setting up database roles: %w", err)
		}
		fmt.Println("Ensured database Roles in member clusters.")
	} else if flags.installDatabaseRoles {
		if err := installDatabaseRoles(ctx, clientMap, flags); err != nil {
			return xerrors.Errorf("failed installing database roles: %w", err)
		}
		fmt.Println("Ensured database Roles in member clusters.")
	}

	return nil
}

// createKubeConfigSecret creates the secret containing the KubeConfig file made from the various
// service account tokens in the member clusters.
func createKubeConfigSecret(ctx context.Context, centralClusterClient kubernetes.Interface, kubeConfigBytes []byte, flags flags) error {
	kubeConfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeConfigSecretName,
			Namespace: flags.centralClusterNamespace,
			Labels:    multiClusterLabels(),
		},
		Data: map[string][]byte{
			kubeConfigSecretKey: kubeConfigBytes,
		},
	}

	fmt.Printf("Creating KubeConfig secret %s/%s in cluster %s\n", flags.centralClusterNamespace, kubeConfigSecret.Name, flags.centralCluster)
	_, err := centralClusterClient.CoreV1().Secrets(flags.centralClusterNamespace).Create(ctx, &kubeConfigSecret, metav1.CreateOptions{})

	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("failed creating secret: %w", err)
	}

	if errors.IsAlreadyExists(err) {
		_, err = centralClusterClient.CoreV1().Secrets(flags.centralClusterNamespace).Update(ctx, &kubeConfigSecret, metav1.UpdateOptions{})
		if err != nil {
			return xerrors.Errorf("failed updating existing secret: %w", err)
		}
	}

	return nil
}

func getCentralRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			Verbs: []string{"*"},
			Resources: []string{
				"mongodbmulticluster", "mongodbmulticluster/finalizers", "mongodbmulticluster/status",
				"mongodbusers", "mongodbusers/status",
				"opsmanagers", "opsmanagers/finalizers", "opsmanagers/status",
				"mongodb", "mongodb/finalizers", "mongodb/status"},
			APIGroups: []string{"mongodb.com"},
		},
	}
}

func buildCentralEntityRole(namespace string) rbacv1.Role {
	rules := append(getCentralRules(), getMemberRules()...)
	return rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mongodb-enterprise-operator-multi-role",
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
		Rules: rules,
	}
}

func buildCentralEntityClusterRole() rbacv1.ClusterRole {
	rules := append(getCentralRules(), getMemberRules()...)
	rules = append(rules, rbacv1.PolicyRule{
		Verbs:     []string{"list", "watch"},
		Resources: []string{"namespaces"},
		APIGroups: []string{""},
	})

	return rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "mongodb-enterprise-operator-multi-cluster-role",
			Labels: multiClusterLabels(),
		},
		Rules: rules,
	}
}
func getMemberRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get", "list", "create", "update", "delete", "watch", "deletecollection"},
			Resources: []string{"secrets", "configmaps", "services"},
			APIGroups: []string{""},
		},
		{
			Verbs:     []string{"get", "list", "create", "update", "delete", "watch", "deletecollection"},
			Resources: []string{"statefulsets"},
			APIGroups: []string{"apps"},
		},
		{
			Verbs:     []string{"get", "list", "watch"},
			Resources: []string{"pods"},
			APIGroups: []string{""},
		},
	}
}

func buildMemberEntityRole(namespace string) rbacv1.Role {
	return rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mongodb-enterprise-operator-multi-role",
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
		Rules: getMemberRules(),
	}
}

func buildMemberEntityClusterRole() rbacv1.ClusterRole {
	rules := append(getMemberRules(), rbacv1.PolicyRule{
		Verbs:     []string{"list", "watch"},
		Resources: []string{"namespaces"},
		APIGroups: []string{""},
	})

	return rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "mongodb-enterprise-operator-multi-cluster-role",
			Labels: multiClusterLabels(),
		},
		Rules: rules,
	}
}

// buildRoleBinding creates the RoleBinding which binds the Role to the given ServiceAccount.
func buildRoleBinding(role rbacv1.Role, serviceAccount string) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mongodb-enterprise-operator-multi-role-binding",
			Labels:    multiClusterLabels(),
			Namespace: role.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: role.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     role.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

// buildClusterRoleBinding creates the ClusterRoleBinding which binds the ClusterRole to the given ServiceAccount.
func buildClusterRoleBinding(clusterRole rbacv1.ClusterRole, sa corev1.ServiceAccount) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "mongodb-enterprise-operator-multi-cluster-role-binding",
			Labels: multiClusterLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

// createMemberServiceAccountAndRoles creates the ServiceAccount and Roles, RoleBindings, ClusterRoles and ClusterRoleBindings required
// for the member clusters.
func createMemberServiceAccountAndRoles(ctx context.Context, c kubernetes.Interface, f flags) error {
	return createServiceAccountAndRoles(ctx, c, f.serviceAccount, f.memberClusterNamespace, f.clusterScoped, memberCluster)
}

// createCentralClusterServiceAccountAndRoles creates the ServiceAccount and Roles, RoleBindings, ClusterRoles and ClusterRoleBindings required
// for the central cluster.
func createCentralClusterServiceAccountAndRoles(ctx context.Context, c kubernetes.Interface, f flags) error {
	// central cluster always uses Roles. Never Cluster Roles.
	return createServiceAccountAndRoles(ctx, c, f.serviceAccount, f.centralClusterNamespace, f.clusterScoped, centralCluster)
}

// createServiceAccountAndRoles creates the ServiceAccount and Roles, RoleBindings, ClusterRoles and ClusterRoleBindings required.
func createServiceAccountAndRoles(ctx context.Context, c kubernetes.Interface, serviceAccountName, namespace string, clusterScoped bool, clusterType clusterType) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "image-registries-secret"},
		},
	}

	_, err := c.CoreV1().ServiceAccounts(sa.Namespace).Create(ctx, &sa, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating service account: %w", err)
	}

	if !clusterScoped {
		var role rbacv1.Role
		if clusterType == centralCluster {
			role = buildCentralEntityRole(sa.Namespace)
		} else {
			role = buildMemberEntityRole(sa.Namespace)
		}

		_, err = c.RbacV1().Roles(sa.Namespace).Create(ctx, &role, metav1.CreateOptions{})
		if !errors.IsAlreadyExists(err) && err != nil {
			return xerrors.Errorf("error creating role: %w", err)
		}

		roleBinding := buildRoleBinding(role, sa.Name)
		_, err = c.RbacV1().RoleBindings(sa.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})
		if !errors.IsAlreadyExists(err) && err != nil {
			return xerrors.Errorf("error creating role binding: %w", err)
		}
		return nil
	}

	var clusterRole rbacv1.ClusterRole
	if clusterType == centralCluster {
		clusterRole = buildCentralEntityClusterRole()
	} else {
		clusterRole = buildMemberEntityClusterRole()
	}

	_, err = c.RbacV1().ClusterRoles().Create(ctx, &clusterRole, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating cluster role: %w", err)
	}
	fmt.Printf("created clusterrole: %s\n", clusterRole.Name)

	clusterRoleBinding := buildClusterRoleBinding(clusterRole, sa)
	_, err = c.RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating cluster role binding: %w", err)
	}
	fmt.Printf("created clusterrolebinding: %s\n", clusterRoleBinding.Name)
	return nil
}

// createServiceAccountsAndRoles creates the required ServiceAccounts in all member clusters.
func createServiceAccountsAndRoles(ctx context.Context, clientMap map[string]kubernetes.Interface, f flags) error {
	fmt.Printf("creating central cluster roles in cluster: %s\n", f.centralCluster)
	c := clientMap[f.centralCluster]
	if err := createCentralClusterServiceAccountAndRoles(ctx, c, f); err != nil {
		return err
	}
	if f.createServiceAccountSecrets {
		if err := createServiceAccountTokenSecret(ctx, c, f.centralClusterNamespace, f.serviceAccount); err != nil {
			return err
		}
	}

	for _, memberCluster := range f.memberClusters {
		if memberCluster == f.centralCluster {
			fmt.Printf("skipping creation of member roles in cluster (it is also the central cluster): %s\n", memberCluster)
			continue
		}
		fmt.Printf("creating member roles in cluster: %s\n", memberCluster)
		c := clientMap[memberCluster]
		if err := createMemberServiceAccountAndRoles(ctx, c, f); err != nil {
			return err
		}
		if f.createServiceAccountSecrets {
			if err := createServiceAccountTokenSecret(ctx, c, f.memberClusterNamespace, f.serviceAccount); err != nil {
				return err
			}
		}
	}

	return nil
}

func createServiceAccountTokenSecret(ctx context.Context, c kubernetes.Interface, namespace string, serviceAccountName string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-token-secret", serviceAccountName),
			Namespace: namespace,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": serviceAccountName,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	_, err := c.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("cannot create secret %+v: %w", *secret, err)
	}

	return nil
}

// createKubeConfigFromServiceAccountTokens builds up a KubeConfig from the ServiceAccount tokens provided.
func createKubeConfigFromServiceAccountTokens(serviceAccountTokens map[string]corev1.Secret, flags flags) (KubeConfigFile, error) {
	config := &KubeConfigFile{
		Kind:       "Config",
		ApiVersion: "v1",
	}

	for i, clusterName := range flags.memberClusters {
		tokenSecret := serviceAccountTokens[clusterName]
		ca, ok := tokenSecret.Data["ca.crt"]
		if !ok {
			return KubeConfigFile{}, xerrors.Errorf("key 'ca.crt' missing from token secret %s", tokenSecret.Name)
		}

		token, ok := tokenSecret.Data["token"]
		if !ok {
			return KubeConfigFile{}, xerrors.Errorf("key 'token' missing from token secret %s", tokenSecret.Name)
		}

		config.Clusters = append(config.Clusters, KubeConfigClusterItem{
			Name: clusterName,
			Cluster: KubeConfigCluster{
				CertificateAuthorityData: ca,
				Server:                   flags.memberClusterApiServerUrls[i],
			},
		})

		ns := flags.memberClusterNamespace
		if flags.clusterScoped {
			ns = ""
		}

		config.Contexts = append(config.Contexts, KubeConfigContextItem{
			Name: clusterName,
			Context: KubeConfigContext{
				Cluster:   clusterName,
				Namespace: ns,
				User:      clusterName,
			},
		})

		config.Users = append(config.Users, KubeConfigUserItem{
			Name: clusterName,
			User: KubeConfigUser{
				Token: string(token),
			},
		})
	}
	return *config, nil
}

// getAllWorkerClusterServiceAccountSecretTokens returns a slice of secrets that should all be
// copied in the central cluster for the operator to use.
func getAllWorkerClusterServiceAccountSecretTokens(ctx context.Context, clientSetMap map[string]kubernetes.Interface, flags flags) (map[string]corev1.Secret, error) {
	allSecrets := map[string]corev1.Secret{}

	for _, cluster := range flags.memberClusters {
		c := clientSetMap[cluster]
		sas, err := getServiceAccounts(ctx, c, flags.memberClusterNamespace)
		if err != nil {
			return nil, xerrors.Errorf("failed getting service accounts: %w", err)
		}

		for _, sa := range sas {
			if sa.Name == flags.serviceAccount {
				token, err := getServiceAccountToken(ctx, c, sa)
				if err != nil {
					return nil, xerrors.Errorf("failed getting service account token: %w", err)
				}
				allSecrets[cluster] = *token
			}
		}
	}
	return allSecrets, nil
}

func getServiceAccounts(ctx context.Context, lister kubernetes.Interface, namespace string) ([]corev1.ServiceAccount, error) {
	saList, err := lister.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})

	if err != nil {
		return nil, xerrors.Errorf("failed to list service accounts in member cluster namespace %s: %w", namespace, err)
	}
	return saList.Items, nil
}

// getServiceAccountToken returns the Secret containing the ServiceAccount token
func getServiceAccountToken(ctx context.Context, secretLister kubernetes.Interface, sa corev1.ServiceAccount) (*corev1.Secret, error) {
	secretList, err := secretLister.CoreV1().Secrets(sa.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, xerrors.Errorf("failed to list secrets in member cluster namespace %s: %w", sa.Namespace, err)
	}
	for _, secret := range secretList.Items {
		// found the associated service account token.
		if strings.HasPrefix(secret.Name, fmt.Sprintf("%s-token", sa.Name)) {
			return &secret, nil
		}
	}
	return nil, xerrors.Errorf("no service account token found for serviceaccount: %s", sa.Name)
}

// copySecret copies a Secret from a source cluster to a target cluster
func copySecret(ctx context.Context, src, dst kubernetes.Interface, namespace, name string) error {
	secret, err := src.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return xerrors.Errorf("failed retrieving secret: %s from source cluster: %w", name, err)
	}
	_, err = dst.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    secret.Labels,
		},
		Data: secret.Data,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return err
	}
	return nil
}

func createServiceAccount(ctx context.Context, c kubernetes.Interface, serviceAccountName, namespace string) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
	}

	_, err := c.CoreV1().ServiceAccounts(sa.Namespace).Create(ctx, &sa, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating service account: %w", err)
	}
	return nil
}

func createDatabaseRole(ctx context.Context, c kubernetes.Interface, roleName, namespace string) error {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"patch", "delete", "get"},
			},
		},
	}
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: appdbServiceAccount,
			},
		},
	}
	_, err := c.RbacV1().Roles(role.Namespace).Create(ctx, &role, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating role: %w", err)
	}

	_, err = c.RbacV1().RoleBindings(roleBinding.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating role binding: %w", err)
	}
	return nil
}

// createDatabaseRoles creates the default ServiceAccounts, Roles and RoleBindings required for running database
// instances in a member cluster.
func createDatabaseRoles(ctx context.Context, client kubernetes.Interface, f flags) error {
	if err := createServiceAccount(ctx, client, appdbServiceAccount, f.memberClusterNamespace); err != nil {
		return err
	}
	if err := createServiceAccount(ctx, client, databasePodsServiceAccount, f.memberClusterNamespace); err != nil {
		return err
	}
	if err := createServiceAccount(ctx, client, opsManagerServiceAccount, f.memberClusterNamespace); err != nil {
		return err
	}
	if err := createDatabaseRole(ctx, client, appdbRole, f.memberClusterNamespace); err != nil {
		return err
	}
	return nil
}

// copyDatabaseRoles copies the ServiceAccounts, Roles and RoleBindings required for running database instances
// in a member cluster. This is used for adding new member clusters by copying over the configuration of a healthy
// source cluster.
func copyDatabaseRoles(ctx context.Context, src, dst kubernetes.Interface, namespace string) error {
	appdbSA, err := src.CoreV1().ServiceAccounts(namespace).Get(ctx, appdbServiceAccount, metav1.GetOptions{})
	if err != nil {
		return xerrors.Errorf("failed retrieving service account %s from source cluster: %w", appdbServiceAccount, err)
	}
	dbpodsSA, err := src.CoreV1().ServiceAccounts(namespace).Get(ctx, databasePodsServiceAccount, metav1.GetOptions{})
	if err != nil {
		return xerrors.Errorf("failed retrieving service account %s from source cluster: %w", databasePodsServiceAccount, err)
	}
	opsManagerSA, err := src.CoreV1().ServiceAccounts(namespace).Get(ctx, opsManagerServiceAccount, metav1.GetOptions{})
	if err != nil {
		return xerrors.Errorf("failed retrieving service account %s from source cluster: %w", opsManagerServiceAccount, err)
	}
	appdbR, err := src.RbacV1().Roles(namespace).Get(ctx, appdbRole, metav1.GetOptions{})
	if err != nil {
		return xerrors.Errorf("failed retrieving role %s from source cluster: %w", appdbRole, err)
	}
	appdbRB, err := src.RbacV1().RoleBindings(namespace).Get(ctx, appdbRoleBinding, metav1.GetOptions{})
	if err != nil {
		return xerrors.Errorf("failed retrieving role binding %s from source cluster: %w", appdbRoleBinding, err)
	}
	if len(appdbSA.ImagePullSecrets) > 0 {
		if err := copySecret(ctx, src, dst, namespace, appdbSA.ImagePullSecrets[0].Name); err != nil {
			fmt.Printf("failed creating image pull secret %s: %s\n", appdbSA.ImagePullSecrets[0].Name, err)
		}

	}
	if len(dbpodsSA.ImagePullSecrets) > 0 {
		if err := copySecret(ctx, src, dst, namespace, dbpodsSA.ImagePullSecrets[0].Name); err != nil {
			fmt.Printf("failed creating image pull secret %s: %s\n", dbpodsSA.ImagePullSecrets[0].Name, err)
		}
	}
	if len(opsManagerSA.ImagePullSecrets) > 0 {
		if err := copySecret(ctx, src, dst, namespace, opsManagerSA.ImagePullSecrets[0].Name); err != nil {
			fmt.Printf("failed creating image pull secret %s: %s\n", opsManagerSA.ImagePullSecrets[0].Name, err)
		}
	}
	_, err = dst.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appdbSA.Name,
			Labels: appdbSA.Labels,
		},
		ImagePullSecrets: appdbSA.DeepCopy().ImagePullSecrets,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating service account: %w", err)
	}
	_, err = dst.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   dbpodsSA.Name,
			Labels: dbpodsSA.Labels,
		},
		ImagePullSecrets: dbpodsSA.DeepCopy().ImagePullSecrets,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating service account: %w", err)

	}
	_, err = dst.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   opsManagerSA.Name,
			Labels: opsManagerSA.Labels,
		},
		ImagePullSecrets: opsManagerSA.DeepCopy().ImagePullSecrets,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating service account: %w", err)
	}

	_, err = dst.RbacV1().Roles(namespace).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appdbR.Name,
			Labels: appdbR.Labels,
		},
		Rules: appdbR.DeepCopy().Rules,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating role: %w", err)
	}
	_, err = dst.RbacV1().RoleBindings(namespace).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appdbRB.Name,
			Labels: appdbRB.Labels,
		},
		Subjects: appdbRB.DeepCopy().Subjects,
		RoleRef:  appdbRB.DeepCopy().RoleRef,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return xerrors.Errorf("error creating role binding: %w", err)
	}

	return nil
}

func installDatabaseRoles(ctx context.Context, clientSet map[string]kubernetes.Interface, f flags) error {
	for _, clusterName := range f.memberClusters {
		if err := createDatabaseRoles(ctx, clientSet[clusterName], f); err != nil {
			return err
		}
	}

	return nil
}

// setupDatabaseRoles installs the required database roles in the member clusters.
// The flags passed to the CLI must contain a healthy source member cluster which will be treated as
// the source of truth for all the member clusters.
func setupDatabaseRoles(ctx context.Context, clientSet map[string]kubernetes.Interface, f flags) error {
	for _, clusterName := range f.memberClusters {
		if clusterName != f.sourceCluster {
			if err := copyDatabaseRoles(ctx, clientSet[f.sourceCluster], clientSet[clusterName], f.memberClusterNamespace); err != nil {
				return err
			}
		}
	}

	return nil
}

// replaceClusterMembersConfigMap creates the configmap used by the operator to know which clusters are members of the multi-cluster setup.
// This will replace the existing configmap.
// NOTE: the configmap is hardcoded to be defaultOperatorConfigMapName
func replaceClusterMembersConfigMap(ctx context.Context, centralClusterClient kubernetes.Interface, flags flags) error {
	members := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultOperatorConfigMapName,
			Namespace: flags.centralClusterNamespace,
			Labels:    multiClusterLabels(),
		},
		Data: map[string]string{},
	}

	addToSet(flags.memberClusters, &members)

	fmt.Printf("Creating Member list Configmap %s/%s in cluster %s\n", flags.centralClusterNamespace, defaultOperatorConfigMapName, flags.centralCluster)
	_, err := centralClusterClient.CoreV1().ConfigMaps(flags.centralClusterNamespace).Create(ctx, &members, metav1.CreateOptions{})

	if err != nil && !errors.IsAlreadyExists(err) {
		return xerrors.Errorf("failed creating secret: %w", err)
	}

	if errors.IsAlreadyExists(err) {
		if _, err := centralClusterClient.CoreV1().ConfigMaps(flags.centralClusterNamespace).Update(ctx, &members, metav1.UpdateOptions{}); err != nil {
			return xerrors.Errorf("error creating configmap: %w", err)
		}
	}

	return nil
}

func addToSet(memberClusters []string, into *corev1.ConfigMap) {
	// override or add
	for _, memberCluster := range memberClusters {
		into.Data[memberCluster] = ""
	}
}
