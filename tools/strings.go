package tools

import (
	"errors"
	"regexp"
	"strings"
)

func ToUpperFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func ToLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// SplitAndKeepDelimiters splits the string using the regex pattern and keeps the delimiters.
func SplitAndKeepDelimiters(input string, re *regexp.Regexp) ([]string, error) {
	if re == nil {
		return nil, errors.New("regex pattern is nil")
	}

	result := []string{}
	lastIndex := 0

	matches := re.FindAllStringIndex(input, -1)
	for _, match := range matches {
		start, end := match[0], match[1]

		if start > lastIndex {
			result = append(result, input[lastIndex:start]) // text before delimiter
		}
		result = append(result, input[start:end]) // the delimiter itself
		lastIndex = end
	}

	if lastIndex < len(input) {
		result = append(result, input[lastIndex:]) // trailing text
	}

	return result, nil
}
