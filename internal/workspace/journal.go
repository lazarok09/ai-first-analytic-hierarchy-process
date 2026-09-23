package workspace

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultJournalDir  = "tmp/journal"
	defaultJournalKeep = 50
	journalIndexName   = "index.jsonl"
)

// JournalConfig controls optional per-run snapshots under tmp/journal/.
// Default Enabled is false until Phase A is dogfooded (see docs/PLAN_JOURNAL.md).
type JournalConfig struct {
	Enabled            bool   `json:"enabled"`
	Dir                string `json:"dir"`
	Keep               int    `json:"keep"`
	IncludeHTML        bool   `json:"include_html"`
	IncludeSensitivity bool   `json:"include_sensitivity"`
}

// WriteOutputsOpts controls optional side effects of WriteOutputs (journal).
type WriteOutputsOpts struct {
	// Journal forces journaling on (true) or off (false). nil uses config/env.
	Journal *bool
	// Command is the triggering verb (e.g. "compute").
	Command string
	// Argv is the process argv for meta (optional).
	Argv []string
	// AHPVersion is recorded in meta.json (optional).
	AHPVersion string
}

// JournalMeta is persisted as meta.json inside a journal entry.
type JournalMeta struct {
	ID               string  `json:"id"`
	CreatedAt        string  `json:"created_at"`
	Command          string  `json:"command"`
	Argv             []string `json:"argv,omitempty"`
	IncludeProposals bool    `json:"include_proposals"`
	AHPVersion       string  `json:"ahp_version,omitempty"`
	WorkspaceTitle   string  `json:"workspace_title"`
	DataHash         string  `json:"data_hash"`
	Complete         bool    `json:"complete"`
	Consistent       bool    `json:"consistent"`
	LeaderID         string  `json:"leader_id,omitempty"`
	LeaderWeight     float64 `json:"leader_weight,omitempty"`
}

// JournalSummary is a small porcelain peek for list/show UIs.
type JournalSummary struct {
	Complete   bool              `json:"complete"`
	Consistent bool              `json:"consistent"`
	Top        []JournalTopRow   `json:"top"`
	WorstCR    *JournalWorstCR   `json:"worst_cr,omitempty"`
}

// JournalTopRow is one ranked alternative in summary.json.
type JournalTopRow struct {
	Rank   int     `json:"rank"`
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// JournalWorstCR names the matrix with the highest CR among complete matrices.
type JournalWorstCR struct {
	Matrix string  `json:"matrix"`
	CR     float64 `json:"cr"`
}

// JournalIndexEntry is one append-only line in index.jsonl.
type JournalIndexEntry struct {
	ID           string  `json:"id"`
	Path         string  `json:"path"`
	CreatedAt    string  `json:"created_at"`
	Command      string  `json:"command"`
	Complete     bool    `json:"complete"`
	Consistent   bool    `json:"consistent"`
	LeaderID     string  `json:"leader_id,omitempty"`
	LeaderWeight float64 `json:"leader_weight,omitempty"`
	DataHash     string  `json:"data_hash"`
}

// JournalEntry is the resolved view returned by JournalShow.
type JournalEntry struct {
	Meta    JournalMeta     `json:"meta"`
	Summary JournalSummary  `json:"summary"`
	Dir     string          `json:"dir"`
}

// DefaultJournalConfig returns Phase A defaults (journal off).
func DefaultJournalConfig() JournalConfig {
	return JournalConfig{
		Enabled:            false,
		Dir:                defaultJournalDir,
		Keep:               defaultJournalKeep,
		IncludeHTML:        false,
		IncludeSensitivity: true,
	}
}

// JournalDir returns the absolute journal root for this workspace.
func (w *Workspace) JournalDir() string {
	cfg := w.ResolveJournalConfig(WriteOutputsOpts{})
	return filepath.Join(w.Root, cfg.Dir)
}

// ResolveJournalConfig merges defaults, ahp.toml [journal], env, and write opts.
// AHP_JOURNAL=0|false is a kill-switch. AHP_JOURNAL=1|true enables when config is off.
func (w *Workspace) ResolveJournalConfig(opts WriteOutputsOpts) JournalConfig {
	cfg := DefaultJournalConfig()
	if loaded, err := w.LoadJournalConfig(); err == nil {
		cfg = loaded
	}
	env := strings.TrimSpace(os.Getenv("AHP_JOURNAL"))
	kill := false
	switch strings.ToLower(env) {
	case "0", "false", "no", "off":
		kill = true
		cfg.Enabled = false
	case "1", "true", "yes", "on":
		cfg.Enabled = true
	}
	if !kill && opts.Journal != nil {
		cfg.Enabled = *opts.Journal
	}
	if cfg.Dir == "" {
		cfg.Dir = defaultJournalDir
	}
	if cfg.Keep <= 0 {
		cfg.Keep = defaultJournalKeep
	}
	return cfg
}

// LoadJournalConfig parses optional [journal] from ahp.toml.
func (w *Workspace) LoadJournalConfig() (JournalConfig, error) {
	cfg := DefaultJournalConfig()
	b, err := os.ReadFile(w.TomlPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	inJournal := false
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inJournal = line == "[journal]"
			continue
		}
		if !inJournal {
			continue
		}
		key, val, ok := splitTOMLKV(line)
		if !ok {
			continue
		}
		switch key {
		case "enabled":
			cfg.Enabled = parseTOMLBool(val, cfg.Enabled)
		case "dir":
			cfg.Dir = unquoteTOML(val)
		case "keep":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && n > 0 {
				cfg.Keep = n
			}
		case "include_html":
			cfg.IncludeHTML = parseTOMLBool(val, cfg.IncludeHTML)
		case "include_sensitivity":
			cfg.IncludeSensitivity = parseTOMLBool(val, cfg.IncludeSensitivity)
		}
	}
	return cfg, nil
}

