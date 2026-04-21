package cmd

import "github.com/spf13/cobra"

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage Terraform state versions",
}

func init() {
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(stateShowCmd)
	stateCmd.AddCommand(stateDownloadCmd)
	stateCmd.AddCommand(stateUploadCmd)
}
