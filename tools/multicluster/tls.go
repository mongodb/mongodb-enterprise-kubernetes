package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apiExt "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrlRuntime "sigs.k8s.io/controller-runtime/pkg/client"
)

type MemberCluster struct {
	name      string
	members   int
	namespace string
}

type CA struct {
	keyPath string
	crtPath string
}

type KubeClients struct {
	client            kubernetes.Interface
	apiExtClient      apiExtClient.ApiextensionsV1Client
	ctrlRuntimeClient ctrlRuntime.Client
}

// This tool handles the creation of a self-signed CA (unless it's provided as input args),
// TLS certificates and configures the MongoDBMulti CR to use these certificates

const (
	kubeConfigEnv      = "KUBECONFIG"
	outputFolder       = "./certs"
	mongoDBMultiKind   = "MongoDBMulti"
	caConfigMapName    = "issuer-ca"
	clusterCertsPrefix = "clustercert"
)

// flags holds all of the fields provided by the user.
type flags struct {
	resourceName            string
	centralCluster          string
	centralClusterNamespace string
	caKeyPath               string
	caCrtPath               string
	caCommonName            string
	csrCountry              string
	csrState                string
	csrOrganization         string
}

// parseFlags returns a struct containing all of the flags provided by the user.
func parseFlags() (flags, error) {
	flags := flags{}
	flag.StringVar(&flags.resourceName, "resource-name", "", "MongoDB resource name. [required]")
	flag.StringVar(&flags.centralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [required]")
	flag.StringVar(&flags.centralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [required]")

	flag.StringVar(&flags.caKeyPath, "ca-key", "", "Path to existing CA root key. If not provided a self-signed CA will be created.")
	flag.StringVar(&flags.caCrtPath, "ca-crt", "", "Path to existing CA root cert. If not provided a self-signed CA will be created.")
	flag.StringVar(&flags.caCommonName, "common-name", "", "Common name used to generate the self-signed CA (required if [ca-key, ca-crt] are not provided)")
	flag.StringVar(&flags.csrCountry, "country", "", "Country used in the subject of the cluster signing (CSR). [required]")
	flag.StringVar(&flags.csrState, "state", "", "State used in the subject of the cluster signing (CSR). [required]")
	flag.StringVar(&flags.csrOrganization, "organization", "", "Organization used in the subject of the cluster signing (CSR). [required]")

	flag.Parse()

	if anyAreEmpty(flags.resourceName, flags.centralCluster, flags.centralClusterNamespace, flags.csrCountry, flags.csrState, flags.csrOrganization) {
		return flags, fmt.Errorf("non empty values are required for [resource-name, central-cluster, central-cluster-namespace, country, state, organization]")
	}

	if flags.caKeyPath == "" && flags.caCommonName == "" {
		return flags, fmt.Errorf("when a self-signed CA has to be created, common-name is required")
	}

	return flags, nil
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

// loadKubeConfigFilePath returns the path of the local KubeConfig file.
func loadKubeConfigFilePath() string {
	env := os.Getenv(kubeConfigEnv)
	if env != "" {
		return env
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

func execCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	fmt.Printf(" cmd: [ %s %s ]\n", name, strings.Join(arg, " "))
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func createCA(flags flags) (*CA, error) {
	if flags.caCrtPath != "" {
		return &CA{keyPath: flags.caKeyPath, crtPath: flags.caCrtPath}, nil
	}

	fmt.Println("1. generate CA")

	caKeyPath := fmt.Sprintf("%s/ca.key", outputFolder)
	caCrtPath := fmt.Sprintf("%s/ca.crt", outputFolder)

	if output, err := execCommand("openssl",
		"genrsa",
		"-out", caKeyPath,
		"2048"); err != nil {
		return nil, fmt.Errorf("%s, %s", err, output)
	}

	if output, err := execCommand("openssl",
		"req", "-x509", "-new", "-nodes",
		"-key", caKeyPath,
		"-subj", fmt.Sprintf("/CN=%s", flags.caCommonName),
		"-days", "3650",
		"-reqexts", "v3_req",
		"-extensions", "v3_ca",
		"-out", caCrtPath); err != nil {
		fmt.Printf("error creating self-signed Certificate Authority: %s, %s", err, output)

		return nil, fmt.Errorf("%s, %s", err, output)
	}

	fmt.Printf(" - root key: %s\n   root cert: %s\n", caKeyPath, caCrtPath)

	return &CA{keyPath: caKeyPath, crtPath: caCrtPath}, nil
}

func createClusterCSRs(memberClusters []MemberCluster, flags flags) error {
	fmt.Println("\n2. generate cluster CSRs")
	for _, mc := range memberClusters {
		clusterCertKey := fmt.Sprintf("%s/%s/cluster-cert-key.key", outputFolder, mc.name)
		clusterCSR := fmt.Sprintf("%s/%s/cluster-cert-signing.csr", outputFolder, mc.name)

		if output, err := execCommand("openssl",
			"genrsa",
			"-out", clusterCertKey,
			"2048"); err != nil {
			return fmt.Errorf("%s, %s", err, output)
		}

		if output, err := execCommand("openssl",
			"req", "-new", "-sha256",
			"-key", clusterCertKey,
			"-subj", fmt.Sprintf("/C=%s/ST=%s/O=%s", flags.csrCountry, flags.csrState, flags.csrOrganization),
			"-out", clusterCSR); err != nil {
			return fmt.Errorf("%s, %s", err, output)
		}
		fmt.Printf(" - cluster: %s\n   certKey: %s\n   CSR: %s\n", mc.name, clusterCertKey, clusterCSR)
	}
	return nil
}

func createClusterServiceCerts(ca CA, flags flags, resourceName string, clusters []MemberCluster) (map[string][]string, error) {
	fmt.Println("\n3. generate server certificates")

	clusterServiceCerts := map[string][]string{}

	for clusterIdx, cluster := range clusters {
		clusterCSR := fmt.Sprintf("%s/%s/cluster-cert-signing.csr", outputFolder, cluster.name)
		clusterServiceCerts[cluster.name] = []string{}

		for podIdx := 0; podIdx < cluster.members; podIdx++ {
			podName := fmt.Sprintf("%s-%d-%d", resourceName, clusterIdx, podIdx)
			podDNS := fmt.Sprintf("%s-svc.%s.svc.cluster.local", podName, cluster.namespace)
			podCert := fmt.Sprintf("%s/%s/%s.crt", outputFolder, cluster.name, podName)

			cmd := fmt.Sprintf("openssl x509 -req -extfile <(printf \"subjectAltName=DNS:%s\") -days 365 -in %s -CA %s -CAkey %s -CAcreateserial -out %s",
				podDNS, clusterCSR, ca.crtPath, ca.keyPath, podCert)

			if output, err := execCommand("bash", "-c", cmd); err != nil {
				return clusterServiceCerts, fmt.Errorf("%s, %s", err, output)
			}

			fmt.Printf(" - cluster: %s\n   pod: %s\n   cert: %s\n", cluster.name, podName, podCert)
			clusterServiceCerts[cluster.name] = append(clusterServiceCerts[cluster.name], podCert)
		}
	}
	return clusterServiceCerts, nil
}

func createSecret(client kubernetes.Interface, namespace string, name string, data map[string][]byte) error {
	kubeConfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.CoreV1().Secrets(namespace).Create(ctx, &kubeConfigSecret, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("failed creating secret: %s", err)
	}

	if errors.IsAlreadyExists(err) {
		_, err = client.CoreV1().Secrets(namespace).Update(ctx, &kubeConfigSecret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed updating existing secret: %s", err)
		}
	}
	return nil
}

func createConfigMap(client kubernetes.Interface, namespace string, name string, data map[string]string) error {
	kubeConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, &kubeConfigMap, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("failed creating config map: %s", err)
	}

	if errors.IsAlreadyExists(err) {
		_, err = client.CoreV1().ConfigMaps(namespace).Update(ctx, &kubeConfigMap, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed updating existing config map: %s", err)
		}
	}
	return nil
}

// getKubernetesClient returns a kubernetes.Clientset using the given context.
func getKubernetesClientSet(context string) (*KubeClients, error) {
	kubeConfigPath := loadKubeConfigFilePath()

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to create client config: %s", err)
	}

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %s", err)
	}

	clientsetApiEx, err := apiExtClient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes APIExt client: %s", err)
	}

	crClient, err := ctrlRuntime.New(config, ctrlRuntime.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes APIExt client: %s", err)
	}

	return &KubeClients{client: clientset, apiExtClient: *clientsetApiEx, ctrlRuntimeClient: crClient}, nil
}

func createCAConfigMaps(clusterClients map[string]KubeClients, ca CA, memberClusters []MemberCluster) error {
	fmt.Println("\n4. create CA config map")

	caCrt, err := os.ReadFile(ca.crtPath)
	if err != nil {
		return err
	}
	data := map[string]string{"ca-pem": string(caCrt), "mms-ca.crt": string(caCrt)}

	for _, mc := range memberClusters {
		client := clusterClients[mc.name].client
		if err := createConfigMap(client, mc.namespace, caConfigMapName, data); err != nil {
			return fmt.Errorf("error creating CA config map in cluster %s: %s", mc.name, err)
		}
		fmt.Printf(" - cluster: %s, config map: %s\n", mc.name, caConfigMapName)
	}

	return nil
}

func filenameWithoutExtension(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

func createClusterCertSecrets(clusterClients map[string]KubeClients, resourceName string, clusterServiceCerts map[string][]string, memberClusters []MemberCluster) error {
	fmt.Println("\n5. create cluster cert secrets")

	name := fmt.Sprintf("%s-%s-cert", clusterCertsPrefix, resourceName)
	data := map[string][]byte{}
	for cluster, clusterCerts := range clusterServiceCerts {
		clusterCertKey, err := os.ReadFile(fmt.Sprintf("%s/%s/cluster-cert-key.key", outputFolder, cluster))
		if err != nil {
			return err
		}

		for _, cert := range clusterCerts {
			certData, err := os.ReadFile(cert)
			if err != nil {
				return err
			}
			secretKey := fmt.Sprintf("%s-pem", filenameWithoutExtension(cert))
			data[secretKey] = []byte(fmt.Sprintf("%s\n%s", string(clusterCertKey), string(certData)))
		}
	}

	for _, mc := range memberClusters {
		client := clusterClients[mc.name].client
		if err := createSecret(client, mc.namespace, name, data); err != nil {
			return fmt.Errorf("error creating cluster cert map secret in cluster %s: %s", mc.name, err)
		}
		fmt.Printf(" - cluster: %s, secret: %s\n", mc.name, name)
	}

	return nil
}

func getMongoDBResource(clients KubeClients, flags flags) (map[string]interface{}, []MemberCluster, error) {
	crds, err := clients.apiExtClient.CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	var mdbMultiCRD *apiExt.CustomResourceDefinition
	if crds.Items != nil {
		for _, crd := range crds.Items {
			if crd.Spec.Names.Kind == mongoDBMultiKind {
				mdbMultiCRD = &crd
				break
			}
		}
	}

	if mdbMultiCRD == nil {
		return nil, nil, fmt.Errorf("%s CRD not found in cluster %s", mongoDBMultiKind, flags.centralCluster)
	}

	u := &unstructured.Unstructured{}
	u.SetAPIVersion(fmt.Sprintf("%s/%s", mdbMultiCRD.Spec.Group, mdbMultiCRD.Spec.Versions[0].Name))
	u.SetKind(mdbMultiCRD.Spec.Names.Kind)
	resourceKey := ctrlRuntime.ObjectKey{Name: flags.resourceName, Namespace: flags.centralClusterNamespace}
	if err := clients.ctrlRuntimeClient.Get(context.Background(), resourceKey, u); err != nil {
		return nil, nil, err
	}

	memberClusters := []MemberCluster{}
	clusterSpecList := u.Object["spec"].(map[string]interface{})["clusterSpecList"].(map[string]interface{})["clusterSpecs"].([]interface{})
	for _, clusterSpec := range clusterSpecList {
		memberClusters = append(memberClusters, MemberCluster{
			name:      clusterSpec.(map[string]interface{})["clusterName"].(string),
			members:   int(clusterSpec.(map[string]interface{})["members"].(int64)),
			namespace: flags.centralClusterNamespace,
		})
	}

	return u.Object, memberClusters, nil
}

func updateResourceTLSConfig(clients KubeClients, cr map[string]interface{}) error {
	fmt.Println("\n6. update MongoDB resource TLS config")
	cr["spec"].(map[string]interface{})["security"] = map[string]interface{}{
		"certsSecretPrefix": "",
		"tls": map[string]interface{}{
			"enabled": true,
			"ca":      caConfigMapName,
			"secretRef": map[string]interface{}{
				"name":   "",
				"prefix": clusterCertsPrefix,
			},
		},
	}
	security, _ := yaml.Marshal(cr["spec"].(map[string]interface{})["security"])
	fmt.Printf("spec.security:\n---\n%s\n---\n", string(security))
	return clients.ctrlRuntimeClient.Update(context.TODO(), &unstructured.Unstructured{Object: cr})
}

func main() {
	flags, err := parseFlags()
	if err != nil {
		fmt.Printf("error parsing flags: %s\n", err)
		os.Exit(1)
	}

	clusterClients := map[string]KubeClients{}
	clients, err := getKubernetesClientSet(flags.centralCluster)
	if err != nil {
		fmt.Printf("error creating kubernetes api clients: %s", err)
		os.Exit(1)
	}
	clusterClients[flags.centralCluster] = *clients

	cr, memberClusters, err := getMongoDBResource(clusterClients[flags.centralCluster], flags)
	if err != nil {
		fmt.Printf("error loading MongoDB Resource: %s", err)
		os.Exit(1)
	}

	for _, mc := range memberClusters {
		os.MkdirAll(fmt.Sprintf("%s/%s", outputFolder, mc.name), os.ModePerm)

		clients, err := getKubernetesClientSet(mc.name)
		if err != nil {
			fmt.Printf("error creating kubernetes api clients: %s", err)
			os.Exit(1)
		}
		clusterClients[mc.name] = *clients
	}

	var ca *CA
	if ca, err = createCA(flags); err != nil {
		fmt.Printf("error creating self-signed Certificate Authority: %s", err)
		os.Exit(1)
	}

	if err := createClusterCSRs(memberClusters, flags); err != nil {
		fmt.Printf("error creating cluster signings (CSR): %s", err)
		os.Exit(1)
	}

	var clusterServiceCerts map[string][]string
	if clusterServiceCerts, err = createClusterServiceCerts(*ca, flags, flags.resourceName, memberClusters); err != nil {
		fmt.Printf("error generating server certificates: %s", err)
		os.Exit(1)
	}

	if err := createCAConfigMaps(clusterClients, *ca, memberClusters); err != nil {
		fmt.Printf("error creating CA config map: %s", err)
		os.Exit(1)
	}

	if err := createClusterCertSecrets(clusterClients, flags.resourceName, clusterServiceCerts, memberClusters); err != nil {
		fmt.Printf("error creating cluster cert secrets: %s", err)
		os.Exit(1)
	}

	if err := updateResourceTLSConfig(clusterClients[flags.centralCluster], cr); err != nil {
		fmt.Printf("error updating MongoDB resource: %s", err)
		os.Exit(1)
	}
}
