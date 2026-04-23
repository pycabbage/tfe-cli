package cmd

import (
	"github.com/pycabbage/tfe-cli/internal/config"
	"github.com/pycabbage/tfe-cli/internal/tfc"
	"github.com/spf13/cobra"
)

var client *tfc.Client

var rootCmd = &cobra.Command{
	Use:           "tfe",
	Short:         "HCP Terraform state management CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isAuthCommand(cmd) {
			return nil
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client, err = tfc.New(cfg)
		return err
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(actionsCmd)
}

func isAuthCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "auth" {
			return true
		}
	}
	return false
}
