package cmd

import (
	"fmt"
	"strings"

	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/spf13/cobra"
)

var stateShowCmd = &cobra.Command{
	Use:   "show [<latest|sv-...>]",
	Short: "Show metadata of a state version",
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
		output.PrintKV([][2]string{
			{"ID", sv.ID},
			{"Serial", fmt.Sprintf("%d", sv.Serial)},
			{"Status", string(sv.Status)},
			{"Terraform Version", sv.TerraformVersion},
			{"Created At", sv.CreatedAt.Format("2006-01-02 15:04:05")},
			{"Download URL", sv.DownloadURL},
		})
		return nil
	},
}
