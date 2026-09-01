package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"

	"learn/internal/domain/document"
)

// Replace \f with markdown sep.
const pageSeparator = "\n\n---\n\n"

func init() {
	document.RegisterPDFExtractor(pdfExtractor{})
}

type pdfExtractor struct{}

func (pdfExtractor) Extract(_ context.Context, rc io.Reader) (string, error) {
	raw, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("read pdf bytes: %w", err)
	}
	if len(raw) == 0 {
		return "", nil
	}
	r, err := pdf.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	pr, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text: %w", err)
	}
	out, err := io.ReadAll(pr)
	if err != nil {
		return "", fmt.Errorf("read pdf text: %w", err)
	}
	return strings.ReplaceAll(string(out), "\f", pageSeparator), nil
}
