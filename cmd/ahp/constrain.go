package main

import (
	"fmt"
	"strconv"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdConstrain() *cobra.Command {
	c := &cobra.Command{
		Use:   "constrain [criterion]",
		Short: "Set attribute eligibility band (min/max/unit) for a criterion",
		Long: `Write data/constraints.csv so out-of-band alternatives are excluded from
synthesis and flagged by ahp doctor.

  ahp constrain value --min 200 --max 400 --unit BRL --prefer lower

Omit flags to list current constraints. Prefer is used by purchase integrity.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runConstrain,
	}
	addWorkspaceFlag(c)
	c.Flags().String("min", "", "Minimum allowed attribute value")
	c.Flags().String("max", "", "Maximum allowed attribute value")
	c.Flags().String("unit", "", "Required attribute unit (e.g. BRL)")
	c.Flags().String("prefer", "", "higher|lower for purchase direction checks")
	c.Flags().String("note", "", "Optional note")
	return c
}

func runConstrain(cmd *cobra.Command, args []string) error {
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	p := cliout.FromCmd(cmd)

	if len(args) == 0 {
		items, err := ws.Constraints()
		if err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
		if p.JSON {
			return p.PrintJSON(items)
		}
		if len(items) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no constraints")
			return nil
		}
		for _, c := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "%s  min=%s max=%s unit=%s prefer=%s\n",
				c.CriterionID, fmtOptF(c.Min), fmtOptF(c.Max), c.Unit, c.Prefer)
		}
		return nil
	}

	crit := args[0]
	minRaw, _ := cmd.Flags().GetString("min")
	maxRaw, _ := cmd.Flags().GetString("max")
	unit, _ := cmd.Flags().GetString("unit")
	prefer, _ := cmd.Flags().GetString("prefer")
	note, _ := cmd.Flags().GetString("note")

	item := workspace.Constraint{CriterionID: crit, Unit: unit, Prefer: prefer, Note: note}
	if minRaw != "" {
		v, err := strconv.ParseFloat(minRaw, 64)
		if err != nil {
			return cliout.Wrap(cliout.ExitUsage, fmt.Errorf("min: %w", err))
		}
		item.Min = &v
	}
	if maxRaw != "" {
		v, err := strconv.ParseFloat(maxRaw, 64)
		if err != nil {
			return cliout.Wrap(cliout.ExitUsage, fmt.Errorf("max: %w", err))
		}
		item.Max = &v
	}
	if item.Min == nil && item.Max == nil && unit == "" {
		return cliout.NewExitError(cliout.ExitUsage, "provide at least --min, --max, or --unit")
	}

	criteria, err := ws.Criteria()
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	found := false
	for _, c := range criteria {
		if c.ID == crit {
			found = true
			break
		}
	}
	if !found {
		return cliout.Wrap(cliout.ExitUsage, fmt.Errorf("unknown criterion %q", crit))
	}

	if err := ws.UpsertConstraint(item); err != nil {
		return cliout.Wrap(cliout.ExitUsage, err)
	}
	if p.JSON {
		return p.PrintJSON(item)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "constraint  %s  min=%s max=%s unit=%s prefer=%s\n",
		item.CriterionID, fmtOptF(item.Min), fmtOptF(item.Max), item.Unit, item.Prefer)
	fmt.Fprintln(cmd.OutOrStdout(), "Next:  ahp doctor   # then ahp compute / status")
	return nil
}

func fmtOptF(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}
