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
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(actionsCmd)
}
