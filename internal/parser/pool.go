package parser

import (
	"sync"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const defaultParsersPerGrammar = 4

type PoolOptions struct {
	MaxParsersPerGrammar int
	ParseTimeout         time.Duration
	CountNodes           bool
}

type Pool struct {
	registry *Registry
	options  PoolOptions

	mu      sync.Mutex
	parsers map[string]chan *sitter.Parser
	metrics Metrics
}

func NewPool(registry *Registry, options PoolOptions) *Pool {
	if registry == nil {
		registry = DefaultRegistry()
	}
	if options.MaxParsersPerGrammar <= 0 {
		options.MaxParsersPerGrammar = defaultParsersPerGrammar
	}
	return &Pool{
		registry: registry,
		options:  options,
		parsers:  make(map[string]chan *sitter.Parser),
	}
}

func (p *Pool) SnapshotMetrics() Metrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.metrics
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, pool := range p.parsers {
		close(pool)
		for parser := range pool {
			parser.Close()
		}
		delete(p.parsers, key)
	}
}

func (p *Pool) acquire(grammar Grammar) (*sitter.Parser, error) {
	p.mu.Lock()
	pool := p.parsers[grammar.Key]
	if pool == nil {
		pool = make(chan *sitter.Parser, p.options.MaxParsersPerGrammar)
		p.parsers[grammar.Key] = pool
	}
	p.mu.Unlock()

	select {
	case parser := <-pool:
		return parser, nil
	default:
		parser := sitter.NewParser()
		if err := parser.SetLanguage(grammar.build()); err != nil {
			parser.Close()
			return nil, err
		}
		p.recordCreatedParser()
		return parser, nil
	}
}

func (p *Pool) release(grammar Grammar, parser *sitter.Parser) {
	if parser == nil {
		return
	}
	parser.Reset()
	p.mu.Lock()
	pool := p.parsers[grammar.Key]
	p.mu.Unlock()
	if pool == nil {
		parser.Close()
		return
	}
	select {
	case pool <- parser:
	default:
		parser.Close()
	}
}

func (p *Pool) recordCreatedParser() {
	p.mu.Lock()
	p.metrics.CreatedParsers++
	p.mu.Unlock()
}

func (p *Pool) recordResult(result parseMetric) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics.Total++
	p.metrics.TotalBytes += int64(result.bytes)
	p.metrics.TotalDuration += result.duration
	if result.unsupported {
		p.metrics.Unsupported++
		p.metrics.Failed++
		return
	}
	if result.timedOut {
		p.metrics.TimedOut++
		p.metrics.Failed++
		return
	}
	if result.failed {
		p.metrics.Failed++
		return
	}
	p.metrics.Succeeded++
}

type parseMetric struct {
	bytes       int
	duration    time.Duration
	failed      bool
	unsupported bool
	timedOut    bool
}
