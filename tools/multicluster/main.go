package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
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
// Service Accounts, Roles and RoleBindings are created in all of the member clusters and the central cluster.
// The Service Account token secrets from the member clusters are merged into a KubeConfig file which is then
// created in the central cluster.

const (
	kubeConfigEnv              = "KUBECONFIG"
	centralCluster clusterType = "CENTRAL"
	memberCluster  clusterType = "MEMBER"
)

// flags holds all of the fields provided by the user.
type flags struct {
	memberClusters             []string
	memberClusterApiServerUrls []string
	serviceAccount             string
	centralCluster             string
	memberClusterNamespace     string
	centralClusterNamespace    string
	cleanup                    bool
	clusterScoped              bool
	installDatabaseRoles       bool
	operatorName               string
	sourceCluster              string
}

const (
	kubeConfigSecretName       = "mongodb-enterprise-operator-multi-cluster-kubeconfig"
	kubeConfigSecretKey        = "kubeconfig"
	appdbServiceAccount        = "mongodb-enterprise-appdb"
	databasePodsServiceAccount = "mongodb-enterprise-database-pods"
	opsManagerServiceAccount   = "mongodb-enterprise-ops-manager"
	appdbRole                  = "mongodb-enterprise-appdb"
	appdbRoleBinding           = "mongodb-enterprise-appdb"
	defaultOperatorName        = "mongodb-enterprise-operator"
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// parseFlags returns a struct containing all of the flags provided by the user.
func parseSetupFlags() (flags, error) {
	var memberClusters string
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	flags := flags{}
	setupCmd.StringVar(&memberClusters, "member-clusters", "", "Comma separated list of member clusters. [required]")
	setupCmd.StringVar(&flags.serviceAccount, "service-account", "mongodb-enterprise-operator-multi-cluster", "Name of the service account which should be used for the Operator to communicate with the member clusters. [optional, default: mongodb-enterprise-operator-multi-cluster]")
	setupCmd.StringVar(&flags.centralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [required]")
	setupCmd.StringVar(&flags.memberClusterNamespace, "member-cluster-namespace", "", "The namespace the member cluster resources will be deployed to. [required]")
	setupCmd.StringVar(&flags.centralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [required]")
	setupCmd.BoolVar(&flags.cleanup, "cleanup", false, "Delete all previously created resources except for namespaces. [optional default: false]")
	setupCmd.BoolVar(&flags.clusterScoped, "cluster-scoped", false, "Create ClusterRole and ClusterRoleBindings for member clusters. [optional default: false]")
	setupCmd.BoolVar(&flags.installDatabaseRoles, "install-database-roles", false, "Install the ServiceAccounts and Roles required for running database workloads in the member clusters. [optional default: false]")
	setupCmd.Parse(os.Args[2:])
	if anyAreEmpty(memberClusters, flags.serviceAccount, flags.centralCluster, flags.memberClusterNamespace, flags.centralClusterNamespace) {
		return flags, fmt.Errorf("non empty values are required for [service-account, member-clusters, central-cluster, member-cluster-namespace, central-cluster-namespace]")
	}

	flags.memberClusters = strings.Split(memberClusters, ",")

	configFilePath := loadKubeConfigFilePath()
	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return flags, fmt.Errorf("error loading kubeconfig file '%s': %s", configFilePath, err)
	}
	if flags.memberClusterApiServerUrls, err = getMemberClusterApiServerUrls(kubeconfig, flags.memberClusters); err != nil {
		return flags, err
	}

	return flags, nil
}

func parseRecoverFlags() (flags, error) {
	var memberClusters string
	recoverCmd := flag.NewFlagSet("recover", flag.ExitOnError)
	flags := flags{}
	recoverCmd.StringVar(&memberClusters, "member-clusters", "", "Comma separated list of member clusters. [required]")
	recoverCmd.StringVar(&flags.serviceAccount, "service-account", "mongodb-enterprise-operator-multi-cluster", "Name of the service account which should be used for the Operator to communicate with the member clusters. [optional, default: mongodb-enterprise-operator-multi-cluster]")
	recoverCmd.StringVar(&flags.centralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [required]")
	recoverCmd.StringVar(&flags.memberClusterNamespace, "member-cluster-namespace", "", "The namespace the member cluster resources will be deployed to. [required]")
	recoverCmd.StringVar(&flags.centralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [required]")
	recoverCmd.BoolVar(&flags.cleanup, "cleanup", false, "Delete all previously created resources except for namespaces. [optional default: false]")
	recoverCmd.BoolVar(&flags.clusterScoped, "cluster-scoped", false, "Create ClusterRole and ClusterRoleBindings for member clusters. [optional default: false]")
	recoverCmd.StringVar(&flags.operatorName, "operator-name", defaultOperatorName, "Name used to identify the deployment of the operator. [optional, default: mongodb-enterprise-operator]")
	recoverCmd.BoolVar(&flags.installDatabaseRoles, "install-database-roles", false, "Install the ServiceAccounts and Roles required for running database workloads in the member clusters. [optional default: false]")
	recoverCmd.StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster for recovery. This has to be one of the healthy member cluster that is the source of truth for new cluster configuration. [required]")
	recoverCmd.Parse(os.Args[2:])
	if anyAreEmpty(memberClusters, flags.serviceAccount, flags.centralCluster, flags.memberClusterNamespace, flags.centralClusterNamespace, flags.sourceCluster) {
		return flags, fmt.Errorf("non empty values are required for [service-account, member-clusters, central-cluster, member-cluster-namespace, central-cluster-namespace, source-cluster]")
	}

	flags.memberClusters = strings.Split(memberClusters, ",")
	if !contains(flags.memberClusters, flags.sourceCluster) {
		return flags, fmt.Errorf("source-cluster has to be one of the healthy member clusters: %s", memberClusters)
	}

	configFilePath := loadKubeConfigFilePath()
	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return flags, fmt.Errorf("error loading kubeconfig file '%s': %s", configFilePath, err)
	}
	if flags.memberClusterApiServerUrls, err = getMemberClusterApiServerUrls(kubeconfig, flags.memberClusters); err != nil {
		return flags, err
	}
	return flags, nil
}

// getMemberClusterApiServerUrls returns the slice of member cluster api urls that should be used.
func getMemberClusterApiServerUrls(kubeconfig *clientcmdapi.Config, clusterNames []string) ([]string, error) {
	var urls []string
	for _, name := range clusterNames {
		if cluster := kubeconfig.Clusters[name]; cluster != nil {
			urls = append(urls, cluster.Server)
		} else {
			return nil, fmt.Errorf("cluster '%s' not found in kubeconfig", name)
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'setup' or 'recover' subcommands")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "setup":
		flags, err := parseSetupFlags()
		if err != nil {
			fmt.Printf("error parsing flags: %s\n", err)

			os.Exit(1)
		}
		if err := ensureMultiClusterResources(flags, getKubernetesClient); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "recover":
		flags, err := parseRecoverFlags()
		if err != nil {
			fmt.Printf("error parsing flags: %s\n", err)

			os.Exit(1)
		}
		if err := ensureMultiClusterResources(flags, getKubernetesClient); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		clientMap, err := createClientMap(flags.memberClusters, flags.centralCluster, loadKubeConfigFilePath(), getKubernetesClient)
		if err != nil {
			fmt.Printf("failed to create clientset map: %s\n", err)
			os.Exit(1)
		}

		patchOperatorDeployment(clientMap, flags)
		fmt.Println("Patched operator to use new member clusters.")
	default:
		fmt.Println("expected 'setup' or 'recover' subcommands")
		os.Exit(1)
	}

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

// createClientMap crates a map of all MultiClusterClient for every member cluster, and the operator cluster.
func createClientMap(memberClusters []string, operatorCluster, kubeConfigPath string, getClient func(clusterName string, kubeConfigPath string) (kubernetes.Interface, error)) (map[string]kubernetes.Interface, error) {
	clientMap := map[string]kubernetes.Interface{}
	for _, c := range memberClusters {
		clientset, err := getClient(c, kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset map: %s", err)
		}
		clientMap[c] = clientset
	}

	clientset, err := getClient(operatorCluster, kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset map: %s", err)
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
		return nil, fmt.Errorf("failed to create client config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %s", err)
	}

	return clientset, nil
}

// performCleanup cleans up all of the resources that were created by this script in the past.
func performCleanup(clientMap map[string]kubernetes.Interface, flags flags) error {
	for _, cluster := range flags.memberClusters {
		c := clientMap[cluster]
		if err := cleanupClusterResources(c, cluster, flags.memberClusterNamespace); err != nil {
			return fmt.Errorf("failed cleaning up cluster %s namespace %s: %s", cluster, flags.memberClusterNamespace, err)
		}
	}
	c := clientMap[flags.centralCluster]
	if err := cleanupClusterResources(c, flags.centralCluster, flags.centralClusterNamespace); err != nil {
		return fmt.Errorf("failed cleaning up cluster %s namespace %s: %s", flags.centralCluster, flags.centralClusterNamespace, err)
	}
	return nil
}

// cleanupClusterResources cleans up all the resources created by this tool in a given namespace.
func cleanupClusterResources(clientset kubernetes.Interface, clusterName, namespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	errorChan := make(chan error)
	done := make(chan struct{})

	listOpts := metav1.ListOptions{
		LabelSelector: "multi-cluster=true",
	}

	go func() {
		// clean up secrets
		secretList, err := clientset.CoreV1().Secrets(namespace).List(ctx, listOpts)

		if err != nil {
			errorChan <- err
			return
		}

		if secretList != nil {
			for _, s := range secretList.Items {
				fmt.Printf("Deleting Secret: %s in cluster %s\n", s.Name, clusterName)
				if err := clientset.CoreV1().Secrets(namespace).Delete(ctx, s.Name, metav1.DeleteOptions{}); err != nil {
					errorChan <- err
					return
				}
			}
		}

		// clean up service accounts
		serviceAccountList, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, listOpts)

		if err != nil {
			errorChan <- err
			return
		}

		if serviceAccountList != nil {
			for _, sa := range serviceAccountList.Items {
				fmt.Printf("Deleting ServiceAccount: %s in cluster %s\n", sa.Name, clusterName)
				if err := clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, sa.Name, metav1.DeleteOptions{}); err != nil {
					errorChan <- err
					return
				}
			}
		}

		// clean up roles
		roleList, err := clientset.RbacV1().Roles(namespace).List(ctx, listOpts)
		if err != nil {
			errorChan <- err
			return
		}

		for _, r := range roleList.Items {
			fmt.Printf("Deleting Role: %s in cluster %s\n", r.Name, clusterName)
			if err := clientset.RbacV1().Roles(namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
				errorChan <- err
				return
			}
		}

		// clean up roles
		roles, err := clientset.RbacV1().Roles(namespace).List(ctx, listOpts)
		if err != nil {
			errorChan <- err
			return
		}

		if roles != nil {
			for _, r := range roles.Items {
				fmt.Printf("Deleting Role: %s in cluster %s\n", r.Name, clusterName)
				if err := clientset.RbacV1().Roles(namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); err != nil {
					errorChan <- err
					return
				}
			}
		}

		// clean up role bindings
		roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, listOpts)
		if !errors.IsNotFound(err) && err != nil {
			errorChan <- err
			return
		}

		if roleBindings != nil {
			for _, crb := range roleBindings.Items {
				fmt.Printf("Deleting RoleBinding: %s in cluster %s\n", crb.Name, clusterName)
				if err := clientset.RbacV1().RoleBindings(namespace).Delete(ctx, crb.Name, metav1.DeleteOptions{}); err != nil {
					errorChan <- err
					return
				}
			}
		}

		// clean up cluster role bindings
		clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, listOpts)
		if !errors.IsNotFound(err) && err != nil {
			errorChan <- err
			return
		}

		if clusterRoleBindings != nil {
			for _, crb := range clusterRoleBindings.Items {
				fmt.Printf("Deleting ClusterRoleBinding: %s in cluster %s\n", crb.Name, clusterName)
				if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{}); err != nil {
					errorChan <- err
					return
				}
			}
		}

		// clean up cluster roles
		clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, listOpts)
		if !errors.IsNotFound(err) && err != nil {
			errorChan <- err
			return
		}

		if clusterRoles != nil {
			for _, cr := range clusterRoles.Items {
				fmt.Printf("Deleting ClusterRole: %s in cluster %s\n", cr.Name, clusterName)
				if err := clientset.RbacV1().ClusterRoles().Delete(ctx, cr.Name, metav1.DeleteOptions{}); err != nil {
					errorChan <- err
					return
				}
			}
		}

		done <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case <-done:
		return nil
	}

}

