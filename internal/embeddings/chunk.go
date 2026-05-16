package embeddings

import (
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Chunk struct {
	Text        string
	ChunkIndex  int
	StartOffset int
	EndOffset   int
	StartLine   int
	EndLine     int
}

func ChunkNode(node EmbeddableNode, config Config) []Chunk {
	config = NormalizeConfig(config)
	startLine := node.StartLine
	endLine := node.EndLine
	if IsShortLabel(node.Label) || len(node.Content) <= config.ChunkSize {
		return []Chunk{{
			Text:        node.Content,
			ChunkIndex:  0,
			StartOffset: 0,
			EndOffset:   len(node.Content),
			StartLine:   startLine,
			EndLine:     endLine,
		}}
	}
	if chunks := structuralChunks(node.Label, node.Content, startLine, config.ChunkSize); len(chunks) > 0 {
		return chunks
	}
	return CharacterChunks(node.Content, startLine, endLine, config.ChunkSize, config.Overlap)
}

func CharacterChunks(content string, startLine int, endLine int, chunkSize int, overlap int) []Chunk {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if overlap < 0 {
		overlap = DefaultOverlap
	}
	if content == "" || len(content) <= chunkSize {
		return []Chunk{{Text: content, ChunkIndex: 0, StartOffset: 0, EndOffset: len(content), StartLine: startLine, EndLine: endLine}}
	}

	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	chunks := make([]Chunk, 0, (len(content)/step)+1)
	for offset := 0; offset < len(content); offset += step {
		chunkEnd := min(offset+chunkSize, len(content))
		chunks = append(chunks, Chunk{
			Text:        content[offset:chunkEnd],
			ChunkIndex:  len(chunks),
			StartOffset: offset,
			EndOffset:   chunkEnd,
			StartLine:   lineAtOffset(content, startLine, offset),
			EndLine:     lineAtOffset(content, startLine, max(chunkEnd-1, offset)),
		})
		if chunkEnd == len(content) {
			break
		}
	}
	return chunks
}

func structuralChunks(label scopeir.NodeLabel, content string, startLine int, chunkSize int) []Chunk {
	if chunkSize <= 0 || len(content) <= chunkSize {
		return nil
	}
	switch label {
	case scopeir.NodeClass, scopeir.NodeInterface:
		return chunkLineSegments(content, declarationMemberSegments(content), startLine, chunkSize, false)
	case scopeir.NodeFunction, scopeir.NodeConstructor, scopeir.NodeMethod:
		return chunkLineSegments(content, functionBodySegments(content), startLine, chunkSize, true)
	default:
		return nil
	}
}

func declarationMemberSegments(content string) []lineSegment {
	lines := lineSegments(content)
	openLine, closeLine := declarationBodyLines(lines)
	if openLine < 0 || closeLine < 0 || closeLine <= openLine+1 {
		return nil
	}
	return lines[openLine+1 : closeLine]
}

func functionBodySegments(content string) []lineSegment {
	lines := lineSegments(content)
	openLine, closeLine := declarationBodyLines(lines)
	if openLine < 0 || closeLine < 0 || closeLine <= openLine {
		return nil
	}
	return lines[openLine : closeLine+1]
}

func chunkLineSegments(content string, segments []lineSegment, startLine int, chunkSize int, includeFirstHeader bool) []Chunk {
	segments = nonBlankSegments(segments)
	if len(segments) == 0 {
		return nil
	}
	chunks := make([]Chunk, 0)
	currentStart := -1
	currentEnd := -1
	headerEnd := -1
	for _, segment := range segments {
		nextStart := segment.Start
		if currentStart >= 0 {
			nextStart = currentStart
		}
		if includeFirstHeader && currentStart < 0 && headerEnd >= 0 {
			nextStart = headerEnd
		}
		nextLen := segment.End - nextStart
		if currentStart >= 0 && nextLen > chunkSize {
			chunks = appendStructuralChunk(chunks, content, currentStart, currentEnd, startLine)
			currentStart = -1
		}
		if currentStart < 0 {
			currentStart = segment.Start
			if includeFirstHeader && len(chunks) == 0 {
				currentStart = 0
				headerEnd = segment.Start
			}
		}
		currentEnd = segment.End
	}
	if currentStart >= 0 && currentEnd > currentStart {
		chunks = appendStructuralChunk(chunks, content, currentStart, currentEnd, startLine)
	}
	if len(chunks) <= 1 {
		return nil
	}
	return chunks
}

func appendStructuralChunk(chunks []Chunk, content string, startOffset int, endOffset int, startLine int) []Chunk {
	startOffset = trimLeadingBlank(content, startOffset, endOffset)
	endOffset = trimTrailingBlank(content, startOffset, endOffset)
	if endOffset <= startOffset {
		return chunks
	}
	return append(chunks, Chunk{
		Text:        content[startOffset:endOffset],
		ChunkIndex:  len(chunks),
		StartOffset: startOffset,
		EndOffset:   endOffset,
		StartLine:   lineAtOffset(content, startLine, startOffset),
		EndLine:     lineAtOffset(content, startLine, max(endOffset-1, startOffset)),
	})
}

type lineSegment struct {
	Start int
	End   int
	Text  string
}

func lineSegments(content string) []lineSegment {
	lines := strings.SplitAfter(content, "\n")
	segments := make([]lineSegment, 0, len(lines))
	offset := 0
	for _, line := range lines {
		end := offset + len(line)
		segments = append(segments, lineSegment{Start: offset, End: end, Text: line})
		offset = end
	}
	return segments
}

func declarationBodyLines(lines []lineSegment) (int, int) {
	openLine := -1
	depth := 0
	for i, line := range lines {
		if openLine < 0 && strings.Contains(line.Text, "{") {
			openLine = i
		}
		depth += strings.Count(line.Text, "{")
		depth -= strings.Count(line.Text, "}")
		if openLine >= 0 && depth <= 0 {
			return openLine, i
		}
	}
	return openLine, -1
}

func nonBlankSegments(segments []lineSegment) []lineSegment {
	filtered := make([]lineSegment, 0, len(segments))
	for _, segment := range segments {
		if strings.TrimSpace(segment.Text) != "" {
			filtered = append(filtered, segment)
		}
	}
	return filtered
}

func trimLeadingBlank(content string, startOffset int, endOffset int) int {
	for startOffset < endOffset && (content[startOffset] == '\n' || content[startOffset] == '\r') {
		startOffset++
	}
	return startOffset
}

func trimTrailingBlank(content string, startOffset int, endOffset int) int {
	for endOffset > startOffset && (content[endOffset-1] == '\n' || content[endOffset-1] == '\r') {
		endOffset--
	}
	return endOffset
}

func lineAtOffset(content string, startLine int, offset int) int {
	if startLine <= 0 {
		startLine = 0
	}
	if offset <= 0 {
		return startLine
	}
	if offset > len(content) {
		offset = len(content)
	}
	return startLine + strings.Count(content[:offset], "\n")
}
