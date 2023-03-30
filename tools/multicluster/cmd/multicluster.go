package cmd

import (
	"github.com/spf13/cobra"
)

// multiclusterCmd represents the multicluster command
var multiclusterCmd = &cobra.Command{
	Use:   "multicluster",
	Short: "Manage MongoDB multicluster environments on k8s",
	Long: `'multicluster' is the toplevel command for managing
multicluster environments that hold MongoDB resources.`,
}

func init() {
	rootCmd.AddCommand(multiclusterCmd)
}
