package cmd

import (
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/build"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("opendex-docker %s-%s\n", build.Version, build.GitCommit[:7])
	},
}
