package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func RemoveSubstring(input string, start, end int) string {
	if start < 0 || end > len(input) || start >= end {
		return input
	}
	return input[:start] + input[end:]
}

func isEscaped(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\\' {
			switch s[i+1] {
			case 'n', 't', 'r', '"', '\\':
				return true
			}
		}
	}
	return false
}

func UnescapeIfNeeded(s string) string {
	if isEscaped(s) {
		if !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
			s = "\"" + s + "\""
		}
		unescaped, err := strconv.Unquote(s)
		if err != nil {
			fmt.Println("Error unquoting string:", err)
			return s
		}
		return unescaped
	}
	return s
}
