package helper

import (
	"strings"
	"unicode"
)

const TagMaxLength = 128

func Sanitize(tag string) string {
	// Tronquer la chaîne si elle dépasse la longueur maximale
	if len(tag) > TagMaxLength {
		tag = tag[:TagMaxLength]
	}

	var sb strings.Builder
	for _, r := range tag {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			sb.WriteRune(r)
		} else if r == ' ' {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}
