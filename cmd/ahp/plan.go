package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/spf13/cobra"
)

func cmdPlan() *cobra.Command {
	c := &cobra.Command{
		Use:   "plan",
		Short: "Preview ranking and CR if proposals were committed (no writes)",
		Long: `Recompute as if proposal judgments were committed and show Δrank / ΔCR.
Does not flip status columns or write output/. Use ahp apply to commit.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runPlan,
	}
	addWorkspaceFlag(c)
	return c
}

func runPlan(cmd *cobra.Command, args []string) error {
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	plan, err := ws.Plan()
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	if p.JSON {
		return p.PrintJSON(plan)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Plan  %s  ·  %s\n", plan.Title, plan.Workspace)
	fmt.Fprintf(cmd.OutOrStdout(), "Proposals  %d pending\n", plan.Proposals)
	fmt.Fprintf(cmd.OutOrStdout(), "Complete   %v → %v    Consistent  %v → %v\n",
		plan.CompleteBefore, plan.CompleteAfter, plan.ConsistentBefore, plan.ConsistentAfter)

	if len(plan.RankDeltas) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nRanking deltas:")
		for _, d := range plan.RankDeltas {
			if d.DeltaRank == 0 && abs(d.DeltaWeight) < 1e-6 {
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %s  rank %d→%d  weight %.4f→%.4f  (Δw %+.4f)\n",
				d.Name, d.RankBefore, d.RankAfter, d.WeightBefore, d.WeightAfter, d.DeltaWeight)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "\nMatrix CR:")
	for _, m := range plan.MatrixDeltas {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  complete %v→%v  CR %s→%s\n",
			m.Matrix, m.CompleteBefore, m.CompleteAfter, fmtOptCR(m.CRBefore), fmtOptCR(m.CRAfter))
	}

	if plan.Proposals > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nNext:  ahp apply -y    # commit %d proposal(s)\n", plan.Proposals)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "\nNo proposals to apply.")
	}
	return nil
}

func fmtOptCR(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.3f", *v)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}