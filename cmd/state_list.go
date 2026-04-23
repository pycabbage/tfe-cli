package cmd

import (
	"fmt"

	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/spf13/cobra"
)

var stateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the last 10 state versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		versions, err := client.ListStateVersions(ctx)
		if err != nil {
			return err
		}

		headers := []string{"ID", "SERIAL", "CREATED AT", "STATUS"}
		rows := make([][]string, 0, len(versions))
		for _, sv := range versions {
			rows = append(rows, []string{
				sv.ID,
				fmt.Sprintf("%d", sv.Serial),
				sv.CreatedAt.Format("2006-01-02 15:04:05"),
				string(sv.Status),
			})
		}
		output.PrintTable(headers, rows)
		return nil
	},
}
