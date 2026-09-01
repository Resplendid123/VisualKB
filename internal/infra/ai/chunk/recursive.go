package chunk

import (
	"strings"

	"learn/internal/domain/document"
)

func recursiveSplit(text string, maxChars, overlap int) []document.Chunk {
	seps := []string{"\n\n", "\n", "。", ".", "，", ",", " "}
	return splitRecursive(text, maxChars, overlap, seps)
}

func splitRecursive(text string, maxChars, overlap int, seps []string) []document.Chunk {
	if len(text) <= maxChars {
		return []document.Chunk{{Index: 0, Content: text}}
	}
	if len(seps) == 0 {
		return fixedSplit(text, maxChars, overlap)
	}
	sep := seps[0]
	rest := seps[1:]
	parts := strings.Split(text, sep)
	var out []document.Chunk
	var buf strings.Builder
	for _, p := range parts {
		if buf.Len()+len(p)+len(sep) > maxChars {
			if buf.Len() > 0 {
				out = append(out, document.Chunk{Index: document.ChunkIndex(len(out)), Content: buf.String()})
				buf.Reset()
			}
			if len(p) > maxChars {
				out = append(out, splitRecursive(p, maxChars, overlap, rest)...)
			} else {
				buf.WriteString(p)
			}
		} else {
			if buf.Len() > 0 {
				buf.WriteString(sep)
			}
			buf.WriteString(p)
		}
	}
	if buf.Len() > 0 {
		out = append(out, document.Chunk{Index: document.ChunkIndex(len(out)), Content: buf.String()})
	}
	return out
}