// ensureNamespace creates the namespace with the given clientset. If an error occurs, it is sent to the given error channel.
func ensureNamespace(ctx context.Context, clientSet kubernetes.Interface, nsName string, errorChan chan error) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: multiClusterLabels(),
		},
	}
	_, err := clientSet.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		errorChan <- fmt.Errorf("failed to create namespace %s: %s", ns.Name, err)
	}
}

// ensureAllClusterNamespacesExist makes sure the namespace we will be creating exists in all clusters.
func ensureAllClusterNamespacesExist(clientSets map[string]kubernetes.Interface, f flags) error {
	totalClusters := len(f.memberClusters) + 1
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(totalClusters*2)*time.Second)
	defer cancel()
	done := make(chan struct{})
	errorChan := make(chan error)

	go func() {
		for _, clusterName := range f.memberClusters {
			ensureNamespace(ctx, clientSets[clusterName], f.memberClusterNamespace, errorChan)
		}
		ensureNamespace(ctx, clientSets[f.centralCluster], f.centralClusterNamespace, errorChan)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case <-done:
		return nil
	}
}

// ensureMultiClusterResources copies the ServiceAccount Secret tokens from the specified
// member clusters, merges them into a KubeConfig file and creates a Secret in the central cluster
// with the contents.
func ensureMultiClusterResources(flags flags, getClient func(clusterName, kubeConfigPath string) (kubernetes.Interface, error)) error {
	clientMap, err := createClientMap(flags.memberClusters, flags.centralCluster, loadKubeConfigFilePath(), getClient)
	if err != nil {
		return fmt.Errorf("failed to create clientset map: %s", err)
	}

	if flags.cleanup {
		if err := performCleanup(clientMap, flags); err != nil {
			return fmt.Errorf("failed performing cleanup of resources: %s", err)
		}
	}

	if err := ensureAllClusterNamespacesExist(clientMap, flags); err != nil {
		return fmt.Errorf("failed ensuring namespaces: %s", err)
	}
	fmt.Println("Ensured namespaces exist in all clusters.")

	if err := createServiceAccountsAndRoles(clientMap, flags); err != nil {
		return fmt.Errorf("failed creating service accounts and roles in all clusters: %s", err)
	}
	fmt.Println("Ensured ServiceAccounts and Roles.")

	secrets, err := getAllWorkerClusterServiceAccountSecretTokens(clientMap, flags)
	if err != nil {
		return fmt.Errorf("failed to get service account secret tokens: %s", err)
	}

	if len(secrets) != len(flags.memberClusters) {
		return fmt.Errorf("required %d serviceaccount tokens but found only %d\n", len(flags.memberClusters), len(secrets))
	}

	kubeConfig, err := createKubeConfigFromServiceAccountTokens(secrets, flags)
	if err != nil {
		return fmt.Errorf("failed to create kube config from service account tokens: %s", err)
	}

	kubeConfigBytes, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig: %s", err)
	}

	centralClusterClient, err := getClient(flags.centralCluster, loadKubeConfigFilePath())
	if err != nil {
		return fmt.Errorf("failed to get central cluster clientset: %s", err)
	}

	if err := createKubeConfigSecret(centralClusterClient, kubeConfigBytes, flags); err != nil {
		return fmt.Errorf("failed creating KubeConfig secret: %s", err)
	}

	if flags.sourceCluster != "" {
		if err := setupDatabaseRoles(clientMap, flags); err != nil {
			return fmt.Errorf("failed setting up database roles: %s", err)
		}
		fmt.Println("Ensured database Roles in member clusters.")
	} else if flags.installDatabaseRoles {
		if err := installDatabaseRoles(clientMap, flags); err != nil {
			return fmt.Errorf("failed installing database roles: %s", err)
		}
		fmt.Println("Ensured database Roles in member clusters.")
	}

	return nil
}

