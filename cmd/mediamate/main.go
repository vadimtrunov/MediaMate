package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.2.0"

var configPath string

// newRootCmd creates the root Cobra command with all subcommands registered.
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "mediamate",
		Short: "AI-powered media server assistant",
		Long: "MediaMate is an AI-powered assistant for managing your personal media server.\n" +
			"It helps you search, download, and manage movies through natural conversation.",
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "configs/mediamate.yaml", "path to configuration file")

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	rootCmd.AddCommand(
		newVersionCmd(),
		newChatCmd(),
		newQueryCmd(),
		newStatusCmd(),
		newConfigCmd(),
		newBotCmd(),
		newStackCmd(),
		newMCPServeCmd(),
	)

	return rootCmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, styleError.Render(err.Error()))
		os.Exit(1)
	}
}

// newVersionCmd returns the "version" subcommand.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("MediaMate v%s\n", version)
		},
	}
}
