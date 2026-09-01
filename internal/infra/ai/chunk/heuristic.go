package chunk

import "learn/internal/domain/document"

func heuristicSplit(text string, maxChars, overlap int) []document.Chunk {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return []document.Chunk{{Index: 0, Content: text}}
	}
	out := []document.Chunk{}
	for start := 0; start < len(runes); {
		end := start + maxChars
		if end >= len(runes) {
			end = len(runes)
		} else {

			for i := end; i > start+maxChars/2; i-- {
				if i < len(runes) && (runes[i-1] == '。' || runes[i-1] == '.' || runes[i-1] == '\n') {
					end = i
					break
				}
			}
		}
		out = append(out, document.Chunk{Index: document.ChunkIndex(len(out)), Content: string(runes[start:end])})
		if end == len(runes) {
			break
		}
		start = max(end-overlap, 0)
	}
	return out
}
