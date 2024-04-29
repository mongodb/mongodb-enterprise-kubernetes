package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubectl-mongodb",
	Short: "Manage and configure MongoDB resources on k8s",
	Long: `This application is a tool to simplify maintenance tasks
of MongoDB resources in your kubernetes cluster.
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		cancel()
	}()
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		rootCmd.Long += getBuildInfoString(buildInfo)
	}
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func getBuildInfoString(buildInfo *debug.BuildInfo) string {
	var vcsHash string
	var vcsTime string
	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			vcsHash = setting.Value
		}
		if setting.Key == "vcs.time" {
			vcsTime = setting.Value
		}
	}

	buildInfoStr := fmt.Sprintf("\nBuild: %s, %s", vcsHash, vcsTime)
	return buildInfoStr
}
