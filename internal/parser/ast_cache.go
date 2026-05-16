package parser

import "container/list"

type CachedTree interface {
	Delete()
}

type ASTCache struct {
	maxSize int
	items   map[string]*list.Element
	order   *list.List
}

type astCacheEntry struct {
	path string
	tree CachedTree
}

func NewASTCache(maxSize int) *ASTCache {
	if maxSize < 1 {
		maxSize = 1
	}
	return &ASTCache{
		maxSize: maxSize,
		items:   make(map[string]*list.Element),
		order:   list.New(),
	}
}

func (c *ASTCache) Get(filePath string) CachedTree {
	if c == nil {
		return nil
	}
	element := c.items[filePath]
	if element == nil {
		return nil
	}
	c.order.MoveToFront(element)
	return element.Value.(astCacheEntry).tree
}

func (c *ASTCache) Set(filePath string, tree CachedTree) {
	if c == nil {
		return
	}
	if element := c.items[filePath]; element != nil {
		entry := element.Value.(astCacheEntry)
		if entry.tree != nil && entry.tree != tree {
			entry.tree.Delete()
		}
		element.Value = astCacheEntry{path: filePath, tree: tree}
		c.order.MoveToFront(element)
		return
	}
	element := c.order.PushFront(astCacheEntry{path: filePath, tree: tree})
	c.items[filePath] = element
	for len(c.items) > c.maxSize {
		c.removeOldest()
	}
}

func (c *ASTCache) Clear() {
	if c == nil {
		return
	}
	for filePath, element := range c.items {
		entry := element.Value.(astCacheEntry)
		if entry.tree != nil {
			entry.tree.Delete()
		}
		delete(c.items, filePath)
	}
	c.order.Init()
}

func (c *ASTCache) Stats() (int, int) {
	if c == nil {
		return 0, 0
	}
	return len(c.items), c.maxSize
}

func (c *ASTCache) removeOldest() {
	element := c.order.Back()
	if element == nil {
		return
	}
	entry := element.Value.(astCacheEntry)
	if entry.tree != nil {
		entry.tree.Delete()
	}
	delete(c.items, entry.path)
	c.order.Remove(element)
}
