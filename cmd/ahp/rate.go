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
		Short: "Opt-in: turn numeric attributes into Saaty pairwise proposals",
		Long: `Bridge objective attributes → judgment proposals (never commits).

For criterion C, reads numeric rows in data/attributes.csv and writes
proposal judgments on matrix alt:C using ratio→nearest-Saaty mapping.

  --prefer lower   smaller attribute wins (price / risk weeks)
  --prefer higher  larger attribute wins (quality scores)
  --stretch        affine-shift so tight bands (e.g. 8.4 vs 9.5) discriminate

Non-positive attributes (e.g. amenities=0) are allowed via an automatic affine shift.
Use --refresh to demote committed pairs back to proposals when attributes change
(never auto-commits). Alias of: ahp pair suggest-from-attributes`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runSuggestFromAttributes,
	}
	addSuggestFromAttributesFlags(c)
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