func splitTOMLKV(line string) (key, val string, ok bool) {
	i := strings.Index(line, "=")
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

func parseTOMLBool(val string, def bool) bool {
	v := strings.ToLower(unquoteTOML(val))
	switch v {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

// WriteJournalEntry snapshots data/ + compute results under tmp/journal/<id>/.
func (w *Workspace) WriteJournalEntry(result *ComputeResult, html string, opts WriteOutputsOpts) (*JournalIndexEntry, error) {
	cfg := w.ResolveJournalConfig(opts)
	if !cfg.Enabled {
		return nil, nil
	}
	if result == nil {
		return nil, fmt.Errorf("journal: nil compute result")
	}

	dataHash, err := w.hashDataFiles()
	if err != nil {
		return nil, err
	}
	shortHash := dataHash
	if len(shortHash) > 8 {
		shortHash = shortHash[:8]
	}

	cmd := opts.Command
	if cmd == "" {
		cmd = "compute"
	}
	now := time.Now().UTC()
	id := fmt.Sprintf("%s_%s_%s", now.Format("20060102T150405Z"), cmd, shortHash)
	relDir := filepath.Join(cfg.Dir, id)
	absDir := filepath.Join(w.Root, relDir)
	if _, err := os.Stat(absDir); err == nil {
		id = fmt.Sprintf("%s_%d", id, now.UnixNano()%1000)
		relDir = filepath.Join(cfg.Dir, id)
		absDir = filepath.Join(w.Root, relDir)
	}

	dataDst := filepath.Join(absDir, "data")
	resultsDst := filepath.Join(absDir, "results")
	if err := os.MkdirAll(dataDst, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resultsDst, 0o755); err != nil {
		return nil, err
	}

	for _, name := range []string{
		"criteria.csv", "alternatives.csv", "pairwise.csv",
		"attributes.csv", "constraints.csv", "quotes.csv",
	} {
		src := w.dataPath(name)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if err := copyFile(src, filepath.Join(dataDst, name)); err != nil {
			return nil, fmt.Errorf("journal copy %s: %w", name, err)
		}
	}

	if err := w.writeResultsTo(resultsDst, result, html, cfg); err != nil {
		return nil, err
	}

	meta := JournalMeta{
		ID:               id,
		CreatedAt:        now.Format(time.RFC3339Nano),
		Command:          cmd,
		Argv:             append([]string(nil), opts.Argv...),
		IncludeProposals: result.IncludeProposals,
		AHPVersion:       opts.AHPVersion,
		WorkspaceTitle:   result.Title,
		DataHash:         shortHash,
		Complete:         result.Complete,
		Consistent:       result.Consistent,
	}
	if len(result.Ranking) > 0 {
		meta.LeaderID = result.Ranking[0].ID
		meta.LeaderWeight = result.Ranking[0].Weight
	}
	if err := writeJSONFile(filepath.Join(absDir, "meta.json"), meta); err != nil {
		return nil, err
	}

	summary := buildJournalSummary(result)
	if err := writeJSONFile(filepath.Join(absDir, "summary.json"), summary); err != nil {
		return nil, err
	}

	entry := JournalIndexEntry{
		ID:           id,
		Path:         filepath.ToSlash(relDir),
		CreatedAt:    meta.CreatedAt,
		Command:      cmd,
		Complete:     result.Complete,
		Consistent:   result.Consistent,
		LeaderID:     meta.LeaderID,
		LeaderWeight: meta.LeaderWeight,
		DataHash:     shortHash,
	}
	if err := w.appendJournalIndex(cfg, entry); err != nil {
		return nil, err
	}
	if _, err := w.JournalPrune(cfg.Keep); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (w *Workspace) writeResultsTo(dir string, result *ComputeResult, html string, cfg JournalConfig) error {
	var weightRows []map[string]string
	for scope, m := range result.Matrices {
		for itemID, wt := range m.Weights {
			weightRows = append(weightRows, map[string]string{
				"scope": scope, "item_id": itemID, "name": m.Names[itemID],
				"weight":     fmt.Sprintf("%.6f", wt),
				"lambda_max": fmtOpt(m.LambdaMax), "ci": fmtOpt(m.CI), "cr": fmtOpt(m.CR),
				"complete": strconv.FormatBool(m.Complete),
			})
		}
	}
	if err := writeCSV(filepath.Join(dir, "weights.csv"),
		[]string{"scope", "item_id", "name", "weight", "lambda_max", "ci", "cr", "complete"}, weightRows); err != nil {
		return err
	}
	rankRows := make([]map[string]string, 0, len(result.Ranking))
	for _, r := range result.Ranking {
		rankRows = append(rankRows, map[string]string{
			"rank": strconv.Itoa(r.Rank), "alternative_id": r.ID, "name": r.Name,
			"global_weight": fmt.Sprintf("%.6f", r.Weight),
		})
	}
	if err := writeCSV(filepath.Join(dir, "ranking.csv"),
		[]string{"rank", "alternative_id", "name", "global_weight"}, rankRows); err != nil {
		return err
	}
	jb, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "compute.json"), jb, 0o644); err != nil {
		return err
	}
	if cfg.IncludeHTML && html != "" {
		if err := os.WriteFile(filepath.Join(dir, "report.html"), []byte(html), 0o644); err != nil {
			return err
		}
	}
	if cfg.IncludeSensitivity {
		sensSrc := filepath.Join(w.OutputDir(), "sensitivity.json")
		if _, err := os.Stat(sensSrc); err == nil {
			if err := copyFile(sensSrc, filepath.Join(dir, "sensitivity.json")); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildJournalSummary(result *ComputeResult) JournalSummary {
	s := JournalSummary{
		Complete:   result.Complete,
		Consistent: result.Consistent,
		Top:        make([]JournalTopRow, 0, 5),
	}
	n := len(result.Ranking)
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		r := result.Ranking[i]
		s.Top = append(s.Top, JournalTopRow{Rank: r.Rank, ID: r.ID, Name: r.Name, Weight: r.Weight})
	}
	var worst *JournalWorstCR
	for scope, m := range result.Matrices {
		if !m.Complete || m.CR == nil {
			continue
		}
		if worst == nil || *m.CR > worst.CR {
			worst = &JournalWorstCR{Matrix: scope, CR: *m.CR}
		}
	}
	s.WorstCR = worst
	return s
}

func (w *Workspace) hashDataFiles() (string, error) {
	h := sha256.New()
	names := []string{
		"criteria.csv", "alternatives.csv", "pairwise.csv",
		"attributes.csv", "constraints.csv", "quotes.csv",
	}
	for _, name := range names {
		path := w.dataPath(name)
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		_, _ = io.WriteString(h, name)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (w *Workspace) appendJournalIndex(cfg JournalConfig, entry JournalIndexEntry) error {
	dir := filepath.Join(w.Root, cfg.Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, journalIndexName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func (w *Workspace) readJournalIndex(cfg JournalConfig) ([]JournalIndexEntry, error) {
	path := filepath.Join(w.Root, cfg.Dir, journalIndexName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []JournalIndexEntry
	sc := bufio.NewScanner(f)
	// journal lines are small; raise limit slightly for safety
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e JournalIndexEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

func (w *Workspace) rewriteJournalIndex(cfg JournalConfig, entries []JournalIndexEntry) error {
	dir := filepath.Join(w.Root, cfg.Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, journalIndexName)
	var b strings.Builder
	for _, e := range entries {
		jb, err := json.Marshal(e)
		if err != nil {
			return err
		}
		b.Write(jb)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// JournalList returns index entries newest-first. limit <= 0 means all.
func (w *Workspace) JournalList(limit int) ([]JournalIndexEntry, error) {
	cfg := w.ResolveJournalConfig(WriteOutputsOpts{})
	entries, err := w.readJournalIndex(cfg)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].CreatedAt == entries[j].CreatedAt {
			return entries[i].ID > entries[j].ID
		}
		return entries[i].CreatedAt > entries[j].CreatedAt
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

// JournalShow resolves id or "latest" and loads meta + summary.
func (w *Workspace) JournalShow(idOrLatest string) (*JournalEntry, error) {
	abs, _, err := w.resolveJournalEntry(idOrLatest)
	if err != nil {
		return nil, err
	}
	var meta JournalMeta
	if err := readJSONFile(filepath.Join(abs, "meta.json"), &meta); err != nil {
		return nil, err
	}
	var summary JournalSummary
	if err := readJSONFile(filepath.Join(abs, "summary.json"), &summary); err != nil {
		return nil, err
	}
	return &JournalEntry{Meta: meta, Summary: summary, Dir: abs}, nil
}

// JournalPath returns the absolute directory for id or "latest".
func (w *Workspace) JournalPath(idOrLatest string) (string, error) {
	abs, _, err := w.resolveJournalEntry(idOrLatest)
	return abs, err
}

// JournalPrune deletes oldest runs beyond keep. keep <= 0 uses config default.
func (w *Workspace) JournalPrune(keep int) (removed int, err error) {
	cfg := w.ResolveJournalConfig(WriteOutputsOpts{})
	if keep <= 0 {
		keep = cfg.Keep
	}
	entries, err := w.readJournalIndex(cfg)
	if err != nil {
		return 0, err
	}
	if len(entries) <= keep {
		return 0, nil
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].CreatedAt == entries[j].CreatedAt {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].CreatedAt < entries[j].CreatedAt
	})
	drop := entries[:len(entries)-keep]
	keepEntries := entries[len(entries)-keep:]
	for _, e := range drop {
		dir := filepath.Join(w.Root, filepath.FromSlash(e.Path))
		_ = os.RemoveAll(dir)
		removed++
	}
	sort.SliceStable(keepEntries, func(i, j int) bool {
		if keepEntries[i].CreatedAt == keepEntries[j].CreatedAt {
			return keepEntries[i].ID < keepEntries[j].ID
		}
		return keepEntries[i].CreatedAt < keepEntries[j].CreatedAt
	})
	if err := w.rewriteJournalIndex(cfg, keepEntries); err != nil {
		return removed, err
	}
	return removed, nil
}

func (w *Workspace) resolveJournalEntry(idOrLatest string) (absDir string, entry JournalIndexEntry, err error) {
	idOrLatest = strings.TrimSpace(idOrLatest)
	if idOrLatest == "" {
		return "", JournalIndexEntry{}, fmt.Errorf("journal: id required (or latest)")
	}
	cfg := w.ResolveJournalConfig(WriteOutputsOpts{})
	entries, err := w.readJournalIndex(cfg)
	if err != nil {
		return "", JournalIndexEntry{}, err
	}
	if len(entries) == 0 {
		return "", JournalIndexEntry{}, fmt.Errorf("journal: no entries")
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].CreatedAt == entries[j].CreatedAt {
			return entries[i].ID > entries[j].ID
		}
		return entries[i].CreatedAt > entries[j].CreatedAt
	})
	if strings.EqualFold(idOrLatest, "latest") {
		e := entries[0]
		return filepath.Join(w.Root, filepath.FromSlash(e.Path)), e, nil
	}
	for _, e := range entries {
		if e.ID == idOrLatest {
			return filepath.Join(w.Root, filepath.FromSlash(e.Path)), e, nil
		}
	}
	// Allow bare directory that exists even if index was pruned oddly.
	cand := filepath.Join(w.Root, cfg.Dir, idOrLatest)
	if st, err := os.Stat(cand); err == nil && st.IsDir() {
		return cand, JournalIndexEntry{ID: idOrLatest, Path: filepath.ToSlash(filepath.Join(cfg.Dir, idOrLatest))}, nil
	}
	return "", JournalIndexEntry{}, fmt.Errorf("journal: unknown id %q", idOrLatest)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func readJSONFile(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
