package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdStatus() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Home screen: completeness, CR, proposals, ranking peek, next step",
		Long: `Summarize the workspace like git status: matrix completeness, consistency,
pending proposals, a ranking peek, and exactly one recommended next command.

Exit codes: human default is 0 after printing. With --check, readiness is:
  0 ready, 2 incomplete, 3 inconsistent, 4 proposals pending.
I/O failures exit 5. Agents get JSON via --json or when stdout is not a TTY.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			include, _ := cmd.Flags().GetBool("include-proposals")
			check, _ := cmd.Flags().GetBool("check")
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
				if err := p.PrintJSON(s); err != nil {
					return cliout.Wrap(cliout.ExitIO, err)
				}
			} else {
				printStatusHuman(p.Out, s)
			}

			if check && s.ExitCode != 0 {
				// Status already printed; exit code only (no duplicate message).
				return &cliout.ExitError{Code: s.ExitCode}
			}
			return nil
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().Bool("check", false, "Exit non-zero when incomplete, inconsistent, or proposals pending")
	return c
}

func printStatusHuman(w io.Writer, s *workspace.StatusSummary) {
	if w == nil {
		w = os.Stdout
	}
	path := displayWorkspacePath(s.Workspace)
	fmt.Fprintf(w, "%s  ·  %s\n", s.Title, path)

	matrices := "incomplete ✗"
	if s.Complete {
		matrices = "complete ✓"
	}
	consistency := "—"
	if s.Complete {
		if s.Consistent {
			consistency = "CR≤0.10 ✓"
		} else {
			consistency = "CR>0.10 ✗"
		}
	}
	proposals := "none"
	if s.PairwiseProposals > 0 {
		proposals = fmt.Sprintf("%d pending", s.PairwiseProposals)
	}
	fmt.Fprintf(w, "Matrices  %s   Consistency  %s   Proposals  %s\n", matrices, consistency, proposals)

	switch {
	case s.RankingMode == workspace.RankingModeEqualFallback:
		fmt.Fprintln(w, "Ranking   (hidden — equal-weight fallback; matrices incomplete)")
	case len(s.Ranking) > 0:
		parts := make([]string, 0, len(s.Ranking))
		limit := 5
		if len(s.Ranking) < limit {
			limit = len(s.Ranking)
		}
		for i := 0; i < limit; i++ {
			r := s.Ranking[i]
			parts = append(parts, fmt.Sprintf("%d. %s %.2f", r.Rank, r.Name, r.Weight))
		}
		fmt.Fprintf(w, "Ranking   %s\n", strings.Join(parts, "   "))
	default:
		fmt.Fprintln(w, "Ranking   (none yet)")
	}

	if s.ProposalsFillGaps {
		fmt.Fprintf(w, "note: %d proposal(s) fill all pairwise gaps — run ahp plan\n", s.PairwiseProposals)
	}

	for _, warn := range s.Warnings {
		fmt.Fprintf(w, "warning: %s\n", warn)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Next:  %s", s.Next.Command)
	if s.Next.Reason != "" {
		fmt.Fprintf(w, "  # %s", s.Next.Reason)
	}
	fmt.Fprintln(w)
	for _, h := range s.Next.Hints {
		fmt.Fprintf(w, "       %s", h.Command)
		if h.Reason != "" {
			fmt.Fprintf(w, "  # %s", h.Reason)
		}
		fmt.Fprintln(w)
	}
}

func displayWorkspacePath(root string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return root
	}
	rel, err := filepath.Rel(cwd, root)
	if err != nil || strings.HasPrefix(rel, "..") {
		return root
	}
	if rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}