// createKubeConfigSecret creates the secret containing the KubeConfig file made from the various
// service account tokens in the member clusters.
func createKubeConfigSecret(centralClusterClient kubernetes.Interface, kubeConfigBytes []byte, flags flags) error {
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

	done := make(chan struct{})
	errorChan := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		fmt.Printf("Creating KubeConfig secret %s/%s in cluster %s\n", flags.centralClusterNamespace, kubeConfigSecret.Name, flags.centralCluster)
		_, err := centralClusterClient.CoreV1().Secrets(flags.centralClusterNamespace).Create(ctx, &kubeConfigSecret, metav1.CreateOptions{})

		if !errors.IsAlreadyExists(err) && err != nil {
			errorChan <- fmt.Errorf("failed creating secret: %s", err)
			return
		}

		if errors.IsAlreadyExists(err) {
			_, err = centralClusterClient.CoreV1().Secrets(flags.centralClusterNamespace).Update(ctx, &kubeConfigSecret, metav1.UpdateOptions{})
			if err != nil {
				errorChan <- fmt.Errorf("failed updating existing secret: %s", err)
				return
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case <-done:
		return nil
	}
}

func getCentralRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			Verbs: []string{"*"},
			Resources: []string{"mongodbmulti", "mongodbmulti/finalizers", "mongousers",
				"opsmanagers", "opsmanagers/finalizers",
				"mongodb", "mongodb/finalizers"},
			APIGroups: []string{"mongodb.com"},
		},
	}
}

