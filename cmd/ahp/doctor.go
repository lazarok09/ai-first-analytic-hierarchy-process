package main

import (
	"fmt"
	"io"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdDoctor() *cobra.Command {
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose workspace layout, schema, missing pairs, and CR hotspots",
		Long: `Check the AHP workspace like cargo/brew doctor.

Reports missing ahp.toml / data files, unknown ids in pairwise or attributes,
empty or incomplete matrices, CR > 0.10 hotspots with repair hints, and
pending proposal rows.

Exit codes (docs/ROADMAP.md):
  0  OK / ready
  2  incomplete (structure or missing pairs)
  3  inconsistent (CR above threshold)
  4  proposals pending (only with --strict)
  5  I/O failure`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runDoctor,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().Bool("strict", false, "Fail with exit 4 when proposal judgments are pending")
	c.Flags().Bool("purchase", false, "Audit price provenance, FX smell, and alt:value direction")
	return c
}

// cmdValidate keeps the old verb as a thin deprecated alias of doctor.
func cmdValidate() *cobra.Command {
	c := &cobra.Command{
		Use:           "validate",
		Short:         "Deprecated alias for ahp doctor",
		Long:          "Deprecated: use `ahp doctor`. Same diagnostics and exit codes.",
		Deprecated:    "use \"ahp doctor\" instead",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runDoctor,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().Bool("strict", false, "Fail with exit 4 when proposal judgments are pending")
	c.Flags().Bool("purchase", false, "Audit price provenance, FX smell, and alt:value direction")
	return c
}

func runDoctor(cmd *cobra.Command, args []string) error {
	include, _ := cmd.Flags().GetBool("include-proposals")
	strict, _ := cmd.Flags().GetBool("strict")
	purchase, _ := cmd.Flags().GetBool("purchase")

	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.NewExitError(cliout.ExitIncomplete, "%s", err.Error())
	}

	report, err := ws.Doctor(workspace.DoctorOptions{
		IncludeProposals: include,
		Strict:           strict,
		Purchase:         purchase,
	})
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}

	p := cliout.FromCmd(cmd)
	if p.JSON {
		if err := p.PrintJSON(report); err != nil {
			return cliout.Wrap(cliout.ExitIO, err)
		}
	} else {
		printDoctorHuman(cmd.OutOrStdout(), report)
	}

	if report.ExitCode == cliout.ExitOK {
		return nil
	}
	// Findings already printed; exit code only.
	return &cliout.ExitError{Code: report.ExitCode}
}

func printDoctorHuman(w io.Writer, report *workspace.DoctorReport) {
	title := report.Title
	if title == "" {
		title = "(untitled)"
	}
	fmt.Fprintf(w, "AHP doctor  %s  (%s)\n", title, report.Workspace)
	fmt.Fprintf(w, "complete=%v  consistent=%v  proposals=%d  exit=%d\n",
		report.Complete, report.Consistent, report.Proposals, report.ExitCode)
	if len(report.Findings) == 0 {
		fmt.Fprintf(w, "ok: workspace looks ready\n")
		return
	}
	for _, f := range report.Findings {
		line := fmt.Sprintf("%s: %s", f.Code, f.Message)
		if f.Fix != "" {
			line += "  →  " + f.Fix
		}
		fmt.Fprintln(w, line)
	}
}
