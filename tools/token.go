package tools

import (
	"regexp"
	"strings"
)

var sentenceEndPattern = regexp.MustCompile(`[.!?]["')\]]?\s*$`)
var clauseEndPattern = regexp.MustCompile(`[;,:-]["')\]]?\s*$`)

func IsBoundary(text string, desperate bool) bool {
	trimmed := strings.TrimSpace(text)
	words := strings.Count(" ", trimmed) + 1
	if sentenceEndPattern.MatchString(trimmed) && words > 3 {
		return true
	}
	if desperate && words > 3 && clauseEndPattern.MatchString(trimmed) {
		return true
	}
	return false
}
