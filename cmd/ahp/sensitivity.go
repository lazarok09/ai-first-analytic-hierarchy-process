package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/spf13/cobra"
)

func cmdSensitivity() *cobra.Command {
	c := &cobra.Command{
		Use:   "sensitivity",
		Short: "One-at-a-time criterion weight robustness (±δ, tornado, rank flip)",
		Long: `Perturb each leaf criterion weight by ±delta (renormalizing the rest) and
report leader weight change, tornado ordering, and estimated rank-reversal
thresholds. Writes output/sensitivity.json.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runSensitivity,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().Float64("delta", 0.05, "Absolute weight perturbation (±)")
	c.Flags().Bool("write", true, "Write output/sensitivity.json")
	return c
}

func runSensitivity(cmd *cobra.Command, args []string) error {
	include, _ := cmd.Flags().GetBool("include-proposals")
	delta, _ := cmd.Flags().GetFloat64("delta")
	writeOut, _ := cmd.Flags().GetBool("write")
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	sens, err := ws.Sensitivity(include, delta)
	if err != nil {
		return cliout.Wrap(cliout.ExitIncomplete, err)
	}

	var sensPath string
	if writeOut {
		if err := os.MkdirAll(ws.OutputDir(), 0o755); err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		sensPath = filepath.Join(ws.OutputDir(), "sensitivity.json")
		b, err := json.MarshalIndent(sens, "", "  ")
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		if err := os.WriteFile(sensPath, b, 0o644); err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
	}

	if p.JSON {
		payload := map[string]any{
			"workspace":         sens.Workspace,
			"title":             sens.Title,
			"include_proposals": sens.IncludeProposals,
			"complete":          sens.Complete,
			"delta":             sens.Delta,
			"base_leader_id":    sens.BaseLeaderID,
			"base_leader_name":  sens.BaseLeaderName,
			"base_ranking":      sens.BaseRanking,
			"by_criterion":      sens.ByCriterion,
			"summary":           sens.Summary,
		}
		if sensPath != "" {
			payload["sensitivity_json"] = sensPath
		}
		return p.PrintJSON(payload)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Sensitivity  %s  ·  %s\n", sens.Title, sens.Workspace)
	fmt.Fprintf(cmd.OutOrStdout(), "Leader  %s   delta ±%.2f\n", sens.BaseLeaderName, sens.Delta)
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", sens.Summary)
	fmt.Fprintln(cmd.OutOrStdout(), "\nTornado (by |Δ leader weight|):")
	for _, c := range sens.ByCriterion {
		flip := ""
		if c.Plus.LeaderChanged || c.Minus.LeaderChanged {
			flip = "  FLIP@±δ"
		}
		rev := ""
		if c.ReversalDelta != nil {
			rev = fmt.Sprintf("  reverse@Δ=%+.3f", *c.ReversalDelta)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-16s  base=%.3f  tornado=%.4f  +δ Δw=%+.4f  -δ Δw=%+.4f%s%s\n",
			c.CriterionName, c.BaseWeight, c.TornadoEffect,
			c.Plus.LeaderDeltaW, c.Minus.LeaderDeltaW, flip, rev)
	}
	if sensPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "\nwrote %s\n", sensPath)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Next:  ahp explain")
	return nil
}
