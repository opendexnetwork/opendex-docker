package cmd

import (
	"context"
	"github.com/spf13/cobra"
)

type SetupOptions struct {
	NoPull bool
}

var (
	setupOpts SetupOptions
)

func init() {
	setupCmd.PersistentFlags().BoolVar(&setupOpts.NoPull, "nopull", false, "don't pull images")
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up OpenDEX environment",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return launcher.Apply()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()
		ctx = context.WithValue(ctx, "rescue", false)
		return launcher.Setup(ctx, !setupOpts.NoPull)
	},
}
