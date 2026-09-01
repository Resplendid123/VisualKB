package query

import (
	"context"
	"fmt"
	"strings"
)

const rewritePrompt = `你是检索查询改写器。规则:
- 保持原意不变,只改述法
- 禁止添加新概念、限定词或猜测
- 输出适合关键词检索的形式(名词化、补全口语省略)
- 只输出改写后的问题,不要任何解释

对话历史:
%s
用户问题:%s
改写后:`

func RewriteQuery(ctx context.Context, llm LLM, q string, history []string) (string, error) {
	out, err := llm.Complete(ctx, fmt.Sprintf(rewritePrompt, strings.Join(history, "\n"), q))
	if err != nil {
		return q, err
	}
	out = StripThink(out)
	if out == "" {
		return q, nil
	}
	return out, nil
}
