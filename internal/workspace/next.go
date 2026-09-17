package workspace

import (
	"fmt"
	"os"

	"github.com/lazarok09/ahp-method/internal/cliout"
)

// NextAction is the single recommended follow-up for status / ahp next.
type NextAction struct {
	// Kind is structure | incomplete | inconsistent | proposals | ready.
	Kind string `json:"kind"`
	// Command is a full CLI invocation using verbs that exist today.
	Command string `json:"command"`
	// Reason explains why this command is recommended.
	Reason string `json:"reason"`
	// Hints are optional secondary suggestions.
	Hints []NextHint `json:"hints,omitempty"`
}

// NextHint is an optional secondary next step.
type NextHint struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

// RecommendNext picks exactly one primary next CLI verb from a StatusSummary.
//
// Priority (first match wins):
//  1. structure gaps (no criteria / no alternatives)
//  2. incomplete matrices → pair missing / pair set
//  3. inconsistent CR → pair repairs (doctor for full scan)
//  4. proposals pending → plan then apply
//  5. ready → compute, or open if report already exists
func RecommendNext(s *StatusSummary) NextAction {
	if s == nil {
		return NextAction{
			Kind:    "structure",
			Command: "ahp init",
			Reason:  "no workspace status available",
		}
	}

	switch {
	case s.Criteria == 0:
		return NextAction{
			Kind:    "structure",
			Command: "ahp add-criterion",
			Reason:  "no criteria yet",
			Hints: []NextHint{{
				Command: "ahp add-alternative",
				Reason:  "add alternatives once criteria exist",
			}},
		}
	case s.Alternatives == 0:
		return NextAction{
			Kind:    "structure",
			Command: "ahp add-alternative",
			Reason:  "no alternatives yet",
		}
	case !s.Complete || len(s.Missing) > 0:
		n := len(s.Missing)
		reason := "fill missing pairwise judgments"
		if n > 0 {
			reason = fmt.Sprintf("fill %d missing pairwise judgment(s)", n)
		}
		return NextAction{
			Kind:    "incomplete",
			Command: "ahp pair missing",
			Reason:  reason,
			Hints: []NextHint{{
				Command: "ahp pair set",
				Reason:  "write judgments (agents: --as proposal)",
			}},
		}
	case !s.Consistent:
		return NextAction{
			Kind:    "inconsistent",
			Command: "ahp pair repairs",
			Reason:  "review CR repair suggestions (CR > 0.10)",
			Hints: []NextHint{{
				Command: "ahp doctor",
				Reason:  "full workspace diagnostics",
			}},
		}
	case s.PairwiseProposals > 0:
		n := s.PairwiseProposals
		return NextAction{
			Kind:    "proposals",
			Command: "ahp plan",
			Reason:  fmt.Sprintf("preview committing %d proposal(s)", n),
			Hints: []NextHint{{
				Command: "ahp apply -y",
				Reason:  "commit proposals after review",
			}},
		}
	case reportExists(s.ReportHTML):
		return NextAction{
			Kind:    "ready",
			Command: "ahp open",
			Reason:  "workspace ready — open the HTML report",
			Hints: []NextHint{{
				Command: "ahp compute",
				Reason:  "re-solve and refresh outputs",
			}},
		}
	default:
		return NextAction{
			Kind:    "ready",
			Command: "ahp compute",
			Reason:  "workspace ready — solve and write outputs",
			Hints: []NextHint{{
				Command: "ahp open",
				Reason:  "recompute and print report.html path",
			}},
		}
	}
}

// NextActionFromStatus is an alias for RecommendNext.
func NextActionFromStatus(s *StatusSummary) NextAction {
	return RecommendNext(s)
}

// ReadinessExit returns the ROADMAP exit code for a status snapshot.
// Used by `ahp status --check` (human default remains 0 after print).
func ReadinessExit(s *StatusSummary) int {
	if s == nil {
		return cliout.ExitIO
	}
	if s.Criteria == 0 || s.Alternatives == 0 || !s.Complete || len(s.Missing) > 0 {
		return cliout.ExitIncomplete
	}
	if !s.Consistent {
		return cliout.ExitInconsistent
	}
	if s.PairwiseProposals > 0 {
		return cliout.ExitProposals
	}
	return cliout.ExitOK
}

func reportExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
