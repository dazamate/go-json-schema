package research

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Cache keeps fetched results on disk as JSON files, one per key, so
// repeated lookups skip the browser. A nil Cache stores nothing.
type Cache struct {
	dir string
	now func() time.Time
}

// NewCache returns a cache under dir.
func NewCache(dir string) *Cache { return &Cache{dir: dir, now: time.Now} }

type entry struct {
	Saved time.Time       `json:"saved"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func (c *Cache) path(bucket, key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, bucket, hex.EncodeToString(sum[:16])+".json")
}

// Get decodes the value saved under key into out, unless it is missing or
// older than ttl.
func (c *Cache) Get(bucket, key string, ttl time.Duration, out any) bool {
	if c == nil {
		return false
	}
	data, err := os.ReadFile(c.path(bucket, key))
	if err != nil {
		return false
	}
	var e entry
	if json.Unmarshal(data, &e) != nil || e.Key != key || c.now().Sub(e.Saved) > ttl {
		return false
	}
	return json.Unmarshal(e.Value, out) == nil
}

// Put saves v under key.
func (c *Cache) Put(bucket, key string, v any) error {
	if c == nil {
		return nil
	}
	val, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data, err := json.Marshal(entry{Saved: c.now(), Key: key, Value: val})
	if err != nil {
		return err
	}
	p := c.path(bucket, key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
