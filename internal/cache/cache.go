package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache defines the interface for caching operations.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// RedisCache implements Cache interface using Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance.
func NewRedisCache(addr string, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

// Get retrieves a value from cache.
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

// Set stores a value in cache with TTL.
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from cache.
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Clear clears all keys from cache (use with caution).
func (r *RedisCache) Clear(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// Close closes the Redis connection.
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// InMemoryCache is a simple in-memory cache implementation for testing/development.
type InMemoryCache struct {
	data map[string]cacheEntry
	mu   chan struct{} // Simple mutex using channel
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewInMemoryCache creates a new in-memory cache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: make(map[string]cacheEntry),
		mu:   make(chan struct{}, 1),
	}
}

func (m *InMemoryCache) lock() {
	m.mu <- struct{}{}
}

func (m *InMemoryCache) unlock() {
	<-m.mu
}

// Get retrieves a value from cache.
func (m *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.lock()
	defer m.unlock()

	entry, exists := m.data[key]
	if !exists {
		return nil, ErrNotFound
	}

	if time.Now().After(entry.expiresAt) {
		delete(m.data, key)
		return nil, ErrNotFound
	}

	return entry.value, nil
}

// Set stores a value in cache with TTL.
func (m *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.lock()
	defer m.unlock()

	m.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a key from cache.
func (m *InMemoryCache) Delete(ctx context.Context, key string) error {
	m.lock()
	defer m.unlock()

	delete(m.data, key)
	return nil
}

// Clear clears all keys from cache.
func (m *InMemoryCache) Clear(ctx context.Context) error {
	m.lock()
	defer m.unlock()

	m.data = make(map[string]cacheEntry)
	return nil
}

// Errors
var (
	ErrNotFound = fmt.Errorf("cache: key not found")
)

// GetJSON retrieves and unmarshals a JSON value from cache.
func GetJSON(ctx context.Context, cache Cache, key string, dest interface{}) error {
	data, err := cache.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// SetJSON marshals and stores a JSON value in cache.
func SetJSON(ctx context.Context, cache Cache, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.Set(ctx, key, data, ttl)
}

