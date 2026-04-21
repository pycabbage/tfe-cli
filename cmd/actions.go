package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var actionsCmd = &cobra.Command{
	Use:   "actions",
	Short: "Workspace lock/unlock operations",
}

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.Lock(cmd.Context(), "manual lock via tfe-cli"); err != nil {
			return err
		}
		fmt.Println("Workspace locked.")
		return nil
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.Unlock(cmd.Context()); err != nil {
			return err
		}
		fmt.Println("Workspace unlocked.")
		return nil
	},
}

func init() {
	actionsCmd.AddCommand(lockCmd)
	actionsCmd.AddCommand(unlockCmd)
}
