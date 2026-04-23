package cmd

import (
	"fmt"
	"os"

	"github.com/pycabbage/tfe-cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutProfile string

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := config.LoadStore()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		name := logoutProfile
		if name == "" {
			name = store.CurrentProfile
		}
		if name == "" {
			return fmt.Errorf("no active profile")
		}

		if err := store.DeleteProfile(name); err != nil {
			return err
		}

		if len(store.Profiles) == 0 {
			path, _ := config.ConfigPath()
			os.Remove(path)
			fmt.Printf("Removed profile %q and deleted config file\n", name)
			return nil
		}

		if err := store.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Removed profile %q\n", name)
		if store.CurrentProfile != "" {
			fmt.Printf("Active profile is now: %s\n", store.CurrentProfile)
		}
		return nil
	},
}

func init() {
	authLogoutCmd.Flags().StringVarP(&logoutProfile, "profile", "p", "", "profile name (default: current profile)")
	authCmd.AddCommand(authLogoutCmd)
}
