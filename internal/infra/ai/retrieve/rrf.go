package retrieve

import "learn/internal/domain/document"

func rrfMerge(lists [][]document.Hit, topN int) []document.Hit {
	const k = 60.0
	scores := map[int64]float64{}
	order := map[int64]int{}
	hits := map[int64]document.Hit{}
	for _, list := range lists {
		for rank, h := range list {
			if _, seen := hits[h.ChunkID]; !seen {
				hits[h.ChunkID] = h
				order[h.ChunkID] = len(order)
			}
			scores[h.ChunkID] += 1.0 / (k + float64(rank+1))
		}
	}
	out := make([]document.Hit, 0, len(hits))
	for id, h := range hits {
		hh := h
		hh.Score = scores[id]
		out = append(out, hh)
	}
	sortRRF(out, order)
	if topN > 0 && topN < len(out) {
		out = out[:topN]
	}
	return out
}

func sortRRF(hits []document.Hit, order map[int64]int) {

	for i := 1; i < len(hits); i++ {
		cur := hits[i]
		j := i - 1
		for j >= 0 {
			if hits[j].Score < cur.Score || (hits[j].Score == cur.Score && order[hits[j].ChunkID] > order[cur.ChunkID]) {
				hits[j+1] = hits[j]
				j--
			} else {
				break
			}
		}
		hits[j+1] = cur
	}
}
