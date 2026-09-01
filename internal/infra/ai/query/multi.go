package query

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const multiQueryPrompt = `为以下问题生成 %d 个不同视角的检索变体,每行一个。规则:
- 覆盖不同关键词、不同表述、不同切入角度
- 不要重复原问题
- 只输出变体本身,不要编号或解释

原问题:%s`

func MultiQuery(ctx context.Context, llm LLM, q string, n int) []string {
	if n <= 0 {
		n = 3
	}
	out, err := llm.Complete(ctx, fmt.Sprintf(multiQueryPrompt, n, q))
	if err != nil {
		return []string{q}
	}
	variants := parseVariants(StripThink(out))
	if len(variants) == 0 {
		return []string{q}
	}
	return variants
}

var leadingPrefixRE = regexp.MustCompile(`^[\s\-•·]*(\d+[.)、]?)?\s*`)

func parseVariants(s string) []string {
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		line = leadingPrefixRE.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
