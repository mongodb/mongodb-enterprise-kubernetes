package common

import (
	"os"
	"path/filepath"

	"golang.org/x/xerrors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

const (
	kubeConfigEnv = "KUBECONFIG"
)

// LoadKubeConfigFilePath returns the path of the local KubeConfig file.
func LoadKubeConfigFilePath() string {
	env := os.Getenv(kubeConfigEnv) // nolint:forbidigo
	if env != "" {
		return env
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

// GetMemberClusterApiServerUrls returns the slice of member cluster api urls that should be used.
func GetMemberClusterApiServerUrls(kubeconfig *clientcmdapi.Config, clusterNames []string) ([]string, error) {
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

// CreateClientMap crates a map of all MultiClusterClient for every member cluster, and the operator cluster.
func CreateClientMap(memberClusters []string, operatorCluster, kubeConfigPath string, getClient func(clusterName string, kubeConfigPath string) (KubeClient, error)) (map[string]KubeClient, error) {
	clientMap := map[string]KubeClient{}
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

// GetKubernetesClient returns a kubernetes.Clientset using the given context from the
// specified KubeConfig filepath.
func GetKubernetesClient(context, kubeConfigPath string) (KubeClient, error) {
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

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, xerrors.Errorf("failed to create dynamic kubernetes clientset: %w", err)
	}

	return NewKubeClientContainer(config, clientset, dynamicClient), nil
}
