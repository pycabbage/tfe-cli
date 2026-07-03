package cmd

import "github.com/spf13/cobra"

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manage Terraform runs",
}

func init() {
	runCmd.AddCommand(runListCmd)
	runCmd.AddCommand(runShowCmd)
}
