package llm

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "learn/internal/infra/ai/llm"

type metrics struct {
	requestDuration metric.Float64Histogram
	ttft            metric.Float64Histogram
	interToken      metric.Float64Histogram
	inputTokens     metric.Int64Histogram
	outputTokens    metric.Int64Histogram
	requestCount    metric.Int64Counter
}

func newMetrics() (*metrics, error) {
	m := otel.Meter(meterName)

	requestDuration, err := m.Float64Histogram(
		"llm.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("end-to-end LLM request duration, including tool calls and reasoning"),
	)
	if err != nil {
		return nil, err
	}
	ttft, err := m.Float64Histogram(
		"llm.stream.time_to_first_token",
		metric.WithUnit("s"),
		metric.WithDescription("time from request sent to first content/tool_call delta"),
	)
	if err != nil {
		return nil, err
	}
	interToken, err := m.Float64Histogram(
		"llm.stream.inter_token",
		metric.WithUnit("s"),
		metric.WithDescription("average gap between subsequent output tokens: (duration - ttft) / max(output_tokens - 1, 1)"),
	)
	if err != nil {
		return nil, err
	}
	inputTokens, err := m.Int64Histogram(
		"llm.request.tokens.input",
		metric.WithUnit("token"),
		metric.WithDescription("prompt tokens consumed per request (from server Usage if reported)"),
	)
	if err != nil {
		return nil, err
	}
	outputTokens, err := m.Int64Histogram(
		"llm.request.tokens.output",
		metric.WithUnit("token"),
		metric.WithDescription("completion tokens generated per request (from server Usage if reported)"),
	)
	if err != nil {
		return nil, err
	}
	requestCount, err := m.Int64Counter(
		"llm.request.count",
		metric.WithDescription("LLM request count partitioned by status"),
	)
	if err != nil {
		return nil, err
	}

	return &metrics{
		requestDuration: requestDuration,
		ttft:            ttft,
		interToken:      interToken,
		inputTokens:     inputTokens,
		outputTokens:    outputTokens,
		requestCount:    requestCount,
	}, nil
}
