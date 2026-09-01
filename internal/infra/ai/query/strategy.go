package query

import (
	"context"
)

type Strategy string

const (
	StrategyNone     Strategy = "none"
	StrategyRewrite  Strategy = "rewrite"
	StrategyMulti    Strategy = "multi"
	StrategyHyDE     Strategy = "hyde"
	StrategyStepBack Strategy = "stepback"
)

type LLM interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type Transformer struct {
	llm      LLM
	strategy Strategy
	multiN   int
}

func NewTransformer(llm LLM, strategy Strategy, multiN int) *Transformer {
	if strategy == "" {
		strategy = StrategyNone
	}
	if multiN <= 0 {
		multiN = 3
	}
	return &Transformer{llm: llm, strategy: strategy, multiN: multiN}
}

func (t *Transformer) Apply(ctx context.Context, q string) []string {
	switch t.strategy {
	case StrategyRewrite:
		out, err := RewriteQuery(ctx, t.llm, q, nil)
		if err != nil || out == "" {
			return []string{q}
		}
		return []string{out}
	case StrategyMulti:
		variants := MultiQuery(ctx, t.llm, q, t.multiN)
		out := make([]string, 0, len(variants)+1)
		out = append(out, q)
		out = append(out, variants...)
		return out
	case StrategyHyDE:
		out, err := HyDEAnswer(ctx, t.llm, q)
		if err != nil || out == "" {
			return []string{q}
		}
		return []string{out}
	case StrategyStepBack:
		out, err := StepBack(ctx, t.llm, q)
		if err != nil || out == "" {
			return []string{q}
		}
		return []string{out}
	default:
		return []string{q}
	}
}
