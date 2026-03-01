// Package cache provides a Dragonfly/Redis client wrapper.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis/Dragonfly client.
type Cache struct {
	Client *redis.Client
}

// ParseURL validates a Redis connection URL.
func ParseURL(url string) (*redis.Options, error) {
	if url == "" {
		return nil, fmt.Errorf("cache URL is empty")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid cache URL: %w", err)
	}
	return opts, nil
}

// New creates a new cache client.
func New(ctx context.Context, url string) (*Cache, error) {
	opts, err := ParseURL(url)
	if err != nil {
		return nil, err
	}

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("pinging cache: %w", err)
	}

	return &Cache{Client: client}, nil
}

// Close shuts down the cache client.
func (c *Cache) Close() error {
	return c.Client.Close()
}

// HealthCheck verifies the cache connection is alive.
func (c *Cache) HealthCheck(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}
