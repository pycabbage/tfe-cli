package cmd

import (
	"github.com/pycabbage/tfe-cli/internal/output"
	"github.com/spf13/cobra"
	"strconv"
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

		headers := []string{"ID", "SERIAL", "CREATED AT", "CREATED BY"}
		rows := make([][]string, 0, len(versions))
		for _, sv := range versions {
			rows = append(rows, []string{
				sv.ID,
				strconv.FormatInt(sv.Attributes.Serial, 10),
				sv.Attributes.CreatedAt.Format("2006-01-02 15:04:05"),
				sv.Attributes.CreatedBy.Username,
			})
		}
		output.PrintTable(headers, rows)
		return nil
	},
}
