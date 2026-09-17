package workspace

import (
	"fmt"
	"sort"
	"strings"
)

// TreeView is a hierarchy snapshot for `ahp tree` (human ASCII or JSON).
type TreeView struct {
	Workspace    string           `json:"workspace"`
	Title        string           `json:"title"`
	Criteria     []TreeCriterion  `json:"criteria"`
	Alternatives []TreeAlternative `json:"alternatives"`
	Matrices     []TreeMatrix     `json:"matrices"`
}

// TreeCriterion is one criterion node; children when parent_id is used.
type TreeCriterion struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	ParentID  string          `json:"parent_id,omitempty"`
	Children  []TreeCriterion `json:"children,omitempty"`
	AltMatrix string          `json:"alt_matrix,omitempty"` // set on leaves: alt:<id>
	HasPairs  bool            `json:"has_pairs"`            // pairwise rows exist for AltMatrix
}

// TreeAlternative is a decision alternative in the tree listing.
type TreeAlternative struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TreeMatrix summarizes a pairwise matrix key and whether judgments exist.
type TreeMatrix struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"` // criteria | subcriteria | alternatives
	HasPairs bool   `json:"has_pairs"`
}

// Tree builds an ASCII/JSON hierarchy: goal → criteria (±children) → alt matrices.
func (w *Workspace) Tree() (*TreeView, error) {
	meta, err := w.LoadMeta()
	if err != nil {
		return nil, err
	}
	criteria, err := w.Criteria()
	if err != nil {
		return nil, err
	}
	alternatives, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	pairwise, err := w.Pairwise()
	if err != nil {
		return nil, err
	}

	present := map[string]bool{}
	for _, p := range pairwise {
		if p.Matrix != "" {
			present[p.Matrix] = true
		}
	}

	byParent := map[string][]Criterion{}
	var roots []Criterion
	for _, c := range criteria {
		if c.ParentID == "" {
			roots = append(roots, c)
		} else {
			byParent[c.ParentID] = append(byParent[c.ParentID], c)
		}
	}
	sortCriteria(roots)
	for pid := range byParent {
		sortCriteria(byParent[pid])
	}

	var matrices []TreeMatrix
	matrices = append(matrices, TreeMatrix{
		Key: "criteria", Kind: "criteria", HasPairs: present["criteria"],
	})

	nodes := make([]TreeCriterion, 0, len(roots))
	for _, r := range roots {
		nodes = append(nodes, buildCriterionNode(r, byParent, present, &matrices))
	}

	alts := make([]TreeAlternative, 0, len(alternatives))
	for _, a := range alternatives {
		alts = append(alts, TreeAlternative{ID: a.ID, Name: a.Name})
	}

	return &TreeView{
		Workspace:    w.Root,
		Title:        meta.Title,
		Criteria:     nodes,
		Alternatives: alts,
		Matrices:     matrices,
	}, nil
}

func buildCriterionNode(c Criterion, byParent map[string][]Criterion, present map[string]bool, matrices *[]TreeMatrix) TreeCriterion {
	children := byParent[c.ID]
	node := TreeCriterion{ID: c.ID, Name: c.Name, ParentID: c.ParentID}
	if len(children) == 0 {
		key := "alt:" + c.ID
		node.AltMatrix = key
		node.HasPairs = present[key]
		*matrices = append(*matrices, TreeMatrix{Key: key, Kind: "alternatives", HasPairs: node.HasPairs})
		return node
	}
	subKey := "criteria:" + c.ID
	*matrices = append(*matrices, TreeMatrix{
		Key: subKey, Kind: "subcriteria", HasPairs: present[subKey],
	})
	node.Children = make([]TreeCriterion, 0, len(children))
	for _, ch := range children {
		node.Children = append(node.Children, buildCriterionNode(ch, byParent, present, matrices))
	}
	return node
}

func sortCriteria(items []Criterion) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

// FormatTree renders a kubectl/tree-style ASCII hierarchy.
func FormatTree(t *TreeView) string {
	if t == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", t.Title)
	fmt.Fprintf(&b, "workspace: %s\n", t.Workspace)
	fmt.Fprintf(&b, "matrix criteria %s\n", pairMark(matrixHas(t.Matrices, "criteria")))

	n := len(t.Criteria)
	for i, c := range t.Criteria {
		writeCriterion(&b, c, "", i == n-1)
	}

	fmt.Fprintf(&b, "\nAlternatives (%d)\n", len(t.Alternatives))
	an := len(t.Alternatives)
	if an == 0 {
		b.WriteString("  (none)\n")
	} else {
		for i, a := range t.Alternatives {
			branch := "├── "
			if i == an-1 {
				branch = "└── "
			}
			name := a.Name
			if name == "" {
				name = a.ID
			}
			fmt.Fprintf(&b, "%s%s  %s\n", branch, a.ID, name)
		}
	}

	fmt.Fprintf(&b, "\nMatrices (%d)\n", len(t.Matrices))
	for _, m := range t.Matrices {
		fmt.Fprintf(&b, "  %s  %s  %s\n", m.Key, m.Kind, pairMark(m.HasPairs))
	}
	return b.String()
}

func writeCriterion(b *strings.Builder, c TreeCriterion, prefix string, last bool) {
	branch := "├── "
	nextPrefix := prefix + "│   "
	if last {
		branch = "└── "
		nextPrefix = prefix + "    "
	}
	label := c.Name
	if label == "" {
		label = c.ID
	}
	fmt.Fprintf(b, "%s%s%s  %s", prefix, branch, c.ID, label)
	if c.AltMatrix != "" {
		fmt.Fprintf(b, "  [%s %s]", c.AltMatrix, pairMark(c.HasPairs))
	} else if len(c.Children) > 0 {
		sub := "criteria:" + c.ID
		fmt.Fprintf(b, "  [%s]", sub)
	}
	b.WriteByte('\n')

	cn := len(c.Children)
	for i, ch := range c.Children {
		writeCriterion(b, ch, nextPrefix, i == cn-1)
	}
}

func pairMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func matrixHas(matrices []TreeMatrix, key string) bool {
	for _, m := range matrices {
		if m.Key == key {
			return m.HasPairs
		}
	}
	return false
}
