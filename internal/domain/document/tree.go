package document

import (
	"time"
	"unicode"
	"unicode/utf8"

	"learn/internal/domain"
)

// DocID points to documents.id when not folder.
type TreeNode struct {
	ID        int64
	UserID    int64
	ParentID  *int64
	Name      string
	IsFolder  bool
	DocID     *int64
	CreatedAt time.Time
}

const MaxFolderDepth = 3

const treeNameMaxLen = 64

// Allows 1-64 runes, no control characters.
func ValidateName(name string) error {
	if name == "" {
		return domain.ErrTreeNodeInvalidName
	}
	if utf8.RuneCountInString(name) > treeNameMaxLen {
		return domain.ErrTreeNodeInvalidName
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return domain.ErrTreeNodeInvalidName
		}
	}
	return nil
}

// Trims whitespace; errors when empty.
func NormalizeName(name string) (string, error) {
	trimmed := trimAll(name)
	if trimmed == "" {
		return "", domain.ErrTreeNodeInvalidName
	}
	if err := ValidateName(trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

func trimAll(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
