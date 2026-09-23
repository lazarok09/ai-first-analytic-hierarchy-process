package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdRate() *cobra.Command {
	c := &cobra.Command{
		Use:   "rate",
		Short: "Opt-in: attributes → Saaty proposals or sum-norm preview",
		Long: `Bridge objective attributes → judgments/previews (never commits Saaty).

  --mode=saaty (default)  ratio→nearest Saaty 1–9 proposals on alt:<c> (lossy)
  --mode=sum-norm         Santos-style sum-normalize for --criterion (preview only)

For Saaty mode:
  --prefer lower|higher, --stretch, --refresh as before.

Alias of Saaty mode: ahp pair suggest-from-attributes`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runRate,
	}
	addSuggestFromAttributesFlags(c)
	c.Flags().String("mode", "saaty", "saaty|sum-norm")
	return c
}

func cmdPairSuggestFromAttributes() *cobra.Command {
	c := &cobra.Command{
		Use:   "suggest-from-attributes",
		Short: "Propose alt pairwise from numeric attributes (never commits)",
		Long: `Read attributes for --criterion and write status=proposal Saaty pairs
on alt:<criterion>. By default committed pairs are left untouched.
Use --refresh to overwrite them as proposals (demote); --dry-run to preview.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runSuggestFromAttributes,
	}
	addSuggestFromAttributesFlags(c)
	return c
}

func addSuggestFromAttributesFlags(c *cobra.Command) {
	addWorkspaceFlag(c)
	c.Flags().String("criterion", "", "Criterion id whose attributes to use (required)")
	c.Flags().String("prefer", "higher", "higher|lower — which attribute direction wins")
	c.Flags().Bool("dry-run", false, "Preview proposals without writing pairwise.csv")
	c.Flags().Bool("refresh", false, "Rewrite matching pairs as proposals even if committed")
	c.Flags().Bool("stretch", false, "Affine-shift attributes so tight numeric bands discriminate on Saaty")
	_ = c.MarkFlagRequired("criterion")
}

func runRate(cmd *cobra.Command, args []string) error {
	mode, _ := cmd.Flags().GetString("mode")
	switch mode {
	case "saaty", "":
		return runSuggestFromAttributes(cmd, args)
	case "sum-norm":
		return runRateSumNorm(cmd)
	default:
		return cliout.NewExitError(cliout.ExitUsage, "unknown --mode %q (saaty|sum-norm)", mode)
	}
}

func runRateSumNorm(cmd *cobra.Command) error {
	crit, _ := cmd.Flags().GetString("criterion")
	preferRaw, _ := cmd.Flags().GetString("prefer")
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	// Ensure prefer is available for this criterion (flag overrides for preview).
	prefer := engine.PreferDirection(preferRaw)
	if prefer != engine.PreferHigher && prefer != engine.PreferLower {
		return cliout.NewExitError(cliout.ExitUsage, "prefer must be higher or lower")
	}
	altIDs, cols, _, err := ws.BuildAbsoluteColumnInputs()
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	var col *engine.AbsoluteColumnInput
	for i := range cols {
		if cols[i].CriterionID == crit {
			col = &cols[i]
			break
		}
	}
	if col == nil {
		// Build a one-off column from attributes using flag prefer.
		attrs, err := ws.Attributes()
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		if len(altIDs) < 2 {
			eligible, _, err := ws.EligibleAlternativeIDs()
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			altIDs = eligible
		}
		values := map[string]float64{}
		for _, id := range altIDs {
			found := false
			for _, a := range attrs {
				if a.CriterionID != crit || a.AlternativeID != id {
					continue
				}
				v, err := engine.ParseNumericAttribute(a.Value)
				if err != nil {
					return cliout.Wrap(cliout.ExitUsage, err)
				}
				values[id] = v
				found = true
				break
			}
			if !found {
				return cliout.NewExitError(cliout.ExitIncomplete, "criterion %s: missing numeric attribute for %s", crit, id)
			}
		}
		col = &engine.AbsoluteColumnInput{CriterionID: crit, Prefer: prefer, Values: values}
	} else {
		col.Prefer = prefer // flag overrides for preview
	}
	m, err := engine.BuildAbsoluteMatrix(altIDs, []engine.AbsoluteColumnInput{*col})
	if err != nil {
		return cliout.Wrap(cliout.ExitUsage, err)
	}
	norm := m.NormByCriterion[crit]
	if p.JSON {
		return p.PrintJSON(map[string]any{
			"mode": "sum-norm", "criterion": crit, "prefer": prefer,
			"norm": norm, "caveat": "Preview only — not Saaty pairwise; no CR.",
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "sum-norm preview  criterion=%s  prefer=%s\n", crit, prefer)
	fmt.Fprintln(cmd.OutOrStdout(), "Preview only — not Saaty pairwise; no CR.")
	for _, id := range engine.RankScores(norm) {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %.4f\n", id, norm[id])
	}
	return nil
}

func runSuggestFromAttributes(cmd *cobra.Command, args []string) error {
	crit, _ := cmd.Flags().GetString("criterion")
	preferRaw, _ := cmd.Flags().GetString("prefer")
	dry, _ := cmd.Flags().GetBool("dry-run")
	refresh, _ := cmd.Flags().GetBool("refresh")
	stretch, _ := cmd.Flags().GetBool("stretch")
	p := cliout.FromCmd(cmd)
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	prefer := engine.PreferDirection(preferRaw)
	res, err := ws.SuggestFromAttributes(workspace.SuggestFromAttributesOptions{
		CriterionID: crit,
		Prefer:      prefer,
		DryRun:      dry,
		Refresh:     refresh,
		Stretch:     stretch,
	})
	if err != nil {
		return cliout.Wrap(cliout.ExitUsage, err)
	}
	if p.JSON {
		return p.PrintJSON(res)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Attribute→proposal  %s\n", res.Summary)
	fmt.Fprintf(cmd.OutOrStdout(), "Matrix  %s   prefer  %s\n", res.Matrix, res.Prefer)
	for _, s := range res.Suggestions {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s vs %s  %s   (%s)\n", s.Left, s.Right, s.ValueLabel, s.Note)
	}
	if !dry && res.Written > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nNext:  ahp plan    # preview before apply\n")
	}
	return nil
}
