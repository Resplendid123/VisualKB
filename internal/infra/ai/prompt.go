package ai

import (
	_ "embed"
	"strings"

	"learn/internal/domain/conversation"
)

var systemPromptTmpl string

const portraitPlaceholder = "(none yet)"

func RenderSystemPrompt(skills []conversation.Skill, portrait string) string {
	out := strings.ReplaceAll(systemPromptTmpl, "{user_portrait}", fallback(portrait, portraitPlaceholder))
	out = strings.ReplaceAll(out, "{available_skills}", renderSkillList(skills))
	return out
}

func renderSkillList(skills []conversation.Skill) string {
	if len(skills) == 0 {
		return "(none yet)"
	}
	lines := make([]string, 0, len(skills))
	for _, s := range skills {
		lines = append(lines, "- "+s.Name+": "+s.Description)
	}
	return strings.Join(lines, "\n")
}

func fallback(s, dflt string) string {
	if strings.TrimSpace(s) == "" {
		return dflt
	}
	return s
}
