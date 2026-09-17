package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdTree() *cobra.Command {
	c := &cobra.Command{
		Use:   "tree",
		Short: "Print goal → criteria → alternative-matrix hierarchy",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			tree, err := ws.Tree()
			if err != nil {
				return err
			}
			if p.JSON {
				return p.PrintJSON(tree)
			}
			fmt.Fprint(cmd.OutOrStdout(), workspace.FormatTree(tree))
			return nil
		},
	}
	addWorkspaceFlag(c)
	return c
}
