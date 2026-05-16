package lbugruntime

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var writeKeywords = map[string]struct{}{
	"CREATE":  {},
	"DELETE":  {},
	"SET":     {},
	"MERGE":   {},
	"REMOVE":  {},
	"DROP":    {},
	"ALTER":   {},
	"COPY":    {},
	"DETACH":  {},
	"FOREACH": {},
	"INSTALL": {},
	"LOAD":    {},
}

func IsWriteQuery(query string) bool {
	var previous rune
	for index := 0; index < len(query); {
		current, size := utf8.DecodeRuneInString(query[index:])
		if !isIdentStart(current) {
			previous = current
			index += size
			continue
		}
		end := index + size
		for end < len(query) {
			next, size := utf8.DecodeRuneInString(query[end:])
			if !isIdentPart(next) {
				break
			}
			end += size
		}
		token := strings.ToUpper(query[index:end])
		if _, ok := writeKeywords[token]; ok && previous != ':' {
			return true
		}
		previous = 0
		index = end
	}
	return false
}

func ValidateReadQuery(query string) error {
	if IsWriteQuery(query) {
		return fmt.Errorf("write operations are not allowed")
	}
	return nil
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
