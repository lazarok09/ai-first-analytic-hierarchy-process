package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/spf13/cobra"
)

func cmdExplain() *cobra.Command {
	c := &cobra.Command{
		Use:   "explain [alternative]",
		Short: "Break down global weight by criterion contributions",
		Long: `Show how each leaf criterion contributes to an alternative's global score
(contribution = criterion_weight × local_weight). Omit the alternative id to
explain the full ranking.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runExplain,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	return c
}

func runExplain(cmd *cobra.Command, args []string) error {
	include, _ := cmd.Flags().GetBool("include-proposals")
	alt := ""
	if len(args) == 1 {
		alt = args[0]
	}
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	exp, err := ws.Explain(include, alt)
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	if p.JSON {
		return p.PrintJSON(exp)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Explain  %s  ·  %s\n", exp.Title, exp.Workspace)
	if !exp.Complete {
		fmt.Fprintln(cmd.OutOrStdout(), "note: matrices incomplete — contributions use current (possibly equal-fallback) weights")
	}
	for _, a := range exp.Alternatives {
		fmt.Fprintf(cmd.OutOrStdout(), "\n#%d  %s  global %.4f\n", a.Rank, a.Name, a.GlobalWeight)
		for _, c := range a.Contributions {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-16s  crit=%.3f × local=%.3f  →  %.4f  (%4.1f%%)\n",
				c.CriterionName, c.CriterionWeight, c.LocalWeight, c.Contribution, c.Share*100)
		}
	}
	return nil
}
