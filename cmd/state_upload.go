package cmd

import (
	"github.com/spf13/cobra"
)

var stateUploadCmd = &cobra.Command{
	Use:   "upload [file_path]",
	Short: "Upload a state version",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := "terraform.tfstate"
		if len(args) > 0 {
			filePath = args[0]
		}
		return client.UploadState(cmd.Context(), filePath)
	},
}
