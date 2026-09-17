package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdNext() *cobra.Command {
	c := &cobra.Command{
		Use:   "next",
		Short: "Print the single best next action for this workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			include, _ := cmd.Flags().GetBool("include-proposals")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			status, err := ws.Status(include)
			if err != nil {
				return err
			}
			next := workspace.RecommendNext(status)
			if p.JSON {
				return p.PrintJSON(next)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n# %s\n", next.Command, next.Reason)
			return nil
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	return c
}
