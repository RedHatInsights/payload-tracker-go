package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	_cache "github.com/go-redis/cache/v9"
	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
	ctx    = context.Background()
	cache  *_cache.Cache
)

func Init() {
	cfg := config.Get()

	// no-op so we can always init this in tests
	fmt.Println(cfg.ConsumerConfig.ConsumerPayloadFieldsRepoImpl)
	if cfg.ConsumerConfig.ConsumerPayloadFieldsRepoImpl != "redis" {
		return
	}

	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		Password: cfg.RedisConfig.Password,
	})
	cache = _cache.New(&_cache.Options{
		Redis: client,
	})
}

func Set(key string, val any, ttl time.Duration) error {
	if err := cache.Set(&_cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: val,
		TTL:   1 * time.Second,
	}); err != nil {
		log.Fatal(err)
	}
	return nil
}

func Get(key string, val any) error {
	return cache.Get(ctx, key, &val)
}
