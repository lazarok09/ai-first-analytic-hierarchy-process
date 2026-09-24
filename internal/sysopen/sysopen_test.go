package sysopen

import (
	"os"
	"strings"
	"testing"
)

func TestCandidatesLinuxIncludesXDG(t *testing.T) {
	path := `/tmp/report.html`
	cs := candidates("linux", path)
	if len(cs) < 2 || cs[0].name != "xdg-open" || cs[1].name != "wslview" {
		t.Fatalf("linux candidates = %#v", cs)
	}
	if cs[0].args[0] != path {
		t.Fatalf("path arg = %q", cs[0].args[0])
	}
}

func TestCandidatesDarwinWindows(t *testing.T) {
	path := `C:\tmp\report.html`
	cs := candidates("darwin", "/tmp/report.html")
	if len(cs) != 1 || cs[0].name != "open" {
		t.Fatalf("darwin: %#v", cs)
	}
	cs = candidates("windows", path)
	if len(cs) != 1 || cs[0].name != "rundll32" {
		t.Fatalf("windows: %#v", cs)
	}
	if len(cs[0].args) != 2 || cs[0].args[1] != path {
		t.Fatalf("windows args: %#v", cs[0].args)
	}
}

func TestCandidatesUnsupported(t *testing.T) {
	if cs := candidates("plan9", "/x"); cs != nil {
		t.Fatalf("want nil, got %#v", cs)
	}
}

func TestIsWSL(t *testing.T) {
	// On this repo's CI/dev we often run under WSL; just ensure the check is stable.
	got := isWSL()
	if os.Getenv("WSL_DISTRO_NAME") != "" && !got {
		t.Fatal("WSL_DISTRO_NAME set but isWSL=false")
	}
	if b, err := os.ReadFile("/proc/version"); err == nil {
		v := strings.ToLower(string(b))
		if (strings.Contains(v, "microsoft") || strings.Contains(v, "wsl")) && !got {
			t.Fatalf("proc/version suggests WSL but isWSL=false: %q", string(b))
		}
	}
}
