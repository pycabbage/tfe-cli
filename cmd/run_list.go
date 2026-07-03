package cmd

import (
	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/spf13/cobra"
)

var runListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the last 10 runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		runs, err := client.ListRuns(ctx)
		if err != nil {
			return err
		}

		headers := []string{"ID", "STATUS", "SOURCE", "CREATED AT"}
		rows := make([][]string, 0, len(runs))
		for _, run := range runs {
			rows = append(rows, []string{
				run.ID,
				string(run.Status),
				string(run.Source),
				run.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		output.PrintTable(headers, rows)
		return nil
	},
}
