package scopeir

import "sort"

type PositionIndex struct {
	entriesByFile map[string][]positionEntry
	size          int
}

type positionEntry struct {
	id     string
	range_ Range
}

func BuildPositionIndex(scopes []ScopeFact) PositionIndex {
	entriesByFile := make(map[string][]positionEntry)
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if _, ok := seen[scope.ID]; ok {
			continue
		}
		seen[scope.ID] = struct{}{}
		entriesByFile[scope.FilePath] = append(entriesByFile[scope.FilePath], positionEntry{
			id:     scope.ID,
			range_: scope.Range,
		})
	}
	for filePath := range entriesByFile {
		sort.Slice(entriesByFile[filePath], func(i, j int) bool {
			return comparePositionEntry(entriesByFile[filePath][i], entriesByFile[filePath][j]) < 0
		})
	}
	return PositionIndex{entriesByFile: entriesByFile, size: len(seen)}
}

func (idx PositionIndex) Size() int {
	return idx.size
}

func (idx PositionIndex) AtPosition(filePath string, line int, col int) (string, bool) {
	bucket := idx.entriesByFile[filePath]
	if len(bucket) == 0 {
		return "", false
	}
	endIndex := findLastStartAtOrBefore(bucket, line, col)
	if endIndex < 0 {
		return "", false
	}
	for index := endIndex; index >= 0; index-- {
		entry := bucket[index]
		if endIsAtOrAfter(entry.range_, line, col) {
			return entry.id, true
		}
	}
	return "", false
}

func comparePositionEntry(left positionEntry, right positionEntry) int {
	if value := compareInt(left.range_.StartLine, right.range_.StartLine); value != 0 {
		return value
	}
	if value := compareInt(left.range_.StartCol, right.range_.StartCol); value != 0 {
		return value
	}
	if value := compareInt(right.range_.EndLine, left.range_.EndLine); value != 0 {
		return value
	}
	return compareInt(right.range_.EndCol, left.range_.EndCol)
}

func findLastStartAtOrBefore(entries []positionEntry, line int, col int) int {
	lo := 0
	hi := len(entries)
	for lo < hi {
		mid := (lo + hi) / 2
		if startIsAtOrBefore(entries[mid].range_, line, col) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo - 1
}

func startIsAtOrBefore(scopeRange Range, line int, col int) bool {
	if scopeRange.StartLine < line {
		return true
	}
	if scopeRange.StartLine > line {
		return false
	}
	return scopeRange.StartCol <= col
}

func endIsAtOrAfter(scopeRange Range, line int, col int) bool {
	if scopeRange.EndLine > line {
		return true
	}
	if scopeRange.EndLine < line {
		return false
	}
	return scopeRange.EndCol >= col
}
