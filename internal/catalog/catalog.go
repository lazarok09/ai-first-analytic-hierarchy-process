// Package catalog is the command-as-doc registry for CLI + MCP (ROADMAP Phase A).
package catalog

import "sort"

// Entry documents one operation shared by CLI and/or MCP.
type Entry struct {
	ID       string   `json:"id"`
	CLI      string   `json:"cli"`
	MCP      string   `json:"mcp,omitempty"`
	Short    string   `json:"short"`
	Long     string   `json:"long,omitempty"`
	Examples []string `json:"examples,omitempty"`
}

// All returns the seeded catalog sorted by id.
func All() []Entry {
	out := append([]Entry(nil), entries...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Lookup finds an entry by id, CLI verb, or MCP tool name.
func Lookup(q string) (Entry, bool) {
	if q == "" {
		return Entry{}, false
	}
	for _, e := range entries {
		if e.ID == q || e.CLI == q || e.MCP == q {
			return e, true
		}
	}
	return Entry{}, false
}

// CLIToMCP returns the MCP tool name for a CLI verb, if mapped.
func CLIToMCP(cli string) string {
	e, ok := Lookup(cli)
	if !ok {
		return ""
	}
	return e.MCP
}

var entries = []Entry{
	{
		ID: "status", CLI: "ahp status", MCP: "workspace_status",
		Short: "Home screen: completeness, CR, proposals, ranking peek, next step",
		Long:  "Summarize the workspace like git status.",
		Examples: []string{
			"ahp status",
			"ahp status --check --json",
		},
	},
	{
		ID: "doctor", CLI: "ahp doctor", MCP: "",
		Short: "Diagnose schema, missing pairs, CR, constraints, purchase/method integrity",
		Examples: []string{
			"ahp doctor",
			"ahp doctor --purchase --json",
			"ahp doctor --method --json",
			"ahp doctor --strict --json",
		},
	},
	{
		ID: "constrain", CLI: "ahp constrain", MCP: "constrain",
		Short: "Set prefer direction and/or eligibility band (min/max/unit) or --must-have",
		Long:  "Writes data/constraints.csv. --prefer higher|lower alone is enough for absolute/Gaussian. Out-of-band alts are excluded from synthesis.",
		Examples: []string{
			"ahp constrain quality --prefer higher",
			"ahp constrain value --min 200 --max 400 --unit BRL --prefer lower",
			"ahp constrain parking --must-have",
			"ahp constrain --json",
		},
	},
	{
		ID: "next", CLI: "ahp next", MCP: "",
		Short: "Print the single best next action",
		Examples: []string{"ahp next --json"},
	},
	{
		ID: "tree", CLI: "ahp tree", MCP: "",
		Short: "Print goal → criteria → alternative-matrix hierarchy",
		Examples: []string{"ahp tree", "ahp tree --json"},
	},
	{
		ID: "pair-missing", CLI: "ahp pair missing", MCP: "missing_pairs",
		Short: "List pairwise gaps with proposal coverage",
		Long:  "JSON: missing_committed, covered_by_proposals, uncovered.",
		Examples: []string{"ahp pair missing", "ahp pair missing --json"},
	},
	{
		ID: "pair-set", CLI: "ahp pair set", MCP: "propose_pairwise",
		Short: "Write one Saaty judgment (--as proposal|committed)",
		Long:  "Agents/non-TTY default to --as proposal (AHP_AGENT=1).",
		Examples: []string{
			"ahp pair set criteria cost quality 3 --as proposal",
			"AHP_AGENT=1 ahp pair set criteria cost quality 3",
		},
	},
	{
		ID: "pair-repairs", CLI: "ahp pair repairs", MCP: "suggest_repairs",
		Short: "CR repair suggestions",
		Examples: []string{"ahp pair repairs --json"},
	},
	{
		ID: "pair-ask", CLI: "ahp pair ask", MCP: "",
		Short: "Interactive Saaty walk (TTY only)",
		Examples: []string{"ahp pair ask"},
	},
	{
		ID: "plan", CLI: "ahp plan", MCP: "",
		Short: "Preview ranking/CR if proposals were committed",
		Examples: []string{"ahp plan", "ahp plan --json"},
	},
	{
		ID: "apply", CLI: "ahp apply", MCP: "commit_proposals",
		Short: "Commit proposal judgments (summary by default; --verbose for rows)",
		Examples: []string{"ahp apply --dry-run", "ahp apply -y", "ahp apply -y --verbose --json"},
	},
	{
		ID: "get", CLI: "ahp get", MCP: "get_state",
		Short: "List ranking|pairs|criteria|alternatives|matrices",
		Examples: []string{
			"ahp get ranking --json",
			"ahp get pairs --status proposal",
		},
	},
	{
		ID: "describe", CLI: "ahp describe", MCP: "",
		Short: "Describe a matrix in detail",
		Examples: []string{"ahp describe matrix criteria --json"},
	},
	{
		ID: "compute", CLI: "ahp compute", MCP: "compute",
		Short: "Solve eigenvectors, CR, synthesis; write outputs",
		Long:  "Default --method=saaty. Opt-in --method=gaussian|hybrid|absolute|compare for comparative views.",
		Examples: []string{
			"ahp compute",
			"ahp compute --include-proposals",
			"ahp compute --method=compare",
			"ahp compute --journal",
		},
	},
	{
		ID: "journal-list", CLI: "ahp journal list", MCP: "journal_list",
		Short: "List tmp/journal compute snapshots",
		Long:  "Journaling is off by default. Enable with --journal, AHP_JOURNAL=1, or [journal] enabled=true.",
		Examples: []string{"ahp journal list", "ahp journal list --limit 5 --json"},
	},
	{
		ID: "journal-show", CLI: "ahp journal show", MCP: "journal_show",
		Short: "Show meta + summary for a journal run",
		Examples: []string{"ahp journal show latest", "ahp journal show <id> --json"},
	},
	{
		ID: "journal-path", CLI: "ahp journal path", MCP: "",
		Short: "Print absolute path of a journal run directory",
		Examples: []string{"ahp journal path latest"},
	},
	{
		ID: "journal-prune", CLI: "ahp journal prune", MCP: "",
		Short: "Delete oldest journal runs beyond --keep",
		Examples: []string{"ahp journal prune --keep 20"},
	},
	{
		ID: "explain", CLI: "ahp explain", MCP: "explain",
		Short: "Criterion contributions for alternative global weights",
		Long:  "contribution = leaf criterion weight × local alternative weight under that criterion.",
		Examples: []string{
			"ahp explain",
			"ahp explain shure_srh240a --json",
		},
	},
	{
		ID: "sensitivity", CLI: "ahp sensitivity", MCP: "sensitivity",
		Short: "±δ leaf-weight tornado and rank-reversal thresholds",
		Long:  "Writes output/sensitivity.json. Requires a complete workspace.",
		Examples: []string{
			"ahp sensitivity",
			"ahp sensitivity --delta 0.10 --json",
		},
	},
	{
		ID: "rate", CLI: "ahp rate", MCP: "suggest_from_attributes",
		Short: "Opt-in: attributes → Saaty proposals (--mode=saaty) or sum-norm preview",
		Long:  "Default --mode=saaty is lossy ratio→Saaty proposals. --mode=sum-norm previews Santos sum-norm locals (no pairwise write).",
		Examples: []string{
			"ahp rate --criterion value --prefer lower --dry-run",
			"ahp rate --mode=sum-norm --criterion value --prefer lower",
			"ahp rate --criterion quality --prefer higher --stretch",
			"ahp pair suggest-from-attributes --criterion value --prefer lower",
		},
	},
	{
		ID: "gaussian", CLI: "ahp gaussian", MCP: "gaussian",
		Short: "Comparative AHP-Gaussian ranking from attributes (σ/μ weights)",
		Long:  "Requires prefer on attribute-ready leaves. Writes output/gaussian.json. No Saaty CR.",
		Examples: []string{
			"ahp constrain quality --prefer higher",
			"ahp gaussian",
			"ahp gaussian --json",
		},
	},
	{
		ID: "absolute", CLI: "ahp absolute", MCP: "absolute",
		Short: "Sum-norm attributes; hybrid score with criteria weights when available",
		Long:  "Writes output/absolute.json. Comparative only.",
		Examples: []string{"ahp absolute", "ahp absolute --json"},
	},
	{
		ID: "pair-suggest-from-attributes", CLI: "ahp pair suggest-from-attributes", MCP: "suggest_from_attributes",
		Short: "Propose alt pairwise from numeric attributes",
		Examples: []string{
			"ahp pair suggest-from-attributes --criterion value --prefer lower --dry-run",
			"ahp pair suggest-from-attributes --criterion quality --prefer higher --stretch",
			"ahp pair suggest-from-attributes --criterion value --prefer lower --refresh",
		},
	},
	{
		ID: "remove-alternative", CLI: "ahp remove-alternative", MCP: "remove_alternative",
		Short: "Delete an alternative and clean attributes + pairwise refs",
		Examples: []string{"ahp remove-alternative mirassol"},
	},
	{
		ID: "quote", CLI: "ahp quote", MCP: "",
		Short: "Multi-source quotes; pick best into attributes",
		Examples: []string{
			"ahp quote add swell value 353 --unit BRL --source \"Decolar Pix\" --url https://…",
			"ahp quote pick value --prefer lower",
			"ahp quote list",
		},
	},
	{
		ID: "share", CLI: "ahp share", MCP: "share",
		Short: "Pasteable ranking summary (WhatsApp-friendly)",
		Examples: []string{"ahp share --top 5", "ahp share --whatsapp"},
	},
	{
		ID: "pair-import", CLI: "ahp pair import", MCP: "import_pairwise",
		Short: "Bulk upsert pairwise judgments from CSV/JSON",
		Examples: []string{
			"ahp pair import judgments.csv",
			"ahp pair import judgments.json --as proposal",
		},
	},
	{
		ID: "catalog", CLI: "ahp catalog", MCP: "",
		Short: "List CLI↔MCP catalog entries",
		Examples: []string{"ahp catalog --json"},
	},
	{
		ID: "docs", CLI: "ahp docs", MCP: "",
		Short: "Show catalog documentation for a command",
		Examples: []string{"ahp docs pair-set", "ahp docs plan"},
	},
	{
		ID: "completion", CLI: "ahp completion", MCP: "",
		Short: "Generate shell completion scripts",
		Examples: []string{
			"ahp completion bash",
			"ahp completion zsh",
		},
	},
}
