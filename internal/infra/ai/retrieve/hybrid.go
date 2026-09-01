package retrieve

import (
	"strconv"
	"strings"
	"time"

	"learn/internal/domain/document"
)

type hitRow struct {
	ChildID    int64     `gorm:"column:child_id"`
	ParentID   *int64    `gorm:"column:parent_id"`
	DocumentID int64     `gorm:"column:document_id"`
	Title      string    `gorm:"column:title"`
	Source     string    `gorm:"column:source"`
	Score      float64   `gorm:"column:score"`
	Content    string    `gorm:"column:content"`
	Header     *string   `gorm:"column:header"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func rowsToHits(rows []hitRow) []document.Hit {
	out := make([]document.Hit, 0, len(rows))
	for _, r := range rows {
		h := document.Hit{
			ChunkID:    r.ChildID,
			DocumentID: r.DocumentID,
			Title:      r.Title,
			Source:     r.Source,
			Content:    r.Content,
			Score:      r.Score,
		}
		if r.ParentID != nil {
			h.ParentID = *r.ParentID
		}
		if r.Header != nil {
			h.Header = *r.Header
		}
		out = append(out, h)
	}
	return out
}

func vecToPg(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}