func buildCentralEntityRole(namespace string) rbacv1.Role {
	return rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mongodb-enterprise-operator-multi-role",
			Namespace: namespace,
			Labels:    multiClusterLabels(),
		},
		Rules: getCentralRules(),
	}
}

func buildCentralEntityClusterRole() rbacv1.ClusterRole {
	rules := append(getCentralRules(), rbacv1.PolicyRule{
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
	}

	_, err := c.CoreV1().ServiceAccounts(sa.Namespace).Create(ctx, &sa, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("error creating service account: %s", err)
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
			return fmt.Errorf("error creating role: %s", err)
		}

		roleBinding := buildRoleBinding(role, sa.Name)
		_, err = c.RbacV1().RoleBindings(sa.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})
		if !errors.IsAlreadyExists(err) && err != nil {
			return fmt.Errorf("error creating role binding: %s", err)
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
		return fmt.Errorf("error creating cluster role: %s", err)
	}

	clusterRoleBinding := buildClusterRoleBinding(clusterRole, sa)
	_, err = c.RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("error creating cluster role binding: %s", err)
	}
	return nil
}

// createServiceAccountsAndRoles creates the required ServiceAccounts in all member clusters.
func createServiceAccountsAndRoles(clientMap map[string]kubernetes.Interface, f flags) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(f.memberClusters)*2)*time.Second)
	defer cancel()

	finishedChan := make(chan struct{})
	errorChan := make(chan error)
	go func() {
		for _, memberCluster := range f.memberClusters {
			c := clientMap[memberCluster]
			if err := createMemberServiceAccountAndRoles(ctx, c, f); err != nil {
				errorChan <- err
			}
		}
		c := clientMap[f.centralCluster]
		if err := createCentralClusterServiceAccountAndRoles(ctx, c, f); err != nil {
			errorChan <- err
			return
		}
		finishedChan <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case <-finishedChan:
		return nil
	}
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
			return KubeConfigFile{}, fmt.Errorf("key 'ca.crt' missing from token secret %s", tokenSecret.Name)
		}

		token, ok := tokenSecret.Data["token"]
		if !ok {
			return KubeConfigFile{}, fmt.Errorf("key 'token' missing from token secret %s", tokenSecret.Name)
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
func getAllWorkerClusterServiceAccountSecretTokens(clientSetMap map[string]kubernetes.Interface, flags flags) (map[string]corev1.Secret, error) {
	allSecrets := map[string]corev1.Secret{}

	for _, cluster := range flags.memberClusters {
		c := clientSetMap[cluster]
		sas, err := getServiceAccountsWithTimeout(c, flags.memberClusterNamespace)
		if err != nil {
			return nil, fmt.Errorf("failed getting service accounts: %s", err)
		}

		for _, sa := range sas {
			if sa.Name == flags.serviceAccount {
				token, err := getServiceAccountTokenWithTimeout(c, sa)
				if err != nil {
					return nil, fmt.Errorf("failed getting service account token: %s", err)
				}
				allSecrets[cluster] = token
			}
		}
	}
	return allSecrets, nil
}

// getServiceAccountsWithTimeout returns a slice of service accounts in the given memberClusterNamespace.
func getServiceAccountsWithTimeout(lister kubernetes.Interface, namespace string) ([]corev1.ServiceAccount, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	accounts := make(chan []corev1.ServiceAccount)
	errorChan := make(chan error)

	go getServiceAccounts(ctx, lister, namespace, accounts, errorChan)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errorChan:
		return nil, err
	case allAccounts := <-accounts:
		return allAccounts, nil
	}
}

