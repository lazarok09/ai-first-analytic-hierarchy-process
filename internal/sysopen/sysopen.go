// Package sysopen launches files with the OS default application.
package sysopen

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open launches path with the platform default file opener.
// On Linux it tries xdg-open, then wslview, then WSL → Windows
// (powershell Invoke-Item / cmd start) when running under WSL.
func Open(path string) error {
	var tried []string
	for _, c := range candidates(runtime.GOOS, path) {
		bin, err := exec.LookPath(c.name)
		if err != nil {
			tried = append(tried, c.name)
			continue
		}
		cmd := exec.Command(bin, c.args...)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
		return nil
	}
	if len(tried) == 0 {
		return fmt.Errorf("no default file opener for GOOS=%s", runtime.GOOS)
	}
	return fmt.Errorf("no usable file opener (tried %s)", strings.Join(tried, ", "))
}

type candidate struct {
	name string
	args []string
}

func candidates(goos, path string) []candidate {
	switch goos {
	case "windows":
		return []candidate{{"rundll32", []string{"url.dll,FileProtocolHandler", path}}}
	case "darwin":
		return []candidate{{"open", []string{path}}}
	case "linux", "freebsd", "netbsd", "openbsd":
		out := []candidate{
			{"xdg-open", []string{path}},
			{"wslview", []string{path}},
		}
		if isWSL() {
			if win, err := windowsPath(path); err == nil {
				out = append(out,
					candidate{"powershell.exe", []string{"-NoProfile", "-Command", "Invoke-Item", "-LiteralPath", win}},
					candidate{"cmd.exe", []string{"/c", "start", "", win}},
				)
			}
		}
		return out
	default:
		return nil
	}
}

func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	v := strings.ToLower(string(b))
	return strings.Contains(v, "microsoft") || strings.Contains(v, "wsl")
}

func windowsPath(linuxPath string) (string, error) {
	out, err := exec.Command("wslpath", "-w", linuxPath).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
