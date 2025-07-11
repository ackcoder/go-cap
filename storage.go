package gocap

import (
	"context"
	"sync"
	"time"
)

type Storage interface {
	// SetChallenge 设置质询令牌
	//   - {token} 质询令牌
	//   - {expiresTs} 过期时刻, 秒级时间戳
	SetChallenge(ctx context.Context, token string, expiresTs int64) error
	// GetChallenge 获取质询令牌过期时间
	//   - {token} 质询令牌
	//   - {isGetDel} 是否获取后删除. 可选
	GetChallenge(ctx context.Context, token string, isGetDel ...bool) (ts int64, exists bool)

	// SetToken 设置验证令牌
	//   - {key} 验证令牌Key
	//   - {expiresTs} 过期时刻, 秒级时间戳
	SetToken(ctx context.Context, key string, expiresTs int64) error
	// GetToken 获取验证令牌过期时间
	//   - {key} 验证令牌Key
	//   - {isGetDel} 是否获取后删除. 可选
	GetToken(ctx context.Context, key string, isGetDel ...bool) (ts int64, exists bool)

	// Cleanup 清理过期数据
	Cleanup() error
}

var _ Storage = (*MemoryStorage)(nil)

const defaultCleanupInterval = 5 * time.Minute

// 内存存储实现
//
// 创建实例 gocap.New() 时的默认存储
// 默认每隔 5分钟 清理一次过期数据
type MemoryStorage struct {
	challengesMap sync.Map //质询查找表; key:token(质询令牌), value:timestamp(过期时间戳)
	tokensMap     sync.Map //令牌查找表; key:`id:hash`(验证令牌key), value:timestamp(过期时间戳)
}

// 创建内存存储实例
//   - {cleanup} 自动清理过期数据的定时间隔, 单位秒, 默认5分钟
func NewMemoryStorage(cleanup ...int64) *MemoryStorage {
	var interval = defaultCleanupInterval
	if len(cleanup) > 0 && cleanup[0] > 0 {
		interval = time.Duration(cleanup[0]) * time.Second
	}

	s := &MemoryStorage{}
	// 启动定期清理
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.Cleanup()
		}
	}()

	return s
}

func (ms *MemoryStorage) SetChallenge(ctx context.Context, token string, expiresTs int64) error {
	ms.challengesMap.Store(token, expiresTs)
	return nil
}

func (ms *MemoryStorage) GetChallenge(ctx context.Context, token string, isGetDel ...bool) (ts int64, exists bool) {
	var tsVar interface{}
	if len(isGetDel) > 0 && isGetDel[0] {
		tsVar, exists = ms.challengesMap.LoadAndDelete(token)
	} else {
		tsVar, exists = ms.challengesMap.Load(token)
	}
	if exists {
		ts = tsVar.(int64)
	}
	return
}

func (ms *MemoryStorage) SetToken(ctx context.Context, key string, expiresTs int64) error {
	ms.tokensMap.Store(key, expiresTs)
	return nil
}

func (ms *MemoryStorage) GetToken(ctx context.Context, key string, isGetDel ...bool) (ts int64, exists bool) {
	var tsVar interface{}
	if len(isGetDel) > 0 && isGetDel[0] {
		tsVar, exists = ms.tokensMap.LoadAndDelete(key)
	} else {
		tsVar, exists = ms.tokensMap.Load(key)
	}
	if exists {
		ts = tsVar.(int64)
	}
	return
}

func (ms *MemoryStorage) Cleanup() error {
	now := time.Now().Unix()

	var expiredChallenges []interface{}
	ms.challengesMap.Range(func(key, value any) bool {
		if value.(int64) < now {
			expiredChallenges = append(expiredChallenges, key)
		}
		return true
	})
	for _, tk := range expiredChallenges {
		ms.challengesMap.Delete(tk)
	}

	var expiredTokens []interface{}
	ms.tokensMap.Range(func(key, value any) bool {
		if value.(int64) < now {
			expiredTokens = append(expiredTokens, key)
		}
		return true
	})
	for _, key := range expiredTokens {
		ms.tokensMap.Delete(key)
	}

	return nil
}
