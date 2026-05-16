package cobol

import "strings"

type CopyReplacing struct {
	Type         string
	From         string
	To           string
	IsPseudotext bool
}

func parseReplacingClause(text string) []CopyReplacing {
	tokens := replacementScanner{text: text}
	replacements := []CopyReplacing{}
	for {
		tokens.skipSpace()
		if tokens.done() {
			break
		}
		replacementType := "EXACT"
		if word, ok := tokens.peekWord(); ok {
			upper := strings.ToUpper(word)
			if upper == "LEADING" || upper == "TRAILING" {
				replacementType = upper
				tokens.consumeWord()
				tokens.skipSpace()
			}
		}
		from, fromPseudo, ok := tokens.readValue()
		if !ok {
			break
		}
		tokens.skipSpace()
		if !tokens.consumeKeyword("BY") {
			break
		}
		tokens.skipSpace()
		to, toPseudo, ok := tokens.readValue()
		if !ok {
			break
		}
		replacements = append(replacements, CopyReplacing{
			Type:         replacementType,
			From:         from,
			To:           to,
			IsPseudotext: fromPseudo || toPseudo,
		})
	}
	return replacements
}

type replacementScanner struct {
	text string
	pos  int
}

func (scanner *replacementScanner) done() bool {
	return scanner.pos >= len(scanner.text)
}

func (scanner *replacementScanner) skipSpace() {
	for !scanner.done() {
		if scanner.text[scanner.pos] != ' ' && scanner.text[scanner.pos] != '\t' && scanner.text[scanner.pos] != '\r' && scanner.text[scanner.pos] != '\n' {
			return
		}
		scanner.pos++
	}
}

func (scanner *replacementScanner) peekWord() (string, bool) {
	pos := scanner.pos
	for pos < len(scanner.text) && isReplacementWordChar(scanner.text[pos]) {
		pos++
	}
	if pos == scanner.pos {
		return "", false
	}
	return scanner.text[scanner.pos:pos], true
}

func (scanner *replacementScanner) consumeWord() {
	for !scanner.done() && isReplacementWordChar(scanner.text[scanner.pos]) {
		scanner.pos++
	}
}

func (scanner *replacementScanner) consumeKeyword(keyword string) bool {
	word, ok := scanner.peekWord()
	if !ok || !strings.EqualFold(word, keyword) {
		return false
	}
	scanner.consumeWord()
	return true
}

func (scanner *replacementScanner) readValue() (string, bool, bool) {
	if scanner.done() {
		return "", false, false
	}
	if strings.HasPrefix(scanner.text[scanner.pos:], "==") {
		start := scanner.pos + 2
		end := strings.Index(scanner.text[start:], "==")
		if end < 0 {
			return "", true, false
		}
		scanner.pos = start + end + 2
		return scanner.text[start : start+end], true, true
	}
	if scanner.text[scanner.pos] == '"' || scanner.text[scanner.pos] == '\'' {
		quote := scanner.text[scanner.pos]
		start := scanner.pos + 1
		scanner.pos = start
		for !scanner.done() && scanner.text[scanner.pos] != quote {
			scanner.pos++
		}
		value := scanner.text[start:scanner.pos]
		if !scanner.done() {
			scanner.pos++
		}
		return value, false, true
	}
	start := scanner.pos
	for !scanner.done() && scanner.text[scanner.pos] != ' ' && scanner.text[scanner.pos] != '\t' && scanner.text[scanner.pos] != '\r' && scanner.text[scanner.pos] != '\n' {
		scanner.pos++
	}
	return scanner.text[start:scanner.pos], false, scanner.pos > start
}

func isReplacementWordChar(ch byte) bool {
	return ch == '_' || ch == '-' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
