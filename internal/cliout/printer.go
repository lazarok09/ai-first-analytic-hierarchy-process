package cliout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// isTTYFn reports whether f is a character device. Overridable in tests.
var isTTYFn = defaultIsTTY

func defaultIsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// IsTTY reports whether f is an interactive terminal.
func IsTTY(f *os.File) bool {
	return isTTYFn(f)
}

// IsStdoutTTY reports whether os.Stdout is an interactive terminal.
func IsStdoutTTY() bool {
	return IsTTY(os.Stdout)
}

// WantJSON is true when --json was set or stdout is not a TTY
// (piped / redirected → machine-friendly JSON by default).
// If stdout is nil, os.Stdout is used.
func WantJSON(jsonFlag bool, stdout *os.File) bool {
	if stdout == nil {
		stdout = os.Stdout
	}
	return jsonFlag || !IsTTY(stdout)
}

// Printer writes CLI output as JSON or human tables.
type Printer struct {
	Out  io.Writer
	Err  io.Writer
	JSON bool
}

// New returns a Printer writing to stdout/stderr with the given JSON mode.
func New(jsonMode bool) *Printer {
	return &Printer{Out: os.Stdout, Err: os.Stderr, JSON: jsonMode}
}

// FromCmd builds a Printer from a Cobra command's --json flag
// (including persistent flags inherited from the root command).
func FromCmd(cmd *cobra.Command) *Printer {
	jsonFlag := false
	if cmd != nil {
		jsonFlag, _ = cmd.Flags().GetBool("json")
	}
	return New(WantJSON(jsonFlag, os.Stdout))
}

// Quiet reports whether human fluff should be suppressed.
// True in JSON mode so callers skip decorative lines.
func (p *Printer) Quiet() bool {
	if p == nil {
		return true
	}
	return p.JSON
}

// PrintJSON encodes v as indented JSON to Out.
func (p *Printer) PrintJSON(v any) error {
	return WriteJSON(p.out(), v)
}

// WriteJSON encodes v as indented JSON to w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintTable writes a human-readable table, or JSON objects when in JSON mode.
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	if p != nil && p.JSON {
		objs := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			m := make(map[string]string, len(headers))
			for i, h := range headers {
				if i < len(row) {
					m[h] = row[i]
				} else {
					m[h] = ""
				}
			}
			objs = append(objs, m)
		}
		return p.PrintJSON(objs)
	}
	w := tabwriter.NewWriter(p.out(), 0, 4, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, strings.Join(headers, "\t"))
		for _, row := range rows {
			cells := make([]string, len(headers))
			for i := range headers {
				if i < len(row) {
					cells[i] = row[i]
				}
			}
			fmt.Fprintln(w, strings.Join(cells, "\t"))
		}
	} else {
		for _, row := range rows {
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
	}
	return w.Flush()
}

// Humanf prints a human-only line to Out unless Quiet (JSON mode).
func (p *Printer) Humanf(format string, args ...any) {
	if p.Quiet() {
		return
	}
	fmt.Fprintf(p.out(), format, args...)
}

// Printf always writes to Out (primary payload in table mode).
func (p *Printer) Printf(format string, args ...any) {
	fmt.Fprintf(p.out(), format, args...)
}

func (p *Printer) out() io.Writer {
	if p == nil || p.Out == nil {
		return os.Stdout
	}
	return p.Out
}
