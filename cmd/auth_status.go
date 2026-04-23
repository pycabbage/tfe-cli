package cmd

import (
	"fmt"

	"github.com/pycabbage/tfe-cli/internal/config"
	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/pycabbage/tfe-cli/internal/tfc"
	"github.com/spf13/cobra"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, _ := config.LoadStore()
		profileName := ""
		if store != nil {
			profileName = store.CurrentProfile
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Println("Not logged in. Run `tfe auth login` to authenticate.")
			return nil
		}

		c, err := tfc.New(cfg)
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		user, err := c.GetCurrentUser(cmd.Context())
		if err != nil {
			fmt.Println("Authentication failed. Token may be invalid.")
			fmt.Printf("Run `tfe auth login` to re-authenticate.\n")
			return nil
		}

		twoFA := "disabled"
		if user.TwoFactor != nil && user.TwoFactor.Enabled {
			twoFA = "enabled"
		}

		pairs := [][2]string{
			{"Profile", profileName},
			{"Username", user.Username},
			{"Email", user.Email},
			{"Two-Factor Auth", twoFA},
			{"Organization", cfg.Organization},
			{"Workspace", cfg.WorkspaceName},
		}
		output.PrintKV(pairs)
		return nil
	},
}

func init() {
	authCmd.AddCommand(authStatusCmd)
}
