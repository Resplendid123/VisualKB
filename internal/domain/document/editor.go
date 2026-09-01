package document

import (
	"context"
	"strings"

	"learn/internal/domain"
)

type EditOp interface {
	Apply(content string) (string, error)
}

// Anchor must match uniquely unless ReplaceAll.
type ReplaceAnchorOp struct {
	Anchor     string `json:"anchor"`
	NewText    string `json:"new_text"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func (o ReplaceAnchorOp) Apply(content string) (string, error) {
	if o.Anchor == "" {
		return "", domain.ErrDocumentAnchorNotFound
	}
	count := strings.Count(content, o.Anchor)
	if count == 0 {
		return "", domain.ErrDocumentAnchorNotFound
	}
	if !o.ReplaceAll && count > 1 {
		return "", domain.ErrDocumentAnchorAmbiguous
	}
	return strings.ReplaceAll(content, o.Anchor, o.NewText), nil
}

type AppendOp struct {
	Text        string `json:"text"`
	WantNewline bool   `json:"want_newline,omitempty"`
}

func (o AppendOp) Apply(content string) (string, error) {
	if o.WantNewline && content != "" && !strings.HasSuffix(content, "\n") {
		return content + "\n" + o.Text, nil
	}
	return content + o.Text, nil
}

type WholeReplaceOp struct {
	Content string `json:"content"`
}

func (o WholeReplaceOp) Apply(_ string) (string, error) {
	if o.Content == "" {
		return "", domain.ErrDocumentEmptyContent
	}
	return o.Content, nil
}

// Shared by HTTP and agent tool JSON.
const (
	OpTypeReplaceAnchor = "replace_anchor"
	OpTypeAppend        = "append"
	OpTypeWholeReplace  = "whole_replace"
)

func ParseEditOp(opType string, args map[string]any) (EditOp, error) {
	switch opType {
	case OpTypeReplaceAnchor:
		anchor, _ := args["anchor"].(string)
		newText, _ := args["new_text"].(string)
		ra, _ := args["replace_all"].(bool)
		return ReplaceAnchorOp{Anchor: anchor, NewText: newText, ReplaceAll: ra}, nil
	case OpTypeAppend:
		text, _ := args["text"].(string)
		wn, _ := args["want_newline"].(bool)
		return AppendOp{Text: text, WantNewline: wn}, nil
	case OpTypeWholeReplace:
		c, _ := args["content"].(string)
		return WholeReplaceOp{Content: c}, nil
	default:
		return nil, domain.ErrDocumentUnknownEditOp
	}
}

// Owner and source checks live in implementations.
type Editor interface {
	ReadCurrent(ctx context.Context, userID, docID int64) (string, error)
	ApplyEdits(ctx context.Context, userID, docID int64, ops []EditOp, title string) (int, error)
}

type Creator interface {
	CreateText(ctx context.Context, userID int64, p CreateParams) (int64, error)
}

// Excludes archived by default.
type Lister interface {
	List(ctx context.Context, userID int64, opts ListOpts) ([]*Document, int64, error)
}
