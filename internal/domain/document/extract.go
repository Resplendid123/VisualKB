package document

import (
	"context"
	"fmt"
	"io"
)

func Extract(ctx context.Context, ct ContentType, rc io.Reader) (string, error) {
	switch ct {
	case "", ContentTypeMarkdown:
		b, err := io.ReadAll(rc)
		if err != nil {
			return "", fmt.Errorf("read markdown: %w", err)
		}
		return string(b), nil
	case ContentTypePDF:
		// Registered by infra to avoid domain dependency.
		return pdfExtractor.Extract(ctx, rc)
	default:
		return "", fmt.Errorf("unsupported content_type %q", ct)
	}
}

type Extractor interface {
	Extract(ctx context.Context, rc io.Reader) (string, error)
}

var pdfExtractor Extractor = errExtractor{}

type errExtractor struct{}

func (errExtractor) Extract(_ context.Context, _ io.Reader) (string, error) {
	return "", fmt.Errorf("pdf extractor not registered")
}

func RegisterPDFExtractor(e Extractor) {
	if e == nil {
		panic("RegisterPDFExtractor: nil extractor")
	}
	pdfExtractor = e
}