func getServiceAccounts(ctx context.Context, lister kubernetes.Interface, namespace string, accounts chan []corev1.ServiceAccount, errorChan chan error) {
	saList, err := lister.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})

	if err != nil {
		errorChan <- fmt.Errorf("failed to list service accounts in member cluster namespace %s: %s", namespace, err)
		return
	}
	accounts <- saList.Items
}

// getServiceAccountTokenWithTimeout returns the Secret containing the ServiceAccount token.
func getServiceAccountTokenWithTimeout(secretLister kubernetes.Interface, sa corev1.ServiceAccount) (corev1.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	secretChan := make(chan corev1.Secret)
	errorChan := make(chan error)

	go getServiceAccountToken(ctx, secretLister, sa, secretChan, errorChan)

	select {
	case <-ctx.Done():
		return corev1.Secret{}, ctx.Err()
	case err := <-errorChan:
		return corev1.Secret{}, err
	case saToken := <-secretChan:
		return saToken, nil
	}
}

// getServiceAccountToken sends the Secret containing the ServiceAccount token to the provided channel.
func getServiceAccountToken(ctx context.Context, secretLister kubernetes.Interface, sa corev1.ServiceAccount, secretChan chan corev1.Secret, errorChan chan error) {
	secretList, err := secretLister.CoreV1().Secrets(sa.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("failed to list secrets in member cluster namespace %s: %s", sa.Namespace, err)
		return
	}
	for _, secret := range secretList.Items {
		// found the associated service account token.
		if strings.HasPrefix(secret.Name, fmt.Sprintf("%s-token", sa.Name)) {
			secretChan <- secret
			return
		}
	}
	errorChan <- fmt.Errorf("no service account token found for serviceaccount: %s", sa.Name)
}

