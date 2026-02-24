package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newConfigCmd returns the "config" subcommand group for configuration management.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}

	cmd.AddCommand(newConfigValidateCmd())
	return cmd
}

// newConfigValidateCmd returns the "config validate" subcommand that checks config file validity.
func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := loadConfig(configPath)
			if err != nil {
				return err
			}
			fmt.Println(styleSuccess.Render("✓ Configuration is valid"))
			return nil
		},
	}
}
