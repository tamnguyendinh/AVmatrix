package parser

import "testing"

type mockCachedTree struct {
	id      string
	deleted int
}

func (t *mockCachedTree) Delete() {
	t.deleted++
}

func TestLegacyASTCacheGetSetEvictionClearAndStats(t *testing.T) {
	cache := NewASTCache(3)
	if got := cache.Get("missing.ts"); got != nil {
		t.Fatalf("cache miss = %#v, want nil", got)
	}
	if size, maxSize := cache.Stats(); size != 0 || maxSize != 3 {
		t.Fatalf("initial stats = %d/%d, want 0/3", size, maxSize)
	}

	a := &mockCachedTree{id: "a"}
	b := &mockCachedTree{id: "b"}
	c := &mockCachedTree{id: "c"}
	d := &mockCachedTree{id: "d"}
	cache.Set("a.ts", a)
	cache.Set("b.ts", b)
	cache.Set("c.ts", c)
	cache.Set("d.ts", d)
	if got := cache.Get("a.ts"); got != nil {
		t.Fatalf("a.ts should be evicted, got %#v", got)
	}
	if a.deleted != 1 {
		t.Fatalf("evicted tree delete count = %d, want 1", a.deleted)
	}
	if cache.Get("b.ts") != b || cache.Get("d.ts") != d {
		t.Fatalf("cache did not preserve b/d entries")
	}

	cache = NewASTCache(3)
	cache.Set("a.ts", a)
	cache.Set("b.ts", b)
	cache.Set("c.ts", c)
	cache.Get("a.ts")
	cache.Set("d.ts", d)
	if cache.Get("a.ts") != a {
		t.Fatalf("recently used a.ts was evicted")
	}
	if got := cache.Get("b.ts"); got != nil {
		t.Fatalf("b.ts should be LRU-evicted, got %#v", got)
	}

	replacement := &mockCachedTree{id: "replacement"}
	cache.Set("a.ts", replacement)
	if a.deleted != 2 {
		t.Fatalf("overwritten tree delete count = %d, want 2", a.deleted)
	}
	if cache.Get("a.ts") != replacement {
		t.Fatalf("replacement tree not returned")
	}
	cache.Clear()
	if size, _ := cache.Stats(); size != 0 {
		t.Fatalf("size after clear = %d, want 0", size)
	}
	if replacement.deleted != 1 {
		t.Fatalf("clear did not delete replacement tree")
	}
}

func TestLegacyASTCacheClampsMaxSize(t *testing.T) {
	cache := NewASTCache(0)
	if _, maxSize := cache.Stats(); maxSize != 1 {
		t.Fatalf("maxSize = %d, want 1", maxSize)
	}
	tree := &mockCachedTree{id: "tree"}
	cache.Set("a.ts", tree)
	if cache.Get("a.ts") != tree {
		t.Fatalf("clamped cache did not retain single entry")
	}
}
