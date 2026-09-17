package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/spf13/cobra"
)

func cmdApply() *cobra.Command {
	c := &cobra.Command{
		Use:   "apply",
		Short: "Commit proposal judgments (terraform-style)",
		Long: `Flip proposal → committed in data/pairwise.csv.

  ahp apply --dry-run   same as ahp plan (no writes)
  ahp apply -y          commit without interactive confirm
  ahp apply             confirm on TTY; agents/non-TTY require -y`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runApply,
	}
	addWorkspaceFlag(c)
	c.Flags().Bool("dry-run", false, "Preview only (same as ahp plan)")
	c.Flags().BoolP("yes", "y", false, "Skip confirmation")
	c.Flags().String("matrix", "", "Optional matrix filter")
	c.Flags().String("left", "", "Optional left id")
	c.Flags().String("right", "", "Optional right id")
	return c
}

func cmdCommitProposals() *cobra.Command {
	c := &cobra.Command{
		Use:        "commit-proposals",
		Short:      "Deprecated alias for ahp apply -y",
		Deprecated: "use \"ahp apply\" instead",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Legacy verb always committed without confirm.
			if err := cmd.Flags().Set("yes", "true"); err != nil {
				return err
			}
			return runApply(cmd, args)
		},
	}
	addWorkspaceFlag(c)
	c.Flags().Bool("dry-run", false, "Preview only")
	c.Flags().BoolP("yes", "y", false, "Skip confirmation")
	c.Flags().String("matrix", "", "Optional matrix filter")
	c.Flags().String("left", "", "Optional left id")
	c.Flags().String("right", "", "Optional right id")
	return c
}

func runApply(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	matrix, _ := cmd.Flags().GetString("matrix")
	left, _ := cmd.Flags().GetString("left")
	right, _ := cmd.Flags().GetString("right")
	if (left == "") != (right == "") {
		return cliout.NewExitError(cliout.ExitUsage, "provide both --left and --right, or neither")
	}

	if dryRun {
		return runPlan(cmd, args)
	}

	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}

	plan, err := ws.Plan()
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	if plan.Proposals == 0 && matrix == "" && left == "" {
		p := cliout.FromCmd(cmd)
		if p.JSON {
			return p.PrintJSON(map[string]any{"committed": 0, "message": "no proposals"})
		}
		fmt.Println("committed 0 proposal(s)")
		return nil
	}

	needConfirm := !yes && cliout.IsStdoutTTY() && !cliout.AgentMode()
	if !yes && !needConfirm {
		// Non-TTY / agent without -y: refuse silent commit.
		return cliout.NewExitError(cliout.ExitUsage,
			"refusing to apply without -y when not an interactive TTY (agents must pass -y)")
	}
	if needConfirm {
		fmt.Fprintf(os.Stderr, "Apply will commit %d proposal(s). Continue? [y/N] ", plan.Proposals)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			return cliout.NewExitError(cliout.ExitUsage, "apply cancelled")
		}
	}

	updated, err := ws.CommitProposals(matrix, left, right)
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	p := cliout.FromCmd(cmd)
	if p.JSON {
		return p.PrintJSON(map[string]any{
			"committed": len(updated),
			"rows":      updated,
		})
	}
	fmt.Printf("committed %d proposal(s)\n", len(updated))
	return nil
}
