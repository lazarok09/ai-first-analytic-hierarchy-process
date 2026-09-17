package catalog

import "testing"

func TestLookupAndCLIToMCP(t *testing.T) {
	e, ok := Lookup("pair-set")
	if !ok || e.CLI != "ahp pair set" || e.MCP != "propose_pairwise" {
		t.Fatalf("got %+v ok=%v", e, ok)
	}
	if CLIToMCP("ahp plan") != "" {
		t.Fatal("plan has no MCP yet")
	}
	if CLIToMCP("ahp apply") != "commit_proposals" {
		t.Fatalf("apply map=%q", CLIToMCP("ahp apply"))
	}
	all := All()
	if len(all) < 10 {
		t.Fatalf("catalog too small: %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatalf("unsorted: %s then %s", all[i-1].ID, all[i].ID)
		}
	}
}
