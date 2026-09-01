package chunk

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"

	"learn/internal/domain/document"
)

func markdownSplit(text string, maxChars, overlap int) ([]document.Chunk, string) {
	src := []byte(text)
	root := goldmark.New().Parser().Parse(gmtext.NewReader(src))
	type section struct {
		heading string
		body    string
	}
	var sections []section
	var currentHeading string
	var currentBody strings.Builder
	collect := func() {
		body := strings.TrimSpace(currentBody.String())
		if body != "" {
			sections = append(sections, section{heading: currentHeading, body: body})
		}
		currentBody.Reset()
	}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok {
			collect()
			currentHeading = string(h.Text(src))
			return ast.WalkSkipChildren, nil
		}
		if t, ok := n.(*ast.Text); ok {
			currentBody.Write(t.Text(src))
			currentBody.WriteByte('\n')
		}
		return ast.WalkContinue, nil
	})
	collect()
	if len(sections) == 0 {
		return nil, ""
	}
	var out []document.Chunk
	for _, sec := range sections {
		body := sec.body
		if len([]rune(body)) > maxChars {
			for _, c := range heuristicSplit(body, maxChars, overlap) {
				if sec.heading != "" {
					c.Header = sec.heading
				}
				out = append(out, c)
			}
		} else {
			out = append(out, document.Chunk{Index: document.ChunkIndex(len(out)), Content: body, Header: sec.heading})
		}
	}
	if len(out) == 0 {
		return nil, ""
	}
	return out, sections[len(sections)-1].heading
}
