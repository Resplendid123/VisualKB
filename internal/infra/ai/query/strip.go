package query

import "regexp"

var thinkRE = regexp.MustCompile(`(?is)<think>.*?</think>`)

func StripThink(s string) string {
	return thinkRE.ReplaceAllString(s, "")
}
