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
		user, err := client.GetCurrentUser(ctx)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		cfg, _ := config.Load()
		twoFA := "disabled"
		if user.TwoFactor != nil && user.TwoFactor.Enabled {
			twoFA = "enabled"
		}

		output.PrintKV([][2]string{
			{"Username", user.Username},
			{"Email", user.Email},
			{"Two-Factor Auth", twoFA},
			{"Organization", cfg.Organization},
			{"Workspace", cfg.WorkspaceName},
		})
		return nil
	},
}
