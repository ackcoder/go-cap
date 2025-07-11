package gocap

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var _ Storage = (*RedisStorage)(nil)

// Redis 存储实现
type RedisStorage struct {
	challengePrefix string //质询数据缓存前缀
	tokenPrefix     string //验证令牌数据缓存前缀

	client *redis.Client
}

type RedisStorageConfig struct {
	RedisAddr string `json:"redis_addr"` //required. host:port
	RedisUser string `json:"redis_user"` //optional.
	RedisPass string `json:"redis_pass"` //optional.
	RedisDb   int    `json:"redis_db"`   //optional.

	PrefixChallenge string `json:"prefix_challenge"` //质询数据缓存前缀
	PrefixToken     string `json:"prefix_token"`     //验证令牌数据缓存前缀
}

// 创建 Redis 存储实例
//   - {conf} 配置项
func NewRedisStorage(conf *RedisStorageConfig) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Username: conf.RedisUser,
		Password: conf.RedisPass,
		DB:       conf.RedisDb,
	})
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	return &RedisStorage{
		tokenPrefix:     conf.PrefixToken,
		challengePrefix: conf.PrefixChallenge,
		client:          client,
	}, nil
}

func (rs *RedisStorage) SetChallenge(ctx context.Context, token string, expiresTs int64) error {
	k := rs.challengePrefix + token
	exp := time.Until(time.Unix(expiresTs, 0))
	return rs.client.Set(ctx, k, expiresTs, exp).Err()
}

func (rs *RedisStorage) GetChallenge(ctx context.Context, token string, isGetDel ...bool) (ts int64, exists bool) {
	var err error
	k := rs.challengePrefix + token
	if len(isGetDel) > 0 && isGetDel[0] {
		ts, err = rs.client.GetDel(ctx, k).Int64()
	} else {
		ts, err = rs.client.Get(ctx, k).Int64()
	}
	if err != nil {
		return
	}
	exists = true
	return
}

func (rs *RedisStorage) SetToken(ctx context.Context, key string, expiresTs int64) error {
	k := rs.tokenPrefix + key
	exp := time.Until(time.Unix(expiresTs, 0))
	return rs.client.Set(ctx, k, expiresTs, exp).Err()
}

func (rs *RedisStorage) GetToken(ctx context.Context, key string, isGetDel ...bool) (ts int64, exists bool) {
	var err error
	k := rs.tokenPrefix + key
	if len(isGetDel) > 0 && isGetDel[0] {
		ts, err = rs.client.GetDel(ctx, k).Int64()
	} else {
		ts, err = rs.client.Get(ctx, k).Int64()
	}
	if err != nil {
		return
	}
	exists = true
	return
}

func (s *RedisStorage) Cleanup() error {
	// 注: redis 会自动清理过期键
	return nil
}
