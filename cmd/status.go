package cmd

import (
	"fmt"

	"github.com/pycabbage/tfe-cli/internal/config"
	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		details, err := client.GetAccountDetails(ctx)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		cfg, _ := config.Load()
		attrs := details.Data.Attributes
		twoFA := "disabled"
		if attrs.TwoFactor.Enabled {
			twoFA = "enabled"
		}

		output.PrintKV([][2]string{
			{"Username", attrs.Username},
			{"Email", attrs.Email},
			{"Two-Factor Auth", twoFA},
			{"Organization", cfg.Organization},
			{"Workspace", cfg.WorkspaceName},
		})
		return nil
	},
}
