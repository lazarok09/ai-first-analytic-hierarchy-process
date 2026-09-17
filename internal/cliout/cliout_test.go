package cliout

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExitConstants(t *testing.T) {
	if ExitOK != 0 || OK != 0 {
		t.Fatal("ExitOK")
	}
	if ExitUsage != 1 || Usage != 1 {
		t.Fatal("ExitUsage")
	}
	if ExitIncomplete != 2 || Incomplete != 2 {
		t.Fatal("ExitIncomplete")
	}
	if ExitInconsistent != 3 || Inconsistent != 3 {
		t.Fatal("ExitInconsistent")
	}
	if ExitProposals != 4 || Proposals != 4 {
		t.Fatal("ExitProposals")
	}
	if ExitIO != 5 || IO != 5 {
		t.Fatal("ExitIO")
	}
}

func TestCode(t *testing.T) {
	if Code(nil) != ExitOK {
		t.Fatalf("nil → %d", Code(nil))
	}
	if Code(errors.New("boom")) != ExitUsage {
		t.Fatalf("plain → %d", Code(errors.New("boom")))
	}
	ee := NewExitError(ExitIncomplete, "missing %d pairs", 3)
	if Code(ee) != ExitIncomplete || ee.Error() != "missing 3 pairs" {
		t.Fatalf("ExitError code=%d msg=%q", Code(ee), ee.Error())
	}
	if Code(Wrap(ExitIO, errors.New("disk"))) != ExitIO {
		t.Fatal("Wrap")
	}
	if Code(Errorf(ExitProposals, "pending")) != ExitProposals {
		t.Fatal("Errorf")
	}
	wrapped := errors.Join(errors.New("ctx"), NewExitError(ExitInconsistent, "CR high"))
	if Code(wrapped) != ExitInconsistent {
		t.Fatalf("Join → %d", Code(wrapped))
	}
	if CodeOf(ee) != ExitIncomplete {
		t.Fatal("CodeOf")
	}
}

func TestWantJSON(t *testing.T) {
	orig := isTTYFn
	t.Cleanup(func() { isTTYFn = orig })

	isTTYFn = func(*os.File) bool { return true }
	if WantJSON(false, os.Stdout) {
		t.Fatal("TTY + no flag → not JSON")
	}
	if !WantJSON(true, os.Stdout) {
		t.Fatal("--json on TTY → JSON")
	}

	isTTYFn = func(*os.File) bool { return false }
	if !WantJSON(false, os.Stdout) {
		t.Fatal("non-TTY → JSON")
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	isTTYFn = defaultIsTTY
	if !WantJSON(false, w) {
		t.Fatal("pipe stdout → JSON")
	}
}

func TestPrintTableVsJSON(t *testing.T) {
	headers := []string{"rank", "name"}
	rows := [][]string{{"1", "Acme"}, {"2", "Globex"}}

	var tableBuf bytes.Buffer
	tp := &Printer{Out: &tableBuf, JSON: false}
	if err := tp.PrintTable(headers, rows); err != nil {
		t.Fatal(err)
	}
	out := tableBuf.String()
	if !strings.Contains(out, "rank") || !strings.Contains(out, "Acme") {
		t.Fatalf("table=%q", out)
	}
	if strings.Contains(strings.TrimSpace(out), "{") {
		t.Fatalf("unexpected JSON in table mode: %q", out)
	}

	var jsonBuf bytes.Buffer
	jp := &Printer{Out: &jsonBuf, JSON: true}
	if err := jp.PrintTable(headers, rows); err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]string
	if err := json.Unmarshal(jsonBuf.Bytes(), &decoded); err != nil {
		t.Fatalf("%v\n%s", err, jsonBuf.String())
	}
	if len(decoded) != 2 || decoded[0]["name"] != "Acme" {
		t.Fatalf("decoded=%v", decoded)
	}
}

func TestPrintJSONAndQuiet(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Out: &buf, JSON: true}
	if !p.Quiet() {
		t.Fatal("JSON → Quiet")
	}
	p.Humanf("nope\n")
	if buf.Len() != 0 {
		t.Fatalf("Humanf leaked: %q", buf.String())
	}
	if err := p.PrintJSON(map[string]int{"ok": 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"ok"`) {
		t.Fatalf("%q", buf.String())
	}
	if (&Printer{JSON: false}).Quiet() {
		t.Fatal("table mode Quiet")
	}
}

func TestFromCmd(t *testing.T) {
	orig := isTTYFn
	t.Cleanup(func() { isTTYFn = orig })
	isTTYFn = func(*os.File) bool { return true }

	root := &cobra.Command{Use: "ahp"}
	root.PersistentFlags().Bool("json", false, "JSON output")
	child := &cobra.Command{Use: "status"}
	root.AddCommand(child)

	child.RunE = func(cmd *cobra.Command, args []string) error {
		if FromCmd(cmd).JSON {
			t.Error("expected table without --json")
		}
		return nil
	}
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	child.RunE = func(cmd *cobra.Command, args []string) error {
		if !FromCmd(cmd).JSON {
			t.Error("expected JSON with --json")
		}
		return nil
	}
	root.SetArgs([]string{"status", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
