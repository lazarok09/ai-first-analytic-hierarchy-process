package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdPair() *cobra.Command {
	c := &cobra.Command{
		Use:   "pair",
		Short: "Judgment UX: missing pairs, set, repairs, interactive ask",
		Long: `Work with Saaty pairwise judgments.

  ahp pair missing   list gaps
  ahp pair set       write one judgment (--as proposal|committed)
  ahp pair repairs   CR repair suggestions
  ahp pair ask       interactive walk over missing pairs (TTY only)

Agents / non-TTY default --as proposal (also AHP_AGENT=1).`,
	}
	c.AddCommand(cmdPairMissing(), cmdPairSet(), cmdPairRepairs(), cmdPairAsk())
	return c
}

func cmdPairMissing() *cobra.Command {
	c := &cobra.Command{
		Use:   "missing",
		Short: "List incomplete Saaty pairs (committed vs proposal coverage)",
		Long: `Report pairwise gaps with proposal coverage:

  missing_committed     gaps ignoring proposals
  covered_by_proposals  gaps that proposals already fill
  uncovered             gaps still needing a judgment

Default JSON is this object (not a flat list). Human output shows a coverage
summary plus an uncovered table (or notes when proposals fill all gaps).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			include, _ := cmd.Flags().GetBool("include-proposals")
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			s, err := ws.Status(include)
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			report := workspace.MissingCoverage{
				MissingCommitted:   s.MissingCommitted,
				CoveredByProposals: s.CoveredByProposals,
				Uncovered:          s.Uncovered,
			}
			if report.MissingCommitted == nil {
				report.MissingCommitted = []workspace.MissingPair{}
			}
			if report.CoveredByProposals == nil {
				report.CoveredByProposals = []workspace.MissingPair{}
			}
			if report.Uncovered == nil {
				report.Uncovered = []workspace.MissingPair{}
			}
			if p.JSON {
				return p.PrintJSON(report)
			}
			p.Humanf("committed gaps: %d   covered by proposals: %d   uncovered: %d\n",
				len(report.MissingCommitted), len(report.CoveredByProposals), len(report.Uncovered))
			if s.ProposalsFillGaps {
				p.Humanf("%d proposal(s) fill all gaps — run: ahp plan\n", s.PairwiseProposals)
				return nil
			}
			if len(report.Uncovered) == 0 {
				p.Humanf("no missing pairs\n")
				return nil
			}
			headers := []string{"matrix", "left", "right"}
			rows := make([][]string, 0, len(report.Uncovered))
			for _, m := range report.Uncovered {
				rows = append(rows, []string{m.Matrix, m.Left, m.Right})
			}
			return p.PrintTable(headers, rows)
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	return c
}

func cmdPairSet() *cobra.Command {
	c := &cobra.Command{
		Use:   "set [matrix] [left] [right] [value]",
		Short: "Write one Saaty pairwise judgment",
		Args:  cobra.ExactArgs(4),
		RunE:  runPairSet,
	}
	addWorkspaceFlag(c)
	c.Flags().String("as", "", "proposal|committed (default: proposal for agents/non-TTY, committed for TTY)")
	c.Flags().String("status", "", "Alias for --as (deprecated)")
	c.Flags().String("note", "", "Note")
	return c
}

func cmdSetPairwise() *cobra.Command {
	c := &cobra.Command{
		Use:        "set-pairwise [matrix] [left] [right] [value]",
		Short:      "Deprecated alias for ahp pair set",
		Deprecated: "use \"ahp pair set\" instead",
		Args:       cobra.ExactArgs(4),
		RunE:       runPairSet,
	}
	addWorkspaceFlag(c)
	c.Flags().String("as", "", "proposal|committed")
	c.Flags().String("status", "", "proposal|committed (alias for --as)")
	c.Flags().String("note", "", "Note")
	return c
}

func runPairSet(cmd *cobra.Command, args []string) error {
	as, _ := cmd.Flags().GetString("as")
	status, _ := cmd.Flags().GetString("status")
	note, _ := cmd.Flags().GetString("note")

	chosen := as
	if chosen == "" {
		chosen = status
	}
	if chosen == "" {
		chosen = cliout.DefaultJudgmentStatus()
	}
	if chosen != "proposal" && chosen != "committed" {
		return cliout.NewExitError(cliout.ExitUsage, "status must be proposal or committed, got %q", chosen)
	}

	v, err := engine.ParseSaaty(args[3])
	if err != nil {
		return cliout.Wrap(cliout.ExitUsage, err)
	}
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	row := workspace.PairwiseRow{
		Matrix: args[0], Left: args[1], Right: args[2], Value: v, Status: chosen, Note: note,
	}
	if err := ws.UpsertPairwise(row); err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}

	p := cliout.FromCmd(cmd)
	payload := map[string]any{
		"matrix": row.Matrix, "left": row.Left, "right": row.Right,
		"value": row.Value, "value_label": engine.FormatSaaty(row.Value),
		"status": row.Status, "note": row.Note,
	}
	if p.JSON {
		return p.PrintJSON(payload)
	}
	fmt.Printf("pairwise %s %s/%s=%s (%s)\n", row.Matrix, row.Left, row.Right, engine.FormatSaaty(row.Value), row.Status)
	return nil
}

func cmdPairRepairs() *cobra.Command {
	c := &cobra.Command{
		Use:   "repairs",
		Short: "List CR repair suggestions for inconsistent matrices",
		RunE: func(cmd *cobra.Command, args []string) error {
			include, _ := cmd.Flags().GetBool("include-proposals")
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			s, err := ws.Status(include)
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			if p.JSON {
				out := make([]map[string]any, 0, len(s.Repairs))
				for _, h := range s.Repairs {
					out = append(out, map[string]any{
						"matrix": h.Matrix, "left": h.Left, "right": h.Right,
						"current": h.Current, "implied": h.Implied, "suggested": h.Suggested,
						"current_label":   engine.FormatSaaty(h.Current),
						"suggested_label": engine.FormatSaaty(h.Suggested),
						"error":           h.Error,
					})
				}
				return p.PrintJSON(out)
			}
			if len(s.Repairs) == 0 {
				p.Humanf("no repair hints\n")
				return nil
			}
			headers := []string{"matrix", "left", "right", "current", "suggested"}
			rows := make([][]string, 0, len(s.Repairs))
			for _, h := range s.Repairs {
				rows = append(rows, []string{
					h.Matrix, h.Left, h.Right,
					engine.FormatSaaty(h.Current), engine.FormatSaaty(h.Suggested),
				})
			}
			return p.PrintTable(headers, rows)
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	return c
}

func cmdPairAsk() *cobra.Command {
	c := &cobra.Command{
		Use:   "ask",
		Short: "Interactive Saaty walk over missing pairs (TTY only)",
		Long: `Prompt for each missing pair using the verbal Saaty scale.
Refuses to run when stdout is not a TTY or AHP_AGENT is set — agents use pair set --as proposal.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliout.AgentMode() || !cliout.IsStdoutTTY() {
				return cliout.NewExitError(cliout.ExitUsage,
					"pair ask requires an interactive TTY (unset AHP_AGENT; do not pipe). Use: ahp pair set … --as proposal")
			}
			include, _ := cmd.Flags().GetBool("include-proposals")
			as, _ := cmd.Flags().GetString("as")
			if as == "" {
				as = "committed"
			}
			if as != "proposal" && as != "committed" {
				return cliout.NewExitError(cliout.ExitUsage, "status must be proposal or committed")
			}

			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			s, err := ws.Status(include)
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			if len(s.Missing) == 0 {
				fmt.Println("no missing pairs")
				return nil
			}

			fmt.Println("Saaty scale: 1 equal, 3 moderate, 5 strong, 7 very strong, 9 extreme (or reciprocals 1/3…)")
			reader := bufio.NewReader(os.Stdin)
			written := 0
			for _, m := range s.Missing {
				fmt.Printf("\n%s: how much more important is %s than %s? [skip=enter] ", m.Matrix, m.Left, m.Right)
				line, err := reader.ReadString('\n')
				if err != nil {
					return cliout.Wrap(cliout.ExitIO, err)
				}
				line = strings.TrimSpace(line)
				if line == "" || strings.EqualFold(line, "skip") || strings.EqualFold(line, "q") {
					continue
				}
				v, err := engine.ParseSaaty(line)
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
					continue
				}
				if err := ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: m.Matrix, Left: m.Left, Right: m.Right, Value: v, Status: as,
				}); err != nil {
					return cliout.Wrap(cliout.ExitIO, err)
				}
				fmt.Printf("  saved %s (%s)\n", engine.FormatSaaty(v), as)
				written++
			}
			fmt.Printf("wrote %d judgment(s)\n", written)
			return nil
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().String("as", "committed", "proposal|committed for answers")
	return c
}
