package cliout

import (
	"os"
	"testing"
)

func TestAgentModeEnv(t *testing.T) {
	origTTY := isTTYFn
	t.Cleanup(func() {
		isTTYFn = origTTY
		_ = os.Unsetenv("AHP_AGENT")
	})

	isTTYFn = func(*os.File) bool { return true }
	_ = os.Unsetenv("AHP_AGENT")
	if AgentMode() {
		t.Fatal("TTY + no env → not agent")
	}
	if DefaultJudgmentStatus() != "committed" {
		t.Fatalf("status=%s", DefaultJudgmentStatus())
	}

	t.Setenv("AHP_AGENT", "1")
	if !AgentMode() || DefaultJudgmentStatus() != "proposal" {
		t.Fatal("AHP_AGENT=1 → proposal")
	}

	_ = os.Unsetenv("AHP_AGENT")
	isTTYFn = func(*os.File) bool { return false }
	if !AgentMode() || DefaultJudgmentStatus() != "proposal" {
		t.Fatal("non-TTY → proposal")
	}
}
