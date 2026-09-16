package engine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const CRAccept = 0.10

var saatyIntensities = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

var riTable = map[int]float64{
	1: 0.00, 2: 0.00, 3: 0.58, 4: 0.90, 5: 1.12,
	6: 1.24, 7: 1.32, 8: 1.41, 9: 1.45, 10: 1.49,
	11: 1.51, 12: 1.48, 13: 1.56, 14: 1.57, 15: 1.59,
}

func RandomIndex(n int) float64 {
	if n <= 0 {
		return 0
	}
	if n >= 15 {
		return riTable[15]
	}
	return riTable[n]
}

type PairKey struct {
	Left, Right string
}

type RepairHint struct {
	Left      string  `json:"left"`
	Right     string  `json:"right"`
	Current   float64 `json:"current"`
	Implied   float64 `json:"implied"`
	Suggested float64 `json:"suggested"`
	Error     float64 `json:"error"`
}

type MatrixResult struct {
	IDs       []string
	Weights   map[string]float64
	LambdaMax float64
	CI        float64
	CR        float64
	Complete  bool
	Missing   []PairKey
	Matrix    [][]float64
	Repairs   []RepairHint
}

func (r MatrixResult) Consistent() bool {
	return r.Complete && !math.IsNaN(r.CR) && r.CR <= CRAccept
}

func IsSaatyValue(v float64) bool {
	if v <= 0 {
		return false
	}
	const tol = 1e-9
	if v >= 1 {
		for _, i := range saatyIntensities {
			if math.Abs(v-float64(i)) < tol {
				return true
			}
		}
		return false
	}
	for _, i := range saatyIntensities {
		if math.Abs(v-1/float64(i)) < tol {
			return true
		}
	}
	return false
}

func OrderedPairs(ids []string) []PairKey {
	out := make([]PairKey, 0)
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			out = append(out, PairKey{Left: a, Right: b})
		}
	}
	return out
}

func Lookup(pairs map[PairKey]float64, left, right string) (float64, bool) {
	if v, ok := pairs[PairKey{left, right}]; ok {
		return v, true
	}
	if v, ok := pairs[PairKey{right, left}]; ok {
		return 1 / v, true
	}
	return 0, false
}

func MissingPairs(ids []string, pairs map[PairKey]float64) []PairKey {
	var missing []PairKey
	for _, p := range OrderedPairs(ids) {
		if _, ok := Lookup(pairs, p.Left, p.Right); !ok {
			missing = append(missing, p)
		}
	}
	return missing
}

func BuildMatrix(ids []string, pairs map[PairKey]float64) [][]float64 {
	n := len(ids)
	a := make([][]float64, n)
	index := make(map[string]int, n)
	for i, id := range ids {
		a[i] = make([]float64, n)
		a[i][i] = 1
		index[id] = i
	}
	for i, left := range ids {
		for _, right := range ids[i+1:] {
			v, ok := Lookup(pairs, left, right)
			if !ok {
				continue
			}
			j := index[right]
			a[i][j] = v
			a[j][i] = 1 / v
		}
	}
	return a
}

func PrincipalEigenvector(matrix [][]float64) (weights []float64, lambda float64) {
	n := len(matrix)
	if n == 0 {
		return nil, 0
	}
	if n == 1 {
		return []float64{1}, 1
	}
	w := make([]float64, n)
	for i := range w {
		w[i] = 1.0 / float64(n)
	}
	tmp := make([]float64, n)
	for iter := 0; iter < 100; iter++ {
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := 0; j < n; j++ {
				sum += matrix[i][j] * w[j]
			}
			tmp[i] = sum
		}
		total := 0.0
		for _, v := range tmp {
			total += v
		}
		if total <= 0 {
			break
		}
		maxDiff := 0.0
		for i := 0; i < n; i++ {
			nw := tmp[i] / total
			if d := math.Abs(nw - w[i]); d > maxDiff {
				maxDiff = d
			}
			w[i] = nw
		}
		if maxDiff < 1e-12 {
			break
		}
	}
	Aw := make([]float64, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			Aw[i] += matrix[i][j] * w[j]
		}
	}
	num, den := 0.0, 0.0
	for i := 0; i < n; i++ {
		num += Aw[i] * w[i]
		den += w[i] * w[i]
	}
	if den > 0 {
		lambda = num / den
	}
	return w, lambda
}

