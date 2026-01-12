package secrets

import (
	"context"
	"sync"
	"time"
)

// CachedSecretProvider wraps a SecretProvider with caching functionality.
//
// Features:
// - LRU eviction when cache is full
// - TTL-based expiration
// - Thread-safe concurrent access
// - Background cleanup of expired entries
type CachedSecretProvider struct {
	provider SecretProvider
	cache    map[string]*cacheEntry
	config   *CacheConfig
	mu       sync.RWMutex
	stopCh   chan struct{}
}

type cacheEntry struct {
	value        string
	expiresAt    time.Time
	lastAccessed time.Time
}

// NewCachedSecretProvider creates a cached provider wrapper.
func NewCachedSecretProvider(provider SecretProvider, config *CacheConfig) *CachedSecretProvider {
	if config == nil {
		config = &CacheConfig{
			Enabled: true,
			TTL:     300,
			MaxSize: 1000,
		}
	}

	// Set defaults if not specified
	if config.TTL == 0 {
		config.TTL = 300
	}
	if config.MaxSize == 0 {
		config.MaxSize = 1000
	}

	p := &CachedSecretProvider{
		provider: provider,
		cache:    make(map[string]*cacheEntry),
		config:   config,
		stopCh:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	if config.Enabled {
		go p.cleanupLoop()
	}

	return p
}

// GetSecret retrieves a secret with caching.
func (p *CachedSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	if !p.config.Enabled {
		return p.provider.GetSecret(ctx, key)
	}

	// Check cache
	p.mu.Lock()
	entry, ok := p.cache[key]
	if ok && entry.expiresAt.After(time.Now()) {
		// Update last accessed time for true LRU
		entry.lastAccessed = time.Now()
		p.mu.Unlock()
		return entry.value, nil
	}
	p.mu.Unlock()

	// Fetch from provider
	value, err := p.provider.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}

	// Store in cache
	p.mu.Lock()
	defer p.mu.Unlock()

	// Evict if cache is full (simple LRU)
	if len(p.cache) >= p.config.MaxSize {
		p.evictOldest()
	}

	now := time.Now()
	p.cache[key] = &cacheEntry{
		value:        value,
		expiresAt:    now.Add(time.Duration(p.config.TTL) * time.Second),
		lastAccessed: now,
	}

	return value, nil
}

// GetSecretMap delegates to underlying provider (no caching for batch ops).
func (p *CachedSecretProvider) GetSecretMap(ctx context.Context, prefix string) (map[string]string, error) {
	return p.provider.GetSecretMap(ctx, prefix)
}

// Close closes the underlying provider and stops cleanup.
func (p *CachedSecretProvider) Close() error {
	close(p.stopCh)
	return p.provider.Close()
}

// cleanupLoop periodically removes expired entries.
func (p *CachedSecretProvider) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			now := time.Now()
			for key, entry := range p.cache {
				if entry.expiresAt.Before(now) {
					delete(p.cache, key)
				}
			}
			p.mu.Unlock()
		case <-p.stopCh:
			return
		}
	}
}

// evictOldest removes the least recently used cache entry (true LRU).
// Must be called with lock held.
func (p *CachedSecretProvider) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range p.cache {
		if oldestKey == "" || entry.lastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccessed
		}
	}

	if oldestKey != "" {
		delete(p.cache, oldestKey)
	}
}

// ClearCache clears all cached secrets (for testing).
func (p *CachedSecretProvider) ClearCache() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = make(map[string]*cacheEntry)
}

// CacheSize returns the current number of cached secrets (for testing).
func (p *CachedSecretProvider) CacheSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.cache)
}
