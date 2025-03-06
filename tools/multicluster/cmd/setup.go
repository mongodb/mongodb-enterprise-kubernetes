package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/10gen/ops-manager-kubernetes/multi/pkg/common"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	multiclusterCmd.AddCommand(setupCmd)

	setupCmd.Flags().StringVar(&common.MemberClusters, "member-clusters", "", "Comma separated list of member clusters. [required]")
	setupCmd.Flags().StringVar(&setupFlags.ServiceAccount, "service-account", "mongodb-enterprise-operator-multi-cluster", "Name of the service account which should be used for the Operator to communicate with the member clusters. [optional, default: mongodb-enterprise-operator-multi-cluster]")
	setupCmd.Flags().StringVar(&setupFlags.CentralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [required]")
	setupCmd.Flags().StringVar(&setupFlags.MemberClusterNamespace, "member-cluster-namespace", "", "The namespace the member cluster resources will be deployed to. [required]")
	setupCmd.Flags().StringVar(&setupFlags.CentralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [required]")
	setupCmd.Flags().BoolVar(&setupFlags.Cleanup, "cleanup", false, "Delete all previously created resources except for namespaces. [optional default: false]")
	setupCmd.Flags().BoolVar(&setupFlags.ClusterScoped, "cluster-scoped", false, "Create ClusterRole and ClusterRoleBindings for member clusters. [optional default: false]")
	setupCmd.Flags().BoolVar(&setupFlags.CreateTelemetryClusterRoles, "create-telemetry-roles", true, "Create ClusterRole and ClusterRoleBindings for member clusters for telemetry. [optional default: true]")
	setupCmd.Flags().BoolVar(&setupFlags.InstallDatabaseRoles, "install-database-roles", false, "Install the ServiceAccounts and Roles required for running database workloads in the member clusters. [optional default: false]")
	setupCmd.Flags().BoolVar(&setupFlags.CreateServiceAccountSecrets, "create-service-account-secrets", true, "Create service account token secrets. [optional default: true]")
	setupCmd.Flags().StringVar(&setupFlags.ImagePullSecrets, "image-pull-secrets", "", "Name of the secret for imagePullSecrets to set in created service accounts")
	setupCmd.Flags().StringVar(&common.MemberClustersApiServers, "member-clusters-api-servers", "", "Comma separated list of api servers addresses. [optional, default will take addresses from KUBECONFIG env var]")
}

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup the multicluster environment for MongoDB resources",
	Long: `'setup' configures the central and member clusters in preparation for a MongoDBMultiCluster deployment.

Example:

kubectl-mongodb multicluster setup --central-cluster="operator-cluster" --member-clusters="cluster-1,cluster-2,cluster-3" --member-cluster-namespace=mongodb --central-cluster-namespace=mongodb --create-service-account-secrets --install-database-roles

`,
	Run: func(cmd *cobra.Command, _ []string) {
		if err := parseSetupFlags(); err != nil {
			fmt.Printf("error parsing flags: %s\n", err)
			os.Exit(1)
		}

		buildInfo, ok := debug.ReadBuildInfo()
		if ok {
			fmt.Println(getBuildInfoString(buildInfo))
		}

		clientMap, err := common.CreateClientMap(setupFlags.MemberClusters, setupFlags.CentralCluster, common.LoadKubeConfigFilePath(), common.GetKubernetesClient)
		if err != nil {
			fmt.Printf("failed to create clientset map: %s", err)
			os.Exit(1)
		}

		if err := common.EnsureMultiClusterResources(cmd.Context(), setupFlags, clientMap); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := common.ReplaceClusterMembersConfigMap(cmd.Context(), clientMap[setupFlags.CentralCluster], setupFlags); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var setupFlags = common.Flags{}

func parseSetupFlags() error {
	if common.AnyAreEmpty(common.MemberClusters, setupFlags.ServiceAccount, setupFlags.CentralCluster, setupFlags.MemberClusterNamespace, setupFlags.CentralClusterNamespace) {
		return xerrors.Errorf("non empty values are required for [service-account, member-clusters, central-cluster, member-cluster-namespace, central-cluster-namespace]")
	}

	setupFlags.MemberClusters = strings.Split(common.MemberClusters, ",")

	if strings.TrimSpace(common.MemberClustersApiServers) != "" {
		setupFlags.MemberClusterApiServerUrls = strings.Split(common.MemberClustersApiServers, ",")
		if len(setupFlags.MemberClusterApiServerUrls) != len(setupFlags.MemberClusters) {
			return xerrors.Errorf("expected %d addresses in member-clusters-api-servers parameter but got %d", len(setupFlags.MemberClusters), len(setupFlags.MemberClusterApiServerUrls))
		}
	}

	configFilePath := common.LoadKubeConfigFilePath()
	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return xerrors.Errorf("error loading kubeconfig file '%s': %w", configFilePath, err)
	}
	if len(setupFlags.MemberClusterApiServerUrls) == 0 {
		if setupFlags.MemberClusterApiServerUrls, err = common.GetMemberClusterApiServerUrls(kubeconfig, setupFlags.MemberClusters); err != nil {
			return err
		}
	}
	return nil
}
