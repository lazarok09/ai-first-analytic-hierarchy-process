package cliout

import (
	"os"
	"strings"
)

// AgentMode is true when the caller looks agent-driven:
// AHP_AGENT=1/true, or stdout is not a TTY (piped / redirected).
func AgentMode() bool {
	v := strings.TrimSpace(os.Getenv("AHP_AGENT"))
	if v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		return true
	}
	return !IsStdoutTTY()
}

// DefaultJudgmentStatus returns the default pairwise status for writes.
// Agents and non-TTY default to proposal; interactive humans default to committed.
func DefaultJudgmentStatus() string {
	if AgentMode() {
		return "proposal"
	}
	return "committed"
}
