package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/pycabbage/tfe-cli/internal/tfc"
	"github.com/spf13/cobra"
)

var runShowCmd = &cobra.Command{
	Use:   "show [<latest|run-...>]",
	Short: "Show metadata of a run and its state version",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id := "latest"
		if len(args) > 0 {
			id = args[0]
			if id != "latest" && !strings.HasPrefix(id, "run-") {
				return fmt.Errorf("invalid run ID: %q (must be \"latest\" or \"run-...\")", id)
			}
		}
		run, err := client.GetRun(ctx, id)
		if err != nil {
			return err
		}

		sv, svErr := client.FindStateVersionForRun(ctx, run)

		rows := [][2]string{
			{"ID", run.ID},
			{"Status", string(run.Status)},
			{"Source", string(run.Source)},
			{"Message", run.Message},
			{"Terraform Version", run.TerraformVersion},
			{"Created At", run.CreatedAt.Format("2006-01-02 15:04:05")},
		}

		switch {
		case svErr != nil && errors.Is(svErr, tfc.ErrRunStateVersionScanLimit):
			rows = append(rows, [2]string{"State Version", "(could not determine — try `tfe state list`)"})
		case svErr != nil:
			return svErr
		case sv == nil:
			rows = append(rows, [2]string{"State Version", "(none)"})
		default:
			rows = append(rows,
				[2]string{"State Version ID", sv.ID},
				[2]string{"State Version Serial", fmt.Sprintf("%d", sv.Serial)},
			)
		}

		output.PrintKV(rows)
		return nil
	},
}