// copySecret copies a Secret from a source cluster to a target cluster
func copySecret(ctx context.Context, src, dst kubernetes.Interface, namespace, name string) error {
	secret, err := src.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed retrieving secret: %s from source cluster: %s", name, err)
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
		return fmt.Errorf("error creating service account: %s", err)
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
		return fmt.Errorf("error creating role: %s", err)
	}

	_, err = c.RbacV1().RoleBindings(roleBinding.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("error creating role binding: %s", err)
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
func copyDatabaseRoles(ctx context.Context, src, dst kubernetes.Interface, namespace string, errorChan chan error) {
	appdbSA, err := src.CoreV1().ServiceAccounts(namespace).Get(ctx, appdbServiceAccount, metav1.GetOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("failed retrieving service account %s from source cluster: %s", appdbServiceAccount, err)
	}
	dbpodsSA, err := src.CoreV1().ServiceAccounts(namespace).Get(ctx, databasePodsServiceAccount, metav1.GetOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("failed retrieving service account %s from source cluster: %s", databasePodsServiceAccount, err)
	}
	opsManagerSA, err := src.CoreV1().ServiceAccounts(namespace).Get(ctx, opsManagerServiceAccount, metav1.GetOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("failed retrieving service account %s from source cluster: %s", opsManagerServiceAccount, err)
	}
	appdbR, err := src.RbacV1().Roles(namespace).Get(ctx, appdbRole, metav1.GetOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("failed retrieving role %s from source cluster: %s", appdbRole, err)
	}
	appdbRB, err := src.RbacV1().RoleBindings(namespace).Get(ctx, appdbRoleBinding, metav1.GetOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("failed retrieving role binding %s from source cluster: %s", appdbRoleBinding, err)
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
		errorChan <- fmt.Errorf("error creating service account: %s", err)
	}
	_, err = dst.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   dbpodsSA.Name,
			Labels: dbpodsSA.Labels,
		},
		ImagePullSecrets: dbpodsSA.DeepCopy().ImagePullSecrets,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		errorChan <- fmt.Errorf("error creating service account: %s", err)

	}
	_, err = dst.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   opsManagerSA.Name,
			Labels: opsManagerSA.Labels,
		},
		ImagePullSecrets: opsManagerSA.DeepCopy().ImagePullSecrets,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		errorChan <- fmt.Errorf("error creating service account: %s", err)
	}

	_, err = dst.RbacV1().Roles(namespace).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appdbR.Name,
			Labels: appdbR.Labels,
		},
		Rules: appdbR.DeepCopy().Rules,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		errorChan <- fmt.Errorf("error creating role: %s", err)
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
		errorChan <- fmt.Errorf("error creating role binding: %s", err)
	}
}

