package cmd

import (
	"fmt"
	"strconv"
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
		a := sv.Attributes
		output.PrintKV([][2]string{
			{"ID", sv.ID},
			{"Serial", strconv.FormatInt(a.Serial, 10)},
			{"Status", a.Status},
			{"Finalized", strconv.FormatBool(a.Finalized)},
			{"Terraform Version", a.TerraformVersion},
			{"Lineage", a.Lineage},
			{"Created At", a.CreatedAt.Format("2006-01-02 15:04:05")},
			{"Created By", a.CreatedBy.Username},
			{"Download URL", a.DownloadURL},
		})
		return nil
	},
}
