package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sangfor.local/hci/hci-common/utils/redis"
	"time"

	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
	"github.com/redis/go-redis/v9"
)

// CacheRepository Redis 缓存仓储接口
type CacheRepository interface {
	// Set 设置缓存
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	// Get 获取缓存
	Get(ctx context.Context, key string, dest interface{}) error
	// Delete 删除缓存
	Delete(ctx context.Context, keys ...string) error
	// Exists 检查键是否存在
	Exists(ctx context.Context, keys ...string) (int64, error)
	// Expire 设置过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// TTL 获取键的剩余生存时间
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Keys 查找匹配的键
	Keys(ctx context.Context, pattern string) ([]string, error)
	// FlushDB 清空当前数据库
	FlushDB(ctx context.Context) error
	// Ping 健康检查
	Ping(ctx context.Context) error
}

// cacheRepo Redis 缓存仓储实现
type cacheRepo struct {
	client      *redis.Client
	keyPrefix   string
	defaultTTL  time.Duration
	maxRetries  int
	retryDelay  time.Duration
}

// CacheConfig Redis 缓存配置
type CacheConfig struct {
	Addr         string        // Redis 地址
	Password     string        // 密码
	DB           int           // 数据库编号
	PoolSize     int           // 连接池大小
	MinIdleConns int           // 最小空闲连接数
	MaxRetries   int           // 最大重试次数
	DialTimeout  time.Duration // 连接超时
	ReadTimeout  time.Duration // 读超时
	WriteTimeout time.Duration // 写超时
	KeyPrefix    string        // 键前缀
	DefaultTTL   time.Duration // 默认过期时间
}

// NewCacheRepository 创建 Redis 缓存仓储
func NewCacheRepository(config *CacheConfig) (CacheRepository, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "config cannot be nil")
	}

	// 设置默认值
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 2
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 1 * time.Hour
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to connect to redis")
	}

	return &cacheRepo{
		client:      client,
		keyPrefix:   config.KeyPrefix,
		defaultTTL:  config.DefaultTTL,
		maxRetries:  config.MaxRetries,
		retryDelay:  100 * time.Millisecond,
	}, nil
}

// buildKey 构建完整的缓存键
func (r *cacheRepo) buildKey(key string) string {
	if r.keyPrefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", r.keyPrefix, key)
}

// Set 设置缓存
func (r *cacheRepo) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if key == "" {
		return errors.New(errors.CodeInvalidParameter, "key cannot be empty")
	}

	fullKey := r.buildKey(key)

	// 序列化值
	data, err := r.serialize(value)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to serialize value")
	}

	// 设置默认过期时间
	if expiration == 0 {
		expiration = r.defaultTTL
	}

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		if err := r.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}
		return nil
	}

	return errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to set cache after retries")
}

// Get 获取缓存
func (r *cacheRepo) Get(ctx context.Context, key string, dest interface{}) error {
	if key == "" {
		return errors.New(errors.CodeInvalidParameter, "key cannot be empty")
	}
	if dest == nil {
		return errors.New(errors.CodeInvalidParameter, "dest cannot be nil")
	}

	fullKey := r.buildKey(key)

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		data, err := r.client.Get(ctx, fullKey).Bytes()
		if err != nil {
			if err == redis.Nil {
				return errors.New(errors.CodeNotFound, "cache not found")
			}
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}

		// 反序列化
		if err := r.deserialize(data, dest); err != nil {
			return errors.Wrap(err, errors.CodeInternalError, "failed to deserialize value")
		}
		return nil
	}

	return errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to get cache after retries")
}

// Delete 删除缓存
func (r *cacheRepo) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return errors.New(errors.CodeInvalidParameter, "keys cannot be empty")
	}

	// 构建完整键
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		if err := r.client.Del(ctx, fullKeys...).Err(); err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}
		return nil
	}

	return errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to delete cache after retries")
}

// Exists 检查键是否存在
func (r *cacheRepo) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, errors.New(errors.CodeInvalidParameter, "keys cannot be empty")
	}

	// 构建完整键
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		count, err := r.client.Exists(ctx, fullKeys...).Result()
		if err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}
		return count, nil
	}

	return 0, errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to check existence after retries")
}

// Expire 设置过期时间
func (r *cacheRepo) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if key == "" {
		return errors.New(errors.CodeInvalidParameter, "key cannot be empty")
	}

	fullKey := r.buildKey(key)

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		if err := r.client.Expire(ctx, fullKey, expiration).Err(); err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}
		return nil
	}

	return errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to set expiration after retries")
}

// TTL 获取键的剩余生存时间
func (r *cacheRepo) TTL(ctx context.Context, key string) (time.Duration, error) {
	if key == "" {
		return 0, errors.New(errors.CodeInvalidParameter, "key cannot be empty")
	}

	fullKey := r.buildKey(key)

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		ttl, err := r.client.TTL(ctx, fullKey).Result()
		if err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}
		return ttl, nil
	}

	return 0, errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to get TTL after retries")
}

// Keys 查找匹配的键
func (r *cacheRepo) Keys(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}

	fullPattern := r.buildKey(pattern)

	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		keys, err := r.client.Keys(ctx, fullPattern).Result()
		if err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}

		// 移除键前缀
		if r.keyPrefix != "" {
			prefixLen := len(r.keyPrefix) + 1 // +1 for ":"
			for j := range keys {
				if len(keys[j]) > prefixLen {
					keys[j] = keys[j][prefixLen:]
				}
			}
		}

		return keys, nil
	}

	return nil, errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to get keys after retries")
}

// FlushDB 清空当前数据库
func (r *cacheRepo) FlushDB(ctx context.Context) error {
	// 重试机制
	var lastErr error
	for i := 0; i < r.maxRetries; i++ {
		if err := r.client.FlushDB(ctx).Err(); err != nil {
			lastErr = err
			time.Sleep(r.retryDelay * time.Duration(i+1))
			continue
		}
		return nil
	}

	return errors.Wrap(lastErr, errors.CodeDatabaseError, "failed to flush DB after retries")
}

// Ping 健康检查
func (r *cacheRepo) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// serialize 序列化值
func (r *cacheRepo) serialize(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return json.Marshal(v)
	default:
		// 对于复杂类型，使用 JSON 序列化
		return json.Marshal(v)
	}
}

// deserialize 反序列化值
func (r *cacheRepo) deserialize(data []byte, dest interface{}) error {
	switch d := dest.(type) {
	case *string:
		*d = string(data)
		return nil
	case *[]byte:
		*d = data
		return nil
	default:
		// 对于复杂类型，使用 JSON 反序列化
		return json.Unmarshal(data, dest)
	}
}

// Close 关闭 Redis 连接
func (r *cacheRepo) Close() error {
	return r.client.Close()
}

// CacheStats 缓存统计信息
type CacheStats struct {
	UsedMemory      uint64        // 已使用内存（字节）
	MaxMemory       uint64        // 最大内存（字节）
	KeysCount       int64         // 键总数
	HitRate         float64       // 命中率
	AvgTTL          time.Duration // 平均 TTL
	ConnectionCount int           // 连接数
}

// GetStats 获取缓存统计信息
func (r *cacheRepo) GetStats(ctx context.Context) (*CacheStats, error) {
	info, err := r.client.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get redis info")
	}

	dbSize, err := r.client.DBSize(ctx).Result()
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get DB size")
	}

	// 解析 info 字符串（简化版，实际应更完善）
	stats := &CacheStats{
		KeysCount: dbSize,
	}

	// 这里应该解析 info 字符串获取详细统计信息
	// 为简化示例，仅返回基本信息

	return stats, nil
}

//Personal.AI order the ending
