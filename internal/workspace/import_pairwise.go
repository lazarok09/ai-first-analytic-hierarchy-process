package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
)

// ImportPairwiseResult summarizes a bulk pairwise write.
type ImportPairwiseResult struct {
	Written int           `json:"written"`
	Rows    []PairwiseRow `json:"rows,omitempty"`
	Summary string        `json:"summary"`
}

// ImportPairwiseCSV reads a pairwise-shaped CSV (matrix,left,right,value,status,note)
// and upserts each row. DefaultStatus is used when a row's status is empty.
func (w *Workspace) ImportPairwiseCSV(path, defaultStatus string) (*ImportPairwiseResult, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	if defaultStatus == "" {
		defaultStatus = "proposal"
	}
	var items []PairwiseRow
	for i, r := range rows {
		row, err := pairwiseFromMap(r, defaultStatus)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+2, err) // +2: header + 1-based
		}
		items = append(items, row)
	}
	return w.importPairwiseRows(items)
}

// ImportPairwiseJSON reads a JSON array of PairwiseRow (or {rows:[...]}) and upserts.
func (w *Workspace) ImportPairwiseJSON(path, defaultStatus string) (*ImportPairwiseResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if defaultStatus == "" {
		defaultStatus = "proposal"
	}
	var items []PairwiseRow
	if err := json.Unmarshal(b, &items); err != nil {
		var wrap struct {
			Rows []PairwiseRow `json:"rows"`
		}
		if err2 := json.Unmarshal(b, &wrap); err2 != nil || len(wrap.Rows) == 0 {
			return nil, fmt.Errorf("pairwise json: %w", err)
		}
		items = wrap.Rows
	}
	for i := range items {
		if items[i].Status == "" {
			items[i].Status = defaultStatus
		}
	}
	return w.importPairwiseRows(items)
}

func (w *Workspace) importPairwiseRows(items []PairwiseRow) (*ImportPairwiseResult, error) {
	written := 0
	for _, row := range items {
		if err := w.UpsertPairwise(row); err != nil {
			return nil, fmt.Errorf("%s %s/%s: %w", row.Matrix, row.Left, row.Right, err)
		}
		written++
	}
	out := &ImportPairwiseResult{
		Written: written,
		Rows:    items,
		Summary: fmt.Sprintf("imported %d pairwise judgment(s)", written),
	}
	return out, nil
}

func pairwiseFromMap(r map[string]string, defaultStatus string) (PairwiseRow, error) {
	matrix := strings.TrimSpace(r["matrix"])
	left := strings.TrimSpace(r["left"])
	right := strings.TrimSpace(r["right"])
	if matrix == "" || left == "" || right == "" {
		return PairwiseRow{}, fmt.Errorf("matrix/left/right required")
	}
	raw := strings.TrimSpace(r["value"])
	v, err := parseSaatyCell(raw)
	if err != nil {
		return PairwiseRow{}, err
	}
	status := strings.TrimSpace(r["status"])
	if status == "" {
		status = defaultStatus
	}
	return PairwiseRow{
		Matrix: matrix, Left: left, Right: right,
		Value: v, Status: status, Note: strings.TrimSpace(r["note"]),
	}, nil
}

func parseSaatyCell(raw string) (float64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty Saaty value")
	}
	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil || b == 0 {
			return 0, fmt.Errorf("bad Saaty fraction %q", raw)
		}
		v := a / b
		if !engine.IsSaatyValue(v) {
			return 0, fmt.Errorf("not a Saaty value: %q", raw)
		}
		return v, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("bad Saaty value %q", raw)
	}
	if !engine.IsSaatyValue(v) {
		return 0, fmt.Errorf("not a Saaty value: %q", raw)
	}
	return v, nil
}
