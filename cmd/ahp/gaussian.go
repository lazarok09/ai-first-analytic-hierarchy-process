package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdGaussian() *cobra.Command {
	c := &cobra.Command{
		Use:   "gaussian",
		Short: "Comparative AHP-Gaussian ranking from attributes (σ/μ criterion weights)",
		Long: `Santos-style absolute measurement + Gaussian factor reweight (comparative only).

Requires numeric attributes for eligible alternatives on leaf criteria and
direction via: ahp constrain <c> --prefer higher|lower

Does not replace Saaty pairwise ranking. Writes output/gaussian.json.
Never claims a Saaty CR for these scores.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runGaussian,
	}
	addWorkspaceFlag(c)
	c.Flags().Bool("write", true, "Write output/gaussian.json")
	return c
}

func cmdAbsolute() *cobra.Command {
	c := &cobra.Command{
		Use:   "absolute",
		Short: "Sum-normalize attributes; score with criteria weights when available (hybrid)",
		Long: `Build the decision matrix from attributes.csv (cost criteria inverted),
sum-normalize per criterion, and if criteria pairwise leaf weights exist,
score alternatives (hybrid absolute measurement).

Writes output/absolute.json. Comparative / preview — does not write pairwise.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runAbsolute,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().Bool("write", true, "Write output/absolute.json")
	return c
}

func runGaussian(cmd *cobra.Command, args []string) error {
	writeOut, _ := cmd.Flags().GetBool("write")
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	res, err := ws.Gaussian()
	if err != nil {
		return cliout.Wrap(cliout.ExitIncomplete, err)
	}
	var path string
	if writeOut {
		path, err = ws.WriteGaussianOutputs(res)
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
	}
	if p.JSON {
		payload := map[string]any{
			"workspace": res.Workspace,
			"title":     res.Title,
			"method":    res.Method,
			"caveat":    res.Caveat,
			"ranking":   res.Ranking,
			"warnings":  res.Warnings,
			"result":    res.Result,
		}
		if path != "" {
			payload["gaussian_json"] = path
		}
		return p.PrintJSON(payload)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "AHP-Gaussian (comparative)  %s\n", res.Title)
	fmt.Fprintln(cmd.OutOrStdout(), res.Caveat)
	for _, w := range res.Warnings {
		fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
	}
	if res.Result != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "factors:")
		for _, f := range res.Result.Factors {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s  μ=%.4f  σ=%.4f  f=%.4f  w=%.4f\n",
				f.CriterionID, f.Mean, f.SD, f.Factor, f.Weight)
		}
	}
	fmt.Fprintln(cmd.OutOrStdout(), "ranking:")
	for _, r := range res.Ranking {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s  %.4f\n", r.Rank, r.Name, r.Weight)
	}
	if path != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "gaussian:", path)
	}
	return nil
}

func runAbsolute(cmd *cobra.Command, args []string) error {
	include, _ := cmd.Flags().GetBool("include-proposals")
	writeOut, _ := cmd.Flags().GetBool("write")
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	res, err := ws.Absolute(include)
	if err != nil {
		return cliout.Wrap(cliout.ExitIncomplete, err)
	}
	var path string
	if writeOut {
		path, err = ws.WriteAbsoluteOutputs(res)
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
	}
	if p.JSON {
		payload := map[string]any{
			"workspace":          res.Workspace,
			"title":              res.Title,
			"method":             res.Method,
			"caveat":             res.Caveat,
			"matrix":             res.Matrix,
			"criterion_weights":  res.CriterionWeights,
			"scores":             res.Scores,
			"ranking":            res.Ranking,
			"warnings":           res.Warnings,
		}
		if path != "" {
			payload["absolute_json"] = path
		}
		return p.PrintJSON(payload)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Absolute measurement (%s)  %s\n", res.Method, res.Title)
	fmt.Fprintln(cmd.OutOrStdout(), res.Caveat)
	for _, w := range res.Warnings {
		fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
	}
	if res.Matrix != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "criteria: %d  alternatives: %d\n",
			len(res.Matrix.Columns), len(res.Matrix.AlternativeIDs))
	}
	if len(res.Ranking) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "ranking:")
		for _, r := range res.Ranking {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s  %.4f\n", r.Rank, r.Name, r.Weight)
		}
	}
	if path != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "absolute:", path)
	}
	return nil
}

// printCompareExtras prints comparative rankings after Saaty compute (JSON already written by AttachComparative).
func printCompareExtras(cmd *cobra.Command, result *workspace.ComputeResult, method string) {
	p := cliout.FromCmd(cmd)
	if p.JSON || method == "" || method == "saaty" {
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), "--- comparative (no Saaty CR on these scores) ---")
	if result.Absolute != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "absolute (%s):\n", result.Absolute.Method)
		if len(result.Absolute.Ranking) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  (matrix only — no hybrid ranking)")
		}
		for _, r := range result.Absolute.Ranking {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s  %.4f\n", r.Rank, r.Name, r.Weight)
		}
	}
	if result.Gaussian != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "gaussian:")
		for _, r := range result.Gaussian.Ranking {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s  %.4f\n", r.Rank, r.Name, r.Weight)
		}
	}
}