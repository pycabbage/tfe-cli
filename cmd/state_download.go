package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var downloadOutput string

var stateDownloadCmd = &cobra.Command{
	Use:   "download [<latest|sv-...>]",
	Short: "Download a state version",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id := "latest"
		if len(args) > 0 {
			id = args[0]
			if id != "latest" && !strings.HasPrefix(id, "sv-") {
				return fmt.Errorf("invalid state version ID: %q (must be \"latest\" or \"sv-...\")", id)
			}
		}

		sv, err := client.GetStateVersion(ctx, id)
		if err != nil {
			return err
		}

		data, err := client.DownloadState(ctx, sv)
		if err != nil {
			return err
		}

		outFile := downloadOutput
		if outFile == "" {
			if id == "latest" {
				outFile = "terraform.tfstate"
			} else {
				outFile = id + ".tfstate"
			}
		}

		if err := os.WriteFile(outFile, data, 0644); err != nil {
			return fmt.Errorf("writing state file: %w", err)
		}
		fmt.Printf("Downloaded to: %s\n", outFile)
		return nil
	},
}

func init() {
	stateDownloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", "Output file path")
}
