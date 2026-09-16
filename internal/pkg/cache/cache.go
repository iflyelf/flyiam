package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config Redis 缓存配置
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Password string
	DB       int
	TTL      int // seconds
}

// Cache Redis 缓存客户端
type Cache struct {
	client  *redis.Client
	enabled bool
	ttl     time.Duration
}

// New 创建缓存客户端
func New(cfg Config) *Cache {
	if !cfg.Enabled {
		log.Println("⚠️  Redis 缓存已禁用")
		return &Cache{
			enabled: false,
			ttl:     time.Duration(cfg.TTL) * time.Second,
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis 连接失败，自动降级: %v", err)
		return &Cache{
			enabled: false,
			ttl:     time.Duration(cfg.TTL) * time.Second,
		}
	}

	log.Printf("✅ Redis 缓存连接成功: %s:%d", cfg.Host, cfg.Port)
	return &Cache{
		client:  client,
		enabled: true,
		ttl:     time.Duration(cfg.TTL) * time.Second,
	}
}

// Get 获取缓存
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if !c.enabled {
		return "", redis.Nil
	}
	return c.client.Get(ctx, key).Result()
}

// Set 设置缓存
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	if !c.enabled {
		return nil
	}
	return c.client.Set(ctx, key, value, c.ttl).Err()
}

// SetWithTTL 设置缓存（自定义过期时间）
func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.enabled {
		return nil
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Del 删除缓存
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if !c.enabled {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// DelPrefix 删除前缀匹配的所有键
func (c *Cache) DelPrefix(ctx context.Context, prefix string) error {
	if !c.enabled {
		return nil
	}

	// 使用 SCAN 遍历所有匹配的键
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// Exists 检查键是否存在
func (c *Cache) Exists(ctx context.Context, keys ...string) (int64, error) {
	if !c.enabled {
		return 0, nil
	}
	return c.client.Exists(ctx, keys...).Result()
}

// Expire 设置键过期时间
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if !c.enabled {
		return nil
	}
	return c.client.Expire(ctx, key, ttl).Err()
}

// Close 关闭连接
func (c *Cache) Close() error {
	if c.enabled && c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsEnabled 返回缓存是否启用
func (c *Cache) IsEnabled() bool {
	return c.enabled
}
