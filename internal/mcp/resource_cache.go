package mcp

import (
	"os"
	"sync"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
)

type resourceGraphCache struct {
	mu      sync.Mutex
	entries map[string]*resourceGraphCacheEntry
}

type resourceGraphCacheEntry struct {
	stat       resourceGraphCacheStat
	graph      *graph.Graph
	routeIndex *mcpRouteIndex
}

type resourceGraphCacheStat struct {
	modTime time.Time
	size    int64
}

func newResourceGraphCache() *resourceGraphCache {
	return &resourceGraphCache{entries: map[string]*resourceGraphCacheEntry{}}
}

func (c *resourceGraphCache) graph(path string) (*graph.Graph, error) {
	stat, err := resourceGraphStat(path)
	if err != nil {
		return nil, err
	}
	if cached := c.cachedGraph(path, stat); cached != nil {
		return cached, nil
	}

	g, err := loadResourceGraphSnapshot(path)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[path] = &resourceGraphCacheEntry{stat: stat, graph: g}
	c.mu.Unlock()
	return g, nil
}

func (c *resourceGraphCache) routeIndex(path string) (*mcpRouteIndex, error) {
	stat, err := resourceGraphStat(path)
	if err != nil {
		return nil, err
	}
	if cached := c.cachedRouteIndex(path, stat); cached != nil {
		return cached, nil
	}

	g, err := c.graph(path)
	if err != nil {
		return nil, err
	}
	index := buildMCPRouteIndex(g)

	c.mu.Lock()
	entry := c.entries[path]
	if entry == nil || entry.stat != stat {
		entry = &resourceGraphCacheEntry{stat: stat, graph: g}
		c.entries[path] = entry
	}
	entry.routeIndex = index
	c.mu.Unlock()
	return index, nil
}

func (c *resourceGraphCache) cachedGraph(path string, stat resourceGraphCacheStat) *graph.Graph {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.entries[path]
	if entry == nil || entry.stat != stat {
		return nil
	}
	return entry.graph
}

func (c *resourceGraphCache) cachedRouteIndex(path string, stat resourceGraphCacheStat) *mcpRouteIndex {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.entries[path]
	if entry == nil || entry.stat != stat {
		return nil
	}
	return entry.routeIndex
}

func resourceGraphStat(path string) (resourceGraphCacheStat, error) {
	info, err := os.Stat(path)
	if err != nil {
		return resourceGraphCacheStat{}, err
	}
	return resourceGraphCacheStat{modTime: info.ModTime(), size: info.Size()}, nil
}
