package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/spf13/cobra"
)

func cmdRemoveAlternative() *cobra.Command {
	c := &cobra.Command{
		Use:   "remove-alternative [id]",
		Short: "Delete an alternative and clean attributes + pairwise that reference it",
		Long: `Remove an alternative from the shortlist without rebuilding the workspace.

Deletes the row in alternatives.csv, all attributes for that id, and any
pairwise judgments (any matrix) where left or right equals the id.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			res, err := ws.RemoveAlternative(args[0])
			if err != nil {
				return cliout.Wrap(cliout.ExitUsage, err)
			}
			if p.JSON {
				return p.PrintJSON(res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed alternative %s  (attrs=%d pairs=%d)\n",
				res.AlternativeID, res.RemovedAttributes, res.RemovedPairs)
			fmt.Fprintln(cmd.OutOrStdout(), "Next:  ahp doctor   # then ahp status / rate --refresh")
			return nil
		},
	}
	addWorkspaceFlag(c)
	return c
}
