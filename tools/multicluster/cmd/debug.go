package cmd

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/10gen/ops-manager-kubernetes/multi/pkg/common"
	"github.com/10gen/ops-manager-kubernetes/multi/pkg/debug"
	"github.com/spf13/cobra"
)

type Flags struct {
	common.Flags
	Anonymize   bool
	UseOwnerRef bool
}

func (f *Flags) ParseDebugFlags() error {
	if len(common.MemberClusters) > 0 {
		f.MemberClusters = strings.Split(common.MemberClusters, ",")
	}

	configFilePath := common.LoadKubeConfigFilePath()
	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig file '%s': %s", configFilePath, err)
	}
	if len(f.CentralCluster) == 0 {
		f.CentralCluster = kubeconfig.CurrentContext
		f.CentralClusterNamespace = kubeconfig.Contexts[kubeconfig.CurrentContext].Namespace
	}

	return nil
}

var debugFlags = &Flags{}

func init() {
	rootCmd.AddCommand(debugCmd)

	debugCmd.Flags().StringVar(&common.MemberClusters, "member-clusters", "", "Comma separated list of member clusters. [optional]")
	debugCmd.Flags().StringVar(&debugFlags.CentralCluster, "central-cluster", "", "The central cluster the operator will be deployed in. [optional]")
	debugCmd.Flags().StringVar(&debugFlags.MemberClusterNamespace, "member-cluster-namespace", "", "The namespace the member cluster resources will be deployed to. [optional]")
	debugCmd.Flags().StringVar(&debugFlags.CentralClusterNamespace, "central-cluster-namespace", "", "The namespace the Operator will be deployed to. [optional]")
	debugCmd.Flags().StringVar(&common.MemberClustersApiServers, "member-clusters-api-servers", "", "Comma separated list of api servers addresses. [optional, default will take addresses from KUBECONFIG env var]")
	debugCmd.Flags().BoolVar(&debugFlags.Anonymize, "anonymize", true, "True if anonymization should be turned on")
	debugCmd.Flags().BoolVar(&debugFlags.UseOwnerRef, "ownerRef", false, "True if the collection should be made with owner references (consider turning it on after CLOUDP-176772 is fixed)")
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Downloads all resources required for debugging and stores them into the disk",
	Long: `'debug' downloads all resources required for debugging and stores them into the disk.

Example:

kubectl-mongodb debug
kubectl-mongodb debug setup --central-cluster="operator-cluster" --member-clusters="cluster-1,cluster-2,cluster-3" --member-cluster-namespace=mongodb --central-cluster-namespace=mongodb

`,
	Run: func(cmd *cobra.Command, args []string) {
		err := debugFlags.ParseDebugFlags()
		if err != nil {
			fmt.Printf("error parsing flags: %s\n", err)
			os.Exit(1)
		}
		clientMap, err := common.CreateClientMap(debugFlags.MemberClusters, debugFlags.CentralCluster, common.LoadKubeConfigFilePath(), common.GetKubernetesClient)
		if err != nil {
			fmt.Printf("failed to create clientset map: %s", err)
			os.Exit(1)
		}

		var collectors []debug.Collector
		collectors = append(collectors, &debug.StatefulSetCollector{})
		collectors = append(collectors, &debug.ConfigMapCollector{})
		collectors = append(collectors, &debug.SecretCollector{})
		collectors = append(collectors, &debug.ServiceAccountCollector{})
		collectors = append(collectors, &debug.RolesCollector{})
		collectors = append(collectors, &debug.RolesBindingsCollector{})
		collectors = append(collectors, &debug.MongoDBCollector{})
		collectors = append(collectors, &debug.MongoDBMultiClusterCollector{})
		collectors = append(collectors, &debug.MongoDBUserCollector{})
		collectors = append(collectors, &debug.OpsManagerCollector{})
		collectors = append(collectors, &debug.MongoDBCommunityCollector{})
		collectors = append(collectors, &debug.EventsCollector{})
		collectors = append(collectors, &debug.LogsCollector{})
		collectors = append(collectors, &debug.AgentHealthFileCollector{})

		var anonymizer debug.Anonymizer
		if debugFlags.Anonymize {
			anonymizer = &debug.SensitiveDataAnonymizer{}
		} else {
			anonymizer = &debug.NoOpAnonymizer{}
		}

		var filter debug.Filter

		if debugFlags.UseOwnerRef {
			filter = &debug.WithOwningReference{}
		} else {
			filter = &debug.AcceptAllFilter{}
		}

		var collectionResults []debug.CollectionResult

		collectionResults = append(collectionResults, debug.Collect(cmd.Context(), clientMap[debugFlags.CentralCluster], debugFlags.CentralCluster, debugFlags.CentralClusterNamespace, filter, collectors, anonymizer))

		if len(debugFlags.MemberClusters) > 0 {
			for i := range debugFlags.MemberClusters {
				collectionResults = append(collectionResults, debug.Collect(cmd.Context(), clientMap[debugFlags.MemberClusters[i]], debugFlags.MemberClusters[i], debugFlags.MemberClusterNamespace, filter, collectors, anonymizer))
			}
		}

		fmt.Printf("==== Report ====\n\n")
		fmt.Printf("Anonymisation: %v\n", debugFlags.Anonymize)
		fmt.Printf("Following owner refs: %v\n", debugFlags.UseOwnerRef)
		fmt.Printf("Collected data from %d clusters\n", len(collectionResults))
		fmt.Printf("\n\n==== Collected Data ====\n\n")

		storeDirectory, err := debug.DebugDirectory()
		if err != nil {
			fmt.Printf("failed to obtain directory for collecting the results: %v", err)
			os.Exit(1)
		}

		if len(collectionResults) > 0 {
			directoryName, compressedFileName, err := debug.WriteToFile(storeDirectory, collectionResults...)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Debug data file (compressed): %v\n", compressedFileName)
			fmt.Printf("Debug data directory: %v\n", directoryName)
		}
	},
}