func installDatabaseRoles(clientSet map[string]kubernetes.Interface, f flags) error {
	totalClusters := len(f.memberClusters) + 1
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(totalClusters*2)*time.Second)
	defer cancel()

	done := make(chan struct{})
	errorChan := make(chan error)

	go func() {
		for _, clusterName := range f.memberClusters {
			if err := createDatabaseRoles(ctx, clientSet[clusterName], f); err != nil {
				errorChan <- err
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case <-done:
		return nil
	}
}

// setupDatabaseRoles installs the required database roles in the member clusters.
// The flags passed to the CLI must contain a healthy source member cluster which will be treated as
// the source of truth for all the member clusters.
func setupDatabaseRoles(clientSet map[string]kubernetes.Interface, f flags) error {
	totalClusters := len(f.memberClusters) + 1
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(totalClusters*2)*time.Second)
	defer cancel()

	done := make(chan struct{})
	errorChan := make(chan error)

	go func() {
		for _, clusterName := range f.memberClusters {
			if clusterName != f.sourceCluster {
				copyDatabaseRoles(ctx, clientSet[f.sourceCluster], clientSet[clusterName], f.memberClusterNamespace, errorChan)
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case <-done:
		return nil
	}
}

// patchOperatorDeployment updates the operator deployment with configurations required for
// dataplane recovery, currently this only includes the names of the member clusters.
func patchOperatorDeployment(clientMap map[string]kubernetes.Interface, flags flags) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	c := clientMap[flags.centralCluster]
	operator, err := c.AppsV1().Deployments(flags.centralClusterNamespace).Get(ctx, flags.operatorName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newArgs := []string{}

	for _, arg := range operator.Spec.Template.Spec.Containers[0].Args {
		if strings.HasPrefix(arg, "-cluster-names") {
			newArgs = append(newArgs, fmt.Sprintf("-cluster-names=%s", strings.Join(flags.memberClusters, ",")))
		} else {
			newArgs = append(newArgs, arg)
		}
	}
	operator.Spec.Template.Spec.Containers[0].Args = newArgs

	_, err = c.AppsV1().Deployments(flags.centralClusterNamespace).Update(ctx, operator, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
