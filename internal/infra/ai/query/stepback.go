package query

import (
	"context"
	"fmt"
)

const stepBackPrompt = `把以下具体问题抽象成一个更通用、更利于检索的上层问题。
规则:
- 保留关键概念,移除具体的限定词
- 用更宽泛的术语替换具体术语
- 只输出抽象后的问题,不要任何解释

具体问题:%s
抽象问题:`

func StepBack(ctx context.Context, llm LLM, q string) (string, error) {
	out, err := llm.Complete(ctx, fmt.Sprintf(stepBackPrompt, q))
	if err != nil {
		return q, err
	}
	out = StripThink(out)
	if out == "" {
		return q, nil
	}
	return out, nil
}
