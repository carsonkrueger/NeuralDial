package tools

import (
	"regexp"
	"strings"
)

var sentenceEndPattern = regexp.MustCompile(`[.!?]["')\]]?\s*$`)
var clauseEndPattern = regexp.MustCompile(`[;,:-]["')\]]?\s*$`)

func IsBoundary(text string, desperate bool) bool {
	trimmed := strings.TrimSpace(text)
	if sentenceEndPattern.MatchString(trimmed) {
		return true
	}
	words := strings.Count(" ", trimmed) + 1
	if desperate && words > 3 && clauseEndPattern.MatchString(trimmed) {
		return true
	}
	return false
}
