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
	multiclusterCmd.AddCommand(recoverCmd)

	recoverCmd.Flags().StringVar(&memberClusters, "member-clusters", "", "Comma separated list of member clusters. [required]")
	recoverCmd.Flags().StringVar(&recoverFlags.serviceAccount, "service-account", "mongodb-enterprise-operator-multi-cluster", "Name of the service account which should be used for the Operator to communicate with the member clusters. [optional, default: mongodb-enterprise-operator-multi-cluster]")
	recoverCmd.Flags().StringVar(&recoverFlags.centralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [required]")
	recoverCmd.Flags().StringVar(&recoverFlags.memberClusterNamespace, "member-cluster-namespace", "", "The namespace the member cluster resources will be deployed to. [required]")
	recoverCmd.Flags().StringVar(&recoverFlags.centralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [required]")
	recoverCmd.Flags().BoolVar(&recoverFlags.cleanup, "cleanup", false, "Delete all previously created resources except for namespaces. [optional default: false]")
	recoverCmd.Flags().BoolVar(&recoverFlags.clusterScoped, "cluster-scoped", false, "Create ClusterRole and ClusterRoleBindings for member clusters. [optional default: false]")
	recoverCmd.Flags().StringVar(&recoverFlags.operatorName, "operator-name", defaultOperatorName, "Name used to identify the deployment of the operator. [optional, default: mongodb-enterprise-operator]")
	recoverCmd.Flags().BoolVar(&recoverFlags.installDatabaseRoles, "install-database-roles", false, "Install the ServiceAccounts and Roles required for running database workloads in the member clusters. [optional default: false]")
	recoverCmd.Flags().StringVar(&recoverFlags.sourceCluster, "source-cluster", "", "The source cluster for recovery. This has to be one of the healthy member cluster that is the source of truth for new cluster configuration. [required]")
	recoverCmd.Flags().BoolVar(&recoverFlags.createServiceAccountSecrets, "create-service-account-secrets", true, "Create service account token secrets. [optional default: true]")
	recoverCmd.Flags().StringVar(&memberClustersApiServers, "member-clusters-api-servers", "", "Comma separated list of api servers addresses. [optional, default will take addresses from KUBECONFIG env var]")
}

// recoverCmd represents the recover command
var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Recover the multicluster environment for MongoDB resources after a dataplane failure",
	Long: `'recover' re-configures a failed multicluster environment to a enable the shuffling of dataplane
resources to a new healthy topology.

Example:

kubectl-mongodb multicluster recover --central-cluster="operator-cluster" --member-clusters="cluster-1,cluster-3,cluster-4" --member-cluster-namespace="mongodb-fresh" --central-cluster-namespace="mongodb" --operator-name=mongodb-enterprise-operator-multi-cluster --source-cluster="cluster-1"

`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := parseRecoverFlags(args); err != nil {
			fmt.Printf("error parsing flags: %s\n", err)
			os.Exit(1)
		}

		clientMap, err := createClientMap(recoverFlags.memberClusters, recoverFlags.centralCluster, loadKubeConfigFilePath(), getKubernetesClient)
		if err != nil {
			fmt.Printf("failed to create clientset map: %s", err)
			os.Exit(1)
		}

		if err := ensureMultiClusterResources(cmd.Context(), recoverFlags, clientMap); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := replaceClusterMembersConfigMap(cmd.Context(), clientMap[recoverFlags.centralCluster], recoverFlags); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	},
}

var recoverFlags = flags{}

func parseRecoverFlags(args []string) error {
	if anyAreEmpty(memberClusters, recoverFlags.serviceAccount, recoverFlags.centralCluster, recoverFlags.memberClusterNamespace, recoverFlags.centralClusterNamespace, recoverFlags.sourceCluster) {
		return xerrors.Errorf("non empty values are required for [service-account, member-clusters, central-cluster, member-cluster-namespace, central-cluster-namespace, source-cluster]")
	}

	recoverFlags.memberClusters = strings.Split(memberClusters, ",")
	if !contains(recoverFlags.memberClusters, recoverFlags.sourceCluster) {
		return xerrors.Errorf("source-cluster has to be one of the healthy member clusters: %s", memberClusters)
	}

	if strings.TrimSpace(memberClustersApiServers) != "" {
		recoverFlags.memberClusterApiServerUrls = strings.Split(memberClustersApiServers, ",")
		if len(recoverFlags.memberClusterApiServerUrls) != len(recoverFlags.memberClusters) {
			return xerrors.Errorf("expected %d addresses in member-clusters-api-servers parameter but got %d", len(recoverFlags.memberClusters), len(recoverFlags.memberClusterApiServerUrls))
		}
	}

	configFilePath := loadKubeConfigFilePath()
	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return xerrors.Errorf("error loading kubeconfig file '%s': %w", configFilePath, err)
	}
	if len(recoverFlags.memberClusterApiServerUrls) == 0 {
		if recoverFlags.memberClusterApiServerUrls, err = getMemberClusterApiServerUrls(kubeconfig, recoverFlags.memberClusters); err != nil {
			return err
		}
	}
	return nil
}
