package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdGet() *cobra.Command {
	c := &cobra.Command{
		Use:   "get [resource]",
		Short: "List resources: ranking | pairs | criteria | alternatives | matrices",
		Long: `kubectl-style inspect (table on TTY, JSON when --json / non-TTY).

  ahp get ranking
  ahp get pairs --status proposal
  ahp get criteria
  ahp get alternatives
  ahp get matrices`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runGet,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().String("status", "all", "For pairs: all|proposal|committed")
	c.Flags().StringP("output", "o", "", "json|table (optional; --json still works)")
	return c
}

func runGet(cmd *cobra.Command, args []string) error {
	applyOutputFlag(cmd)
	p := cliout.FromCmd(cmd)
	include, _ := cmd.Flags().GetBool("include-proposals")
	statusFilter, _ := cmd.Flags().GetString("status")

	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}

	switch strings.ToLower(args[0]) {
	case "ranking", "rankings":
		result, err := ws.Compute(include)
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		if p.JSON {
			return p.PrintJSON(result.Ranking)
		}
		rows := make([][]string, 0, len(result.Ranking))
		for _, r := range result.Ranking {
			rows = append(rows, []string{
				strconv.Itoa(r.Rank), r.ID, r.Name, fmt.Sprintf("%.4f", r.Weight),
			})
		}
		return p.PrintTable([]string{"rank", "id", "name", "weight"}, rows)

	case "pairs", "pairwise":
		items, err := ws.Pairwise()
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		var filtered []workspace.PairwiseRow
		for _, row := range items {
			switch statusFilter {
			case "", "all":
				filtered = append(filtered, row)
			case "proposal", "committed":
				if row.Status == statusFilter {
					filtered = append(filtered, row)
				}
			default:
				return cliout.NewExitError(cliout.ExitUsage, "status must be all|proposal|committed")
			}
		}
		if p.JSON {
			return p.PrintJSON(filtered)
		}
		rows := make([][]string, 0, len(filtered))
		for _, row := range filtered {
			rows = append(rows, []string{
				row.Matrix, row.Left, row.Right, engine.FormatSaaty(row.Value), row.Status,
			})
		}
		return p.PrintTable([]string{"matrix", "left", "right", "value", "status"}, rows)

	case "criteria", "criterion":
		items, err := ws.Criteria()
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		if p.JSON {
			return p.PrintJSON(items)
		}
		rows := make([][]string, 0, len(items))
		for _, c := range items {
			rows = append(rows, []string{c.ID, c.Name, c.ParentID})
		}
		return p.PrintTable([]string{"id", "name", "parent_id"}, rows)

	case "alternatives", "alternative", "alts":
		items, err := ws.Alternatives()
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		if p.JSON {
			return p.PrintJSON(items)
		}
		rows := make([][]string, 0, len(items))
		for _, a := range items {
			rows = append(rows, []string{a.ID, a.Name})
		}
		return p.PrintTable([]string{"id", "name"}, rows)

	case "matrices", "matrix":
		result, err := ws.Compute(include)
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		type row struct {
			Key        string   `json:"key"`
			Complete   bool     `json:"complete"`
			Consistent bool     `json:"consistent"`
			CR         *float64 `json:"cr,omitempty"`
		}
		keys := make([]string, 0, len(result.Matrices))
		for k := range result.Matrices {
			keys = append(keys, k)
		}
		// stable-ish: reuse describe sort via simple insertion
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		out := make([]row, 0, len(keys))
		tableRows := make([][]string, 0, len(keys))
		for _, k := range keys {
			m := result.Matrices[k]
			out = append(out, row{Key: k, Complete: m.Complete, Consistent: m.Consistent, CR: m.CR})
			cr := "—"
			if m.CR != nil {
				cr = fmt.Sprintf("%.3f", *m.CR)
			}
			tableRows = append(tableRows, []string{k, strconv.FormatBool(m.Complete), strconv.FormatBool(m.Consistent), cr})
		}
		if p.JSON {
			return p.PrintJSON(out)
		}
		return p.PrintTable([]string{"key", "complete", "consistent", "cr"}, tableRows)

	default:
		return cliout.NewExitError(cliout.ExitUsage,
			"unknown resource %q — use ranking|pairs|criteria|alternatives|matrices", args[0])
	}
}

func cmdDescribe() *cobra.Command {
	c := &cobra.Command{
		Use:   "describe [type] [name]",
		Short: "Describe a resource in detail (matrix …)",
		Long: `Show a detailed view of one resource.

  ahp describe matrix criteria
  ahp describe matrix alt:cost`,
		Args:          cobra.ExactArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runDescribe,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().StringP("output", "o", "", "json|table")
	return c
}

func runDescribe(cmd *cobra.Command, args []string) error {
	applyOutputFlag(cmd)
	p := cliout.FromCmd(cmd)
	include, _ := cmd.Flags().GetBool("include-proposals")
	kind := strings.ToLower(args[0])
	name := args[1]

	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}

	switch kind {
	case "matrix", "matrices":
		result, err := ws.Compute(include)
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		m, ok := result.Matrices[name]
		if !ok {
			return cliout.NewExitError(cliout.ExitUsage, "unknown matrix %q — try: ahp get matrices", name)
		}
		payload := map[string]any{
			"matrix": name, "ids": m.IDs, "names": m.Names,
			"weights": m.Weights, "complete": m.Complete, "consistent": m.Consistent,
			"lambda_max": m.LambdaMax, "ci": m.CI, "cr": m.CR,
			"missing": m.Missing, "repairs": m.Repairs, "matrix_values": m.Matrix,
		}
		if p.JSON {
			return p.PrintJSON(payload)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Matrix  %s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "complete=%v  consistent=%v  CR=%s\n",
			m.Complete, m.Consistent, fmtOptCR(m.CR))
		fmt.Fprintln(cmd.OutOrStdout(), "ids:", strings.Join(m.IDs, ", "))
		if len(m.Weights) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "weights:")
			for _, id := range m.IDs {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  %.4f\n", id, m.Weights[id])
			}
		}
		if len(m.Missing) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "missing: %d pair(s) — ahp pair missing\n", len(m.Missing))
		}
		if len(m.Repairs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "repairs: %d hint(s) — ahp pair repairs\n", len(m.Repairs))
		}
		return nil

	default:
		return cliout.NewExitError(cliout.ExitUsage, "unknown type %q — use: describe matrix <key>", kind)
	}
}

func applyOutputFlag(cmd *cobra.Command) {
	o, _ := cmd.Flags().GetString("output")
	switch strings.ToLower(o) {
	case "json":
		_ = cmd.Flags().Set("json", "true")
	case "table":
		_ = cmd.Flags().Set("json", "false")
	}
}