func Consistency(lambdaMax float64, n int) (ci, cr float64) {
	if n <= 2 {
		return 0, 0
	}
	ci = (lambdaMax - float64(n)) / float64(n-1)
	ri := RandomIndex(n)
	if ri == 0 {
		return ci, 0
	}
	return ci, ci / ri
}

func NearestSaaty(value float64) float64 {
	if value <= 0 {
		return 1
	}
	best := 1.0
	bestErr := math.Inf(1)
	for _, i := range saatyIntensities {
		for _, c := range []float64{float64(i), 1 / float64(i)} {
			err := math.Abs(math.Log(c) - math.Log(value))
			if err < bestErr {
				bestErr = err
				best = c
			}
		}
	}
	return best
}

func SuggestRepairs(ids []string, weights map[string]float64, pairs map[PairKey]float64, limit int) []RepairHint {
	var hints []RepairHint
	for _, p := range OrderedPairs(ids) {
		cur, ok := Lookup(pairs, p.Left, p.Right)
		if !ok {
			continue
		}
		wl, wr := weights[p.Left], weights[p.Right]
		if wl <= 0 || wr <= 0 {
			continue
		}
		implied := wl / wr
		suggested := NearestSaaty(implied)
		err := math.Abs(math.Log(cur) - math.Log(implied))
		if err < 1e-9 {
			continue
		}
		hints = append(hints, RepairHint{
			Left: p.Left, Right: p.Right,
			Current: cur, Implied: implied, Suggested: suggested, Error: err,
		})
	}
	sort.Slice(hints, func(i, j int) bool { return hints[i].Error > hints[j].Error })
	if limit > 0 && len(hints) > limit {
		hints = hints[:limit]
	}
	return hints
}

func SolvePairwise(ids []string, pairs map[PairKey]float64) MatrixResult {
	missing := MissingPairs(ids, pairs)
	complete := len(missing) == 0
	if len(ids) == 0 {
		return MatrixResult{Weights: map[string]float64{}, Complete: true, Matrix: [][]float64{}}
	}
	matrix := BuildMatrix(ids, pairs)
	if !complete {
		eq := 1.0 / float64(len(ids))
		w := make(map[string]float64, len(ids))
		for _, id := range ids {
			w[id] = eq
		}
		return MatrixResult{
			IDs: ids, Weights: w,
			LambdaMax: math.NaN(), CI: math.NaN(), CR: math.NaN(),
			Complete: false, Missing: missing, Matrix: matrix,
		}
	}
	vec, lam := PrincipalEigenvector(matrix)
	ci, cr := Consistency(lam, len(ids))
	w := make(map[string]float64, len(ids))
	for i, id := range ids {
		w[id] = vec[i]
	}
	var repairs []RepairHint
	if cr > CRAccept {
		repairs = SuggestRepairs(ids, w, pairs, 5)
	}
	return MatrixResult{
		IDs: ids, Weights: w, LambdaMax: lam, CI: ci, CR: cr,
		Complete: true, Matrix: matrix, Repairs: repairs,
	}
}

func Synthesize(criterionWeights map[string]float64, local map[string]map[string]float64, altIDs []string) map[string]float64 {
	totals := make(map[string]float64, len(altIDs))
	for _, alt := range altIDs {
		totals[alt] = 0
	}
	for cid, cw := range criterionWeights {
		loc := local[cid]
		for _, alt := range altIDs {
			totals[alt] += cw * loc[alt]
		}
	}
	return totals
}

func FormatSaaty(v float64) string {
	if v >= 1 {
		r := math.Round(v)
		if math.Abs(v-r) < 1e-9 {
			return strconv.Itoa(int(r))
		}
	}
	inv := 1 / v
	r := math.Round(inv)
	if math.Abs(inv-r) < 1e-9 {
		return "1/" + strconv.Itoa(int(r))
	}
	return strconv.FormatFloat(v, 'g', 6, 64)
}

func ParseSaaty(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		num, err1 := strconv.ParseFloat(strings.TrimSpace(raw[:i]), 64)
		den, err2 := strconv.ParseFloat(strings.TrimSpace(raw[i+1:]), 64)
		if err1 != nil {
			return 0, err1
		}
		if err2 != nil {
			return 0, err2
		}
		if den == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return num / den, nil
	}
	return strconv.ParseFloat(raw, 64)
}
