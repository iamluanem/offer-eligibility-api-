package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

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

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Clear(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

type InMemoryCache struct {
	data map[string]cacheEntry
	mu   chan struct{}
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

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

func (m *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.lock()
	defer m.unlock()

	m.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (m *InMemoryCache) Delete(ctx context.Context, key string) error {
	m.lock()
	defer m.unlock()

	delete(m.data, key)
	return nil
}

func (m *InMemoryCache) Clear(ctx context.Context) error {
	m.lock()
	defer m.unlock()

	m.data = make(map[string]cacheEntry)
	return nil
}

var (
	ErrNotFound = fmt.Errorf("cache: key not found")
)

func GetJSON(ctx context.Context, cache Cache, key string, dest interface{}) error {
	data, err := cache.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func SetJSON(ctx context.Context, cache Cache, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.Set(ctx, key, data, ttl)
}
