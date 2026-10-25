package research

import (
	"testing"
	"time"
)

func TestCacheRoundTripsAndExpires(t *testing.T) {
	c := NewCache(t.TempDir())
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	if err := c.Put("serp", "lemon|us|en", map[string]int{"n": 1}); err != nil {
		t.Fatal(err)
	}
	var got map[string]int
	if !c.Get("serp", "lemon|us|en", time.Hour, &got) || got["n"] != 1 {
		t.Fatalf("Get = %v", got)
	}
	now = now.Add(2 * time.Hour)
	if c.Get("serp", "lemon|us|en", time.Hour, &got) {
		t.Fatal("expired entry returned")
	}
	if c.Get("serp", "other", time.Hour, &got) {
		t.Fatal("missing key returned")
	}
}

func TestNilCacheIsANoOp(t *testing.T) {
	var c *Cache
	if err := c.Put("b", "k", 1); err != nil {
		t.Fatal(err)
	}
	var v int
	if c.Get("b", "k", time.Hour, &v) {
		t.Fatal("nil cache returned a value")
	}
}
