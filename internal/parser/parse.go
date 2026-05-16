package parser

import (
	"context"
	"errors"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

var (
	ErrUnsupportedLanguage = errors.New("unsupported parser language")
	ErrParseCanceled       = errors.New("parse canceled")
)

type Request struct {
	FilePath string
	Language scanner.Language
	Source   []byte
}

type Result struct {
	FilePath  string           `json:"filePath"`
	Language  scanner.Language `json:"language"`
	Grammar   string           `json:"grammar"`
	RootKind  string           `json:"rootKind"`
	HasError  bool             `json:"hasError"`
	NodeCount int              `json:"nodeCount"`
	Bytes     int              `json:"bytes"`
	Duration  time.Duration    `json:"duration"`
	Tree      *sitter.Tree     `json:"-"`
}

func (r *Result) Close() {
	if r != nil && r.Tree != nil {
		r.Tree.Close()
		r.Tree = nil
	}
}

func (p *Pool) Parse(ctx context.Context, request Request) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	start := time.Now()
	metric := parseMetric{bytes: len(request.Source)}
	defer func() {
		metric.duration = time.Since(start)
		p.recordResult(metric)
	}()

	grammar, err := p.registry.Resolve(request.Language, request.FilePath)
	if err != nil {
		metric.unsupported = true
		return nil, err
	}

	if p.options.ParseTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.options.ParseTimeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		metric.timedOut = errors.Is(err, context.DeadlineExceeded)
		metric.failed = !metric.timedOut
		return nil, err
	}

	tsParser, err := p.acquire(grammar)
	if err != nil {
		metric.failed = true
		return nil, err
	}
	defer p.release(grammar, tsParser)

	tree := tsParser.ParseWithOptions(
		func(offset int, _ sitter.Point) []byte {
			if offset >= len(request.Source) {
				return []byte{}
			}
			return request.Source[offset:]
		},
		nil,
		&sitter.ParseOptions{
			ProgressCallback: func(_ sitter.ParseState) bool {
				return ctx.Err() != nil
			},
		},
	)
	if tree == nil {
		err := ctx.Err()
		if err == nil {
			err = ErrParseCanceled
		}
		metric.timedOut = errors.Is(err, context.DeadlineExceeded)
		metric.failed = !metric.timedOut
		return nil, err
	}

	root := tree.RootNode()
	nodeCount := 0
	if p.options.CountNodes {
		nodeCount = countNodes(root)
	}
	result := &Result{
		FilePath:  request.FilePath,
		Language:  request.Language,
		Grammar:   grammar.Key,
		RootKind:  root.Kind(),
		HasError:  root.HasError(),
		NodeCount: nodeCount,
		Bytes:     len(request.Source),
		Duration:  time.Since(start),
		Tree:      tree,
	}
	return result, nil
}

func countNodes(node *sitter.Node) int {
	if node == nil {
		return 0
	}
	total := 1
	for index := uint(0); index < node.ChildCount(); index++ {
		total += countNodes(node.Child(index))
	}
	return total
}
