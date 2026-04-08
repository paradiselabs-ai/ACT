package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
)

// ResponseCache is a process-local content-addressed cache for non-streaming
// LLM responses. It catches deterministic-input repeats — e.g. Assurance
// re-scoring an unchanged submission, QA re-synthesizing the same validated
// output, or Observer re-firing on an unchanged snapshot — without making
// the network call.
//
// The cache is intentionally conservative:
//   - Only used for SendMessages (non-streaming).
//   - Key is sha256(model + system + messages + tool names).
//   - TTL'd; size-bounded with crude FIFO eviction (no LRU complexity).
//
// It is NOT a substitute for provider-side prompt caching (Anthropic etc.) —
// it just elides repeat calls entirely. Provider caching elides re-prefill
// cost on calls that still happen; this cache elides the calls themselves.
type ResponseCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	order   []string // insertion order for FIFO eviction
	ttl     time.Duration
	maxSize int
	hits    int
	misses  int
}

type cacheEntry struct {
	response *ProviderResponse
	expires  time.Time
}

const (
	defaultCacheTTL     = 10 * time.Minute
	defaultCacheMaxSize = 200
)

// globalResponseCache is shared across all baseProvider instances. Tier 1
// agents share it, so e.g. two Assurance calls with identical input from
// different sessions still hit the same entry.
var globalResponseCache = NewResponseCache(defaultCacheTTL, defaultCacheMaxSize)

// NewResponseCache constructs a cache. Exposed for tests; production code
// uses globalResponseCache.
func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Key produces a stable hash of the inputs that determine an LLM response.
// We hash structured JSON rather than relying on Go's string formatting.
// Tool list is reduced to sorted names + descriptions — full schema would
// over-discriminate without changing semantics meaningfully.
func (c *ResponseCache) Key(model string, system string, messages []message.Message, toolList []tools.BaseTool) string {
	type keyInput struct {
		Model    string             `json:"model"`
		System   string             `json:"system"`
		Messages []message.Message  `json:"messages"`
		Tools    []string           `json:"tools"`
	}
	toolNames := make([]string, 0, len(toolList))
	for _, t := range toolList {
		if t == nil {
			continue
		}
		info := t.Info()
		toolNames = append(toolNames, info.Name)
	}
	ki := keyInput{
		Model:    model,
		System:   system,
		Messages: messages,
		Tools:    toolNames,
	}
	b, err := json.Marshal(ki)
	if err != nil {
		// Marshal failure → unhashable → return a sentinel that will never
		// match (no cache hits, no panics). Empty string disables caching
		// at the call site.
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Get returns a cached response if one exists and has not expired.
// Expired entries are evicted lazily on access.
func (c *ResponseCache) Get(key string) (*ProviderResponse, bool) {
	if key == "" {
		logging.Debug("response_cache.get.skip", "reason", "empty_key")
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		c.misses++
		logging.Debug("response_cache.miss",
			"key", shortKey(key),
			"hits", c.hits,
			"misses", c.misses,
			"size", len(c.entries),
		)
		return nil, false
	}
	if time.Now().After(e.expires) {
		delete(c.entries, key)
		c.misses++
		logging.Info("response_cache.miss",
			"reason", "expired",
			"key", shortKey(key),
			"hits", c.hits,
			"misses", c.misses,
			"size", len(c.entries),
		)
		return nil, false
	}
	c.hits++
	logging.Info("response_cache.hit",
		"key", shortKey(key),
		"hits", c.hits,
		"misses", c.misses,
		"size", len(c.entries),
		"content_bytes", len(e.response.Content),
		"expires_in_sec", int(time.Until(e.expires).Seconds()),
	)
	// Return a shallow copy so callers can mutate freely without poisoning
	// the cached entry.
	cp := *e.response
	return &cp, true
}

// Put stores a response under the given key. FIFO-evicts the oldest entry
// when the cache is at capacity.
func (c *ResponseCache) Put(key string, resp *ProviderResponse) {
	if key == "" || resp == nil {
		logging.Debug("response_cache.put.skip", "key_empty", key == "", "resp_nil", resp == nil)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	evicted := ""
	if _, exists := c.entries[key]; !exists {
		c.order = append(c.order, key)
		if len(c.order) > c.maxSize {
			evicted = c.order[0]
			c.order = c.order[1:]
			delete(c.entries, evicted)
		}
	}
	cp := *resp
	c.entries[key] = cacheEntry{
		response: &cp,
		expires:  time.Now().Add(c.ttl),
	}
	logging.Debug("response_cache.put",
		"key", shortKey(key),
		"size", len(c.entries),
		"evicted", shortKey(evicted),
		"ttl_sec", int(c.ttl.Seconds()),
		"content_bytes", len(resp.Content),
	)
}

// shortKey returns the first 12 chars of a hash key for log readability.
// Empty input returns empty string.
func shortKey(k string) string {
	if len(k) <= 12 {
		return k
	}
	return k[:12]
}

// Stats returns hit/miss counters for diagnostics. Cheap; safe to call
// from any goroutine.
func (c *ResponseCache) Stats() (hits int, misses int, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses, len(c.entries)
}
