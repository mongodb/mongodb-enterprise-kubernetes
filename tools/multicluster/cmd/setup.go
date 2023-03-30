package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	multiclusterCmd.AddCommand(setupCmd)

	setupCmd.Flags().StringVar(&memberClusters, "member-clusters", "", "Comma separated list of member clusters. [required]")
	setupCmd.Flags().StringVar(&setupFlags.serviceAccount, "service-account", "mongodb-enterprise-operator-multi-cluster", "Name of the service account which should be used for the Operator to communicate with the member clusters. [optional, default: mongodb-enterprise-operator-multi-cluster]")
	setupCmd.Flags().StringVar(&setupFlags.centralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [required]")
	setupCmd.Flags().StringVar(&setupFlags.memberClusterNamespace, "member-cluster-namespace", "", "The namespace the member cluster resources will be deployed to. [required]")
	setupCmd.Flags().StringVar(&setupFlags.centralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [required]")
	setupCmd.Flags().BoolVar(&setupFlags.cleanup, "cleanup", false, "Delete all previously created resources except for namespaces. [optional default: false]")
	setupCmd.Flags().BoolVar(&setupFlags.clusterScoped, "cluster-scoped", false, "Create ClusterRole and ClusterRoleBindings for member clusters. [optional default: false]")
	setupCmd.Flags().BoolVar(&setupFlags.installDatabaseRoles, "install-database-roles", false, "Install the ServiceAccounts and Roles required for running database workloads in the member clusters. [optional default: false]")
	setupCmd.Flags().BoolVar(&setupFlags.createServiceAccountSecrets, "create-service-account-secrets", true, "Create service account token secrets. [optional default: true]")
	setupCmd.Flags().StringVar(&memberClustersApiServers, "member-clusters-api-servers", "", "Comma separated list of api servers addresses. [optional, default will take addresses from KUBECONFIG env var]")
}

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup the multicluster environment for MongoDB resources",
	Long: `'setup' configures the central and member clusters in preparation for a MongoDBMultiCluster deployment.

Example:

kubectl-mongodb multicluster setup --central-cluster="operator-cluster" --member-clusters="cluster-1,cluster-2,cluster-3" --member-cluster-namespace=mongodb --central-cluster-namespace=mongodb --create-service-account-secrets --install-database-roles

`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := parseSetupFlags(args); err != nil {
			fmt.Printf("error parsing flags: %s\n", err)
			os.Exit(1)
		}

		clientMap, err := createClientMap(setupFlags.memberClusters, setupFlags.centralCluster, loadKubeConfigFilePath(), getKubernetesClient)
		if err != nil {
			fmt.Printf("failed to create clientset map: %s", err)
			os.Exit(1)
		}

		if err := ensureMultiClusterResources(cmd.Context(), setupFlags, clientMap); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := replaceClusterMembersConfigMap(cmd.Context(), clientMap[setupFlags.centralCluster], setupFlags); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	},
}

var setupFlags = flags{}

func parseSetupFlags(args []string) error {

	if anyAreEmpty(memberClusters, setupFlags.serviceAccount, setupFlags.centralCluster, setupFlags.memberClusterNamespace, setupFlags.centralClusterNamespace) {
		return xerrors.Errorf("non empty values are required for [service-account, member-clusters, central-cluster, member-cluster-namespace, central-cluster-namespace]")
	}

	setupFlags.memberClusters = strings.Split(memberClusters, ",")

	if strings.TrimSpace(memberClustersApiServers) != "" {
		setupFlags.memberClusterApiServerUrls = strings.Split(memberClustersApiServers, ",")
		if len(setupFlags.memberClusterApiServerUrls) != len(setupFlags.memberClusters) {
			return xerrors.Errorf("expected %d addresses in member-clusters-api-servers parameter but got %d", len(setupFlags.memberClusters), len(setupFlags.memberClusterApiServerUrls))
		}
	}

	configFilePath := loadKubeConfigFilePath()
	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return xerrors.Errorf("error loading kubeconfig file '%s': %w", configFilePath, err)
	}
	if len(setupFlags.memberClusterApiServerUrls) == 0 {
		if setupFlags.memberClusterApiServerUrls, err = getMemberClusterApiServerUrls(kubeconfig, setupFlags.memberClusters); err != nil {
			return err
		}
	}
	return nil
}
