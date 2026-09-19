package workspace

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
)

// QuoteRow is one candidate price/source for an alternative×criterion.
// Multiple quotes may exist; PickQuote writes the winner into attributes.csv.
type QuoteRow struct {
	AlternativeID string `json:"alternative_id"`
	CriterionID   string `json:"criterion_id"`
	Value         string `json:"value"`
	Unit          string `json:"unit"`
	Source        string `json:"source"`
	Note          string `json:"note"`
	URL           string `json:"url,omitempty"`
}

func (w *Workspace) Quotes() ([]QuoteRow, error) {
	rows, err := readCSV(w.dataPath("quotes.csv"))
	if err != nil {
		return nil, err
	}
	var out []QuoteRow
	for _, r := range rows {
		alt := strings.TrimSpace(r["alternative_id"])
		crit := strings.TrimSpace(r["criterion_id"])
		if alt == "" || crit == "" {
			continue
		}
		out = append(out, QuoteRow{
			AlternativeID: alt,
			CriterionID:   crit,
			Value:         strings.TrimSpace(r["value"]),
			Unit:          strings.TrimSpace(r["unit"]),
			Source:        strings.TrimSpace(r["source"]),
			Note:          strings.TrimSpace(r["note"]),
			URL:           strings.TrimSpace(r["url"]),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AlternativeID != out[j].AlternativeID {
			return out[i].AlternativeID < out[j].AlternativeID
		}
		if out[i].CriterionID != out[j].CriterionID {
			return out[i].CriterionID < out[j].CriterionID
		}
		return out[i].Source < out[j].Source
	})
	return out, nil
}

func (w *Workspace) WriteQuotes(items []QuoteRow) error {
	rows := make([]map[string]string, 0, len(items))
	for _, q := range items {
		rows = append(rows, map[string]string{
			"alternative_id": q.AlternativeID,
			"criterion_id":   q.CriterionID,
			"value":          q.Value,
			"unit":           q.Unit,
			"source":         q.Source,
			"note":           q.Note,
			"url":            q.URL,
		})
	}
	return writeCSV(w.dataPath("quotes.csv"),
		[]string{"alternative_id", "criterion_id", "value", "unit", "source", "note", "url"}, rows)
}

// UpsertQuote replaces a quote with the same alt+criterion+source, or appends.
func (w *Workspace) UpsertQuote(item QuoteRow) error {
	if item.AlternativeID == "" || item.CriterionID == "" {
		return fmt.Errorf("alternative_id and criterion_id are required")
	}
	if strings.TrimSpace(item.Source) == "" {
		return fmt.Errorf("source is required for quotes (name the shop/page)")
	}
	if _, err := engine.ParseNumericAttribute(item.Value); err != nil {
		return fmt.Errorf("value: %w", err)
	}
	items, err := w.Quotes()
	if err != nil {
		return err
	}
	kept := items[:0]
	for _, q := range items {
		if q.AlternativeID == item.AlternativeID && q.CriterionID == item.CriterionID &&
			strings.EqualFold(q.Source, item.Source) {
			continue
		}
		kept = append(kept, q)
	}
	kept = append(kept, item)
	return w.WriteQuotes(kept)
}

// PickQuoteResult lists which attribute rows were written from quotes.csv.
type PickQuoteResult struct {
	CriterionID string            `json:"criterion_id"`
	Prefer      string            `json:"prefer"`
	Picked      []AttributeRow    `json:"picked"`
	Skipped     []string          `json:"skipped,omitempty"`
	Summary     string            `json:"summary"`
}

// PickQuote selects the best numeric quote per alternative for a criterion
// (min if prefer=lower, max if prefer=higher) and writes attributes.csv.
func (w *Workspace) PickQuote(criterionID, prefer string) (*PickQuoteResult, error) {
	if criterionID == "" {
		return nil, fmt.Errorf("criterion is required")
	}
	if prefer != "higher" && prefer != "lower" {
		return nil, fmt.Errorf("prefer must be higher or lower")
	}
	quotes, err := w.Quotes()
	if err != nil {
		return nil, err
	}
	byAlt := map[string][]QuoteRow{}
	for _, q := range quotes {
		if q.CriterionID != criterionID {
			continue
		}
		byAlt[q.AlternativeID] = append(byAlt[q.AlternativeID], q)
	}
	out := &PickQuoteResult{CriterionID: criterionID, Prefer: prefer}
	if len(byAlt) == 0 {
		return nil, fmt.Errorf("no quotes for criterion %q — ahp quote add …", criterionID)
	}
	for alt, list := range byAlt {
		bestIdx := -1
		var bestVal float64
		for i, q := range list {
			v, err := engine.ParseNumericAttribute(q.Value)
			if err != nil {
				out.Skipped = append(out.Skipped, fmt.Sprintf("%s/%s: %v", alt, q.Source, err))
				continue
			}
			if bestIdx < 0 ||
				(prefer == "lower" && v < bestVal) ||
				(prefer == "higher" && v > bestVal) {
				bestIdx = i
				bestVal = v
			}
		}
		if bestIdx < 0 {
			continue
		}
		q := list[bestIdx]
		note := q.Note
		if note == "" {
			note = fmt.Sprintf("picked from %d quote(s); prefer=%s", len(list), prefer)
		} else {
			note = note + fmt.Sprintf(" [picked prefer=%s among %d]", prefer, len(list))
		}
		if q.URL != "" && !strings.Contains(note, q.URL) {
			note = note + " " + q.URL
		}
		// Discarded competitors in note for audit.
		if len(list) > 1 {
			var others []string
			for i, o := range list {
				if i == bestIdx {
					continue
				}
				others = append(others, fmt.Sprintf("%s=%s", o.Source, o.Value))
			}
			note = note + "; discarded: " + strings.Join(others, ", ")
		}
		row := AttributeRow{
			AlternativeID: alt, CriterionID: criterionID,
			Value: q.Value, Unit: q.Unit, Source: q.Source, Note: note,
		}
		if err := w.UpsertAttribute(row); err != nil {
			return nil, err
		}
		out.Picked = append(out.Picked, row)
	}
	sort.Slice(out.Picked, func(i, j int) bool {
		return out.Picked[i].AlternativeID < out.Picked[j].AlternativeID
	})
	out.Summary = fmt.Sprintf("picked %d attribute(s) on %s (%s-better)", len(out.Picked), criterionID, prefer)
	return out, nil
}

// ShareCard is one alternative line for pasteable summaries.
type ShareCard struct {
	Rank   int               `json:"rank"`
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Weight float64           `json:"weight"`
	Attrs  map[string]string `json:"attrs,omitempty"`
	URL    string            `json:"url,omitempty"`
}

// ShareSummary is a paste-friendly ranking snapshot (WhatsApp / chat).
type ShareSummary struct {
	Title        string      `json:"title"`
	Complete     bool        `json:"complete"`
	Consistent   bool        `json:"consistent"`
	Cards        []ShareCard `json:"cards"`
	WhatsAppText string      `json:"whatsapp_text"`
}

// Share builds a ranking summary with attribute highlights and any URL found
// in quote/attribute source or note.
func (w *Workspace) Share(includeProposals bool, topN int) (*ShareSummary, error) {
	if topN <= 0 {
		topN = 5
	}
	meta, err := w.LoadMeta()
	if err != nil {
		return nil, err
	}
	res, err := w.Compute(includeProposals)
	if err != nil {
		return nil, err
	}
	attrs, err := w.Attributes()
	if err != nil {
		return nil, err
	}
	quotes, _ := w.Quotes()
	attrBy := map[string]AttributeRow{}
	for _, a := range attrs {
		attrBy[a.AlternativeID+"|"+a.CriterionID] = a
	}
	urlByAlt := map[string]string{}
	for _, q := range quotes {
		if q.URL != "" {
			urlByAlt[q.AlternativeID] = q.URL
		}
	}
	out := &ShareSummary{
		Title: meta.Title, Complete: res.Complete, Consistent: res.Consistent,
	}
	n := topN
	if n > len(res.Ranking) {
		n = len(res.Ranking)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "*%s*\n", meta.Title)
	if !res.Complete {
		b.WriteString("⚠️ ranking incomplete\n")
	}
	b.WriteString("\n")
	medals := []string{"🥇", "🥈", "🥉"}
	for i := 0; i < n; i++ {
		r := res.Ranking[i]
		card := ShareCard{
			Rank: r.Rank, ID: r.ID, Name: r.Name, Weight: r.Weight,
			Attrs: map[string]string{},
		}
		for _, a := range attrs {
			if a.AlternativeID != r.ID {
				continue
			}
			label := a.Value
			if a.Unit != "" {
				label = label + " " + a.Unit
			}
			if a.Source != "" {
				label = label + " (" + a.Source + ")"
			}
			card.Attrs[a.CriterionID] = label
			if card.URL == "" {
				card.URL = firstURL(a.Source, a.Note, urlByAlt[r.ID])
			}
		}
		if card.URL == "" {
			card.URL = urlByAlt[r.ID]
		}
		out.Cards = append(out.Cards, card)

		medal := strconv.Itoa(r.Rank) + "."
		if r.Rank >= 1 && r.Rank <= 3 {
			medal = medals[r.Rank-1]
		}
		fmt.Fprintf(&b, "%s *%s*\n", medal, r.Name)
		if v, ok := card.Attrs["value"]; ok {
			fmt.Fprintf(&b, "💵 %s\n", v)
		}
		for crit, v := range card.Attrs {
			if crit == "value" {
				continue
			}
			fmt.Fprintf(&b, "• %s: %s\n", crit, v)
		}
		if card.URL != "" {
			fmt.Fprintf(&b, "🔗 %s\n", card.URL)
		}
		b.WriteString("\n")
	}
	out.WhatsAppText = strings.TrimSpace(b.String())
	return out, nil
}

func firstURL(parts ...string) string {
	for _, p := range parts {
		for _, tok := range strings.Fields(p) {
			if strings.HasPrefix(tok, "http://") || strings.HasPrefix(tok, "https://") {
				if u, err := url.Parse(tok); err == nil && u.Host != "" {
					return tok
				}
			}
		}
	}
	return ""
}
