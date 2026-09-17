package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/render"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const instructions = `AHP (Analytic Hierarchy Process) local workspace tools.

Source of truth is CSV under the workspace:
- data/criteria.csv, data/alternatives.csv
- data/pairwise.csv (Saaty 1–9 or reciprocals; status proposal|committed)
- data/attributes.csv (objective facts; do not treat as automatic weights)
- output/weights.csv, output/ranking.csv, output/report.html after compute

Rules:
- Propose pairwise with status=proposal. Do not silently commit subjective judgments.
- Humans commit by editing CSV status or calling commit_proposals / commit_pairwise.
- After structure or judgment changes, call compute then read the HTML/CSV outputs.
- Prefer missing_pairs and workspace_status before filling matrices.
- If CR > 0.10, call suggest_repairs and propose revised values (still as proposals).
- Use explain / sensitivity for close rankings. Use suggest_from_attributes to turn numeric attributes into proposal judgments (never commits).`

func Run() error {
	s := server.NewMCPServer("ahp-method", "0.3.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(instructions),
	)

	s.AddTool(mcp.NewTool("init_workspace",
		mcp.WithDescription("Create a local AHP workspace (ahp.toml + data CSVs)."),
		mcp.WithString("path", mcp.Description("Directory path"), mcp.DefaultString(".")),
		mcp.WithString("title", mcp.Description("Decision title"), mcp.DefaultString("Untitled decision")),
		mcp.WithString("description", mcp.Description("Description"), mcp.DefaultString("")),
	), wrap(func(args map[string]any) (any, error) {
		path := strArg(args, "path", ".")
		ws := workspace.Open(path)
		if err := ws.EnsureLayout(); err != nil {
			return nil, err
		}
		meta := workspace.Meta{Title: strArg(args, "title", "Untitled decision"), Description: strArg(args, "description", "")}
		if err := ws.SaveMeta(meta); err != nil {
			return nil, err
		}
		return map[string]any{"workspace": ws.Root, "title": meta.Title}, nil
	}))

	s.AddTool(mcp.NewTool("set_goal",
		mcp.WithDescription("Set the decision title/description in ahp.toml."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Title")),
		mcp.WithString("description", mcp.Description("Description"), mcp.DefaultString("")),
		mcp.WithString("workspace", mcp.Description("Workspace path")),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		meta := workspace.Meta{Title: strArg(args, "title", ""), Description: strArg(args, "description", "")}
		if err := ws.SaveMeta(meta); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "title": meta.Title}, nil
	}))

	s.AddTool(mcp.NewTool("upsert_criterion",
		mcp.WithDescription("Create or update a criterion row in data/criteria.csv."),
		mcp.WithString("item_id", mcp.Required(), mcp.Description("Criterion id")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Display name")),
		mcp.WithString("parent_id", mcp.Description("Parent id"), mcp.DefaultString("")),
		mcp.WithString("description", mcp.Description("Description"), mcp.DefaultString("")),
		mcp.WithString("workspace", mcp.Description("Workspace path")),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		item := workspace.Criterion{
			ID: strArg(args, "item_id", ""), Name: strArg(args, "name", ""),
			ParentID: strArg(args, "parent_id", ""), Description: strArg(args, "description", ""),
		}
		if err := ws.UpsertCriterion(item); err != nil {
			return nil, err
		}
		return item, nil
	}))

	s.AddTool(mcp.NewTool("upsert_alternative",
		mcp.WithDescription("Create or update an alternative in data/alternatives.csv."),
		mcp.WithString("item_id", mcp.Required(), mcp.Description("Alternative id")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Display name")),
		mcp.WithString("description", mcp.Description("Description"), mcp.DefaultString("")),
		mcp.WithString("workspace", mcp.Description("Workspace path")),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		item := workspace.Alternative{
			ID: strArg(args, "item_id", ""), Name: strArg(args, "name", ""),
			Description: strArg(args, "description", ""),
		}
		if err := ws.UpsertAlternative(item); err != nil {
			return nil, err
		}
		return item, nil
	}))

	s.AddTool(mcp.NewTool("upsert_attribute",
		mcp.WithDescription("Write an objective fact into data/attributes.csv (not a judgment)."),
		mcp.WithString("alternative_id", mcp.Required()),
		mcp.WithString("criterion_id", mcp.Required()),
		mcp.WithString("value", mcp.Required()),
		mcp.WithString("unit", mcp.DefaultString("")),
		mcp.WithString("source", mcp.DefaultString("")),
		mcp.WithString("note", mcp.DefaultString("")),
		mcp.WithString("workspace"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		item := workspace.AttributeRow{
			AlternativeID: strArg(args, "alternative_id", ""),
			CriterionID:   strArg(args, "criterion_id", ""),
			Value:         strArg(args, "value", ""),
			Unit:          strArg(args, "unit", ""),
			Source:        strArg(args, "source", ""),
			Note:          strArg(args, "note", ""),
		}
		if err := ws.UpsertAttribute(item); err != nil {
			return nil, err
		}
		return item, nil
	}))

	s.AddTool(mcp.NewTool("propose_pairwise",
		mcp.WithDescription("Write a Saaty pairwise judgment as status=proposal. Never commits."),
		mcp.WithString("matrix", mcp.Required()),
		mcp.WithString("left", mcp.Required()),
		mcp.WithString("right", mcp.Required()),
		mcp.WithString("value", mcp.Required(), mcp.Description("Saaty 1-9 or 1/n")),
		mcp.WithString("note", mcp.DefaultString("")),
		mcp.WithString("workspace"),
	), wrap(func(args map[string]any) (any, error) {
		return upsertPair(args, "proposal")
	}))

	s.AddTool(mcp.NewTool("commit_pairwise",
		mcp.WithDescription("Commit a pairwise judgment (prefer a human editing the CSV)."),
		mcp.WithString("matrix", mcp.Required()),
		mcp.WithString("left", mcp.Required()),
		mcp.WithString("right", mcp.Required()),
		mcp.WithString("value", mcp.Required()),
		mcp.WithString("note", mcp.DefaultString("")),
		mcp.WithString("workspace"),
	), wrap(func(args map[string]any) (any, error) {
		return upsertPair(args, "committed")
	}))

	s.AddTool(mcp.NewTool("commit_proposals",
		mcp.WithDescription("Flip proposal rows to committed. Optional matrix/left/right filters."),
		mcp.WithString("matrix"),
		mcp.WithString("left"),
		mcp.WithString("right"),
		mcp.WithString("workspace"),
	), wrap(func(args map[string]any) (any, error) {
		left, right := strArg(args, "left", ""), strArg(args, "right", "")
		if (left == "") != (right == "") {
			return nil, fmt.Errorf("provide both left and right, or neither")
		}
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		return ws.CommitProposals(strArg(args, "matrix", ""), left, right)
	}))

	s.AddTool(mcp.NewTool("get_state",
		mcp.WithDescription("Aggregated hierarchy, matrices, CR, ranking (does not write files)."),
		mcp.WithString("workspace"),
		mcp.WithBoolean("include_proposals", mcp.Description("Include proposals")),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		result, err := ws.Compute(boolArg(args, "include_proposals"))
		if err != nil {
			return nil, err
		}
		matrices := map[string]any{}
		for k, v := range result.Matrices {
			matrices[k] = map[string]any{
				"complete": v.Complete, "cr": v.CR, "weights": v.Weights,
				"missing": v.Missing, "repairs": v.Repairs,
			}
		}
		return map[string]any{
			"title": result.Title, "complete": result.Complete, "consistent": result.Consistent,
			"warnings": result.Warnings, "criteria": result.Criteria, "alternatives": result.Alternatives,
			"ranking": result.Ranking, "matrices": matrices,
		}, nil
	}))

	s.AddTool(mcp.NewTool("workspace_status",
		mcp.WithDescription("Compact status: counts, missing pairs, CR repairs, ranking."),
		mcp.WithString("workspace"),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		return ws.Status(boolArg(args, "include_proposals"))
	}))

	s.AddTool(mcp.NewTool("missing_pairs",
		mcp.WithDescription("Pairwise gaps with coverage: missing_committed, covered_by_proposals, uncovered."),
		mcp.WithString("workspace"),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		s, err := ws.Status(boolArg(args, "include_proposals"))
		if err != nil {
			return nil, err
		}
		cov := workspace.MissingCoverage{
			MissingCommitted:   s.MissingCommitted,
			CoveredByProposals: s.CoveredByProposals,
			Uncovered:          s.Uncovered,
		}
		if cov.MissingCommitted == nil {
			cov.MissingCommitted = []workspace.MissingPair{}
		}
		if cov.CoveredByProposals == nil {
			cov.CoveredByProposals = []workspace.MissingPair{}
		}
		if cov.Uncovered == nil {
			cov.Uncovered = []workspace.MissingPair{}
		}
		return cov, nil
	}))

	s.AddTool(mcp.NewTool("suggest_repairs",
		mcp.WithDescription("Pairs that most disagree with derived weights (for CR > 0.10 matrices)."),
		mcp.WithString("workspace"),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		s, err := ws.Status(boolArg(args, "include_proposals"))
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(s.Repairs))
		for _, h := range s.Repairs {
			out = append(out, map[string]any{
				"matrix": h.Matrix, "left": h.Left, "right": h.Right,
				"current": h.Current, "implied": h.Implied, "suggested": h.Suggested,
				"current_label": engine.FormatSaaty(h.Current),
				"suggested_label": engine.FormatSaaty(h.Suggested),
			})
		}
		return out, nil
	}))

	s.AddTool(mcp.NewTool("compute",
		mcp.WithDescription("Run AHP math and write output/weights.csv, ranking.csv, report.html."),
		mcp.WithString("workspace"),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		result, err := ws.Compute(boolArg(args, "include_proposals"))
		if err != nil {
			return nil, err
		}
		paths, err := ws.WriteOutputs(result, render.HTML(result))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"complete": result.Complete, "consistent": result.Consistent,
			"warnings": result.Warnings, "ranking": result.Ranking, "paths": paths,
		}, nil
	}))

	s.AddTool(mcp.NewTool("explain",
		mcp.WithDescription("Break down alternative global weights into criterion contributions."),
		mcp.WithString("workspace"),
		mcp.WithString("alternative_id", mcp.Description("Optional alternative id filter")),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		return ws.Explain(boolArg(args, "include_proposals"), strArg(args, "alternative_id", ""))
	}))

	s.AddTool(mcp.NewTool("sensitivity",
		mcp.WithDescription("One-at-a-time leaf-weight sensitivity (±delta). Writes output/sensitivity.json."),
		mcp.WithString("workspace"),
		mcp.WithNumber("delta", mcp.Description("Absolute weight perturbation (default 0.05)")),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		delta := floatArg(args, "delta", 0.05)
		sens, err := ws.Sensitivity(boolArg(args, "include_proposals"), delta)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(ws.OutputDir(), 0o755); err != nil {
			return nil, err
		}
		path := ws.OutputDir() + "/sensitivity.json"
		b, err := json.MarshalIndent(sens, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"result": sens, "sensitivity_json": path}, nil
	}))

	s.AddTool(mcp.NewTool("suggest_from_attributes",
		mcp.WithDescription("Convert numeric attributes under a criterion into Saaty pairwise proposals (never commits)."),
		mcp.WithString("criterion_id", mcp.Required()),
		mcp.WithString("prefer", mcp.Description("higher|lower"), mcp.DefaultString("higher")),
		mcp.WithBoolean("dry_run"),
		mcp.WithString("workspace"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		prefer := engine.PreferDirection(strArg(args, "prefer", "higher"))
		return ws.SuggestFromAttributes(workspace.SuggestFromAttributesOptions{
			CriterionID: strArg(args, "criterion_id", ""),
			Prefer:      prefer,
			DryRun:      boolArg(args, "dry_run"),
		})
	}))

	s.AddTool(mcp.NewTool("render_report",
		mcp.WithDescription("Compute and return the HTML report path for humans to open."),
		mcp.WithString("workspace"),
		mcp.WithBoolean("include_proposals"),
	), wrap(func(args map[string]any) (any, error) {
		ws, err := open(args)
		if err != nil {
			return nil, err
		}
		result, err := ws.Compute(boolArg(args, "include_proposals"))
		if err != nil {
			return nil, err
		}
		paths, err := ws.WriteOutputs(result, render.HTML(result))
		if err != nil {
			return nil, err
		}
		return paths, nil
	}))

	return server.ServeStdio(s)
}

func upsertPair(args map[string]any, status string) (any, error) {
	ws, err := open(args)
	if err != nil {
		return nil, err
	}
	v, err := engine.ParseSaaty(strArg(args, "value", ""))
	if err != nil {
		return nil, err
	}
	row := workspace.PairwiseRow{
		Matrix: strArg(args, "matrix", ""),
		Left:   strArg(args, "left", ""),
		Right:  strArg(args, "right", ""),
		Value:  v,
		Status: status,
		Note:   strArg(args, "note", ""),
	}
	if err := ws.UpsertPairwise(row); err != nil {
		return nil, err
	}
	return row, nil
}

func open(args map[string]any) (*workspace.Workspace, error) {
	path := strArg(args, "workspace", "")
	if path == "" {
		path = os.Getenv("AHP_WORKSPACE")
	}
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = workspace.Find(cwd)
	}
	ws := workspace.Open(path)
	if _, err := os.Stat(ws.TomlPath()); err != nil {
		return nil, fmt.Errorf("no ahp.toml at %s — call init_workspace first", ws.Root)
	}
	return ws, nil
}

func wrap(fn func(map[string]any) (any, error)) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		out, err := fn(args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}
}

func strArg(args map[string]any, key, def string) string {
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func boolArg(args map[string]any, key string) bool {
	if args == nil {
		return false
	}
	v, ok := args[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func floatArg(args map[string]any, key string, def float64) float64 {
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return def
		}
		return f
	case string:
		var f float64
		if _, err := fmt.Sscanf(t, "%f", &f); err == nil {
			return f
		}
		return def
	default:
		return def
	}
}
