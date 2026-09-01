package query

import (
	"context"
	"fmt"
)

const hydePrompt = `基于你的知识,用一段话简洁回答以下问题，回答风格使用正式书面语,陈述句为主。
问题:%s
回答:`

func HyDEAnswer(ctx context.Context, llm LLM, q string) (string, error) {
	out, err := llm.Complete(ctx, fmt.Sprintf(hydePrompt, q))
	if err != nil {
		return q, err
	}
	out = StripThink(out)
	if out == "" {
		return q, nil
	}
	return out, nil
}
