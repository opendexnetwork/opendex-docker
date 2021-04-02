package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(consoleCmd)
}

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Start the console",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return launcher.Apply()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()
		return launcher.StartConsole(ctx)
	},
}

