package chunk

import "learn/internal/domain/document"

func fixedSplit(text string, maxChars, overlap int) []document.Chunk {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return []document.Chunk{{Index: 0, Content: text}}
	}
	out := []document.Chunk{}
	for start := 0; start < len(runes); {
		end := min(start+maxChars, len(runes))
		out = append(out, document.Chunk{Index: document.ChunkIndex(len(out)), Content: string(runes[start:end])})
		if end == len(runes) {
			break
		}
		start = max(end-overlap, 0)
	}
	return out
}
