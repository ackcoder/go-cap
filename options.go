package gocap

import "time"

type CapOption func(c *Cap)

// 配置自定义存储
func WithStorage(storage Storage) CapOption {
	return func(c *Cap) {
		c.storage = storage
	}
}

// 配置质询数量/大小/难度
//   - {count} 生成组数量. 默认50
//   - {size} 每组大小. 默认32
//   - {difficulty} 难度级别. 默认4
func WithChallenge(count, size, difficulty int) CapOption {
	return func(c *Cap) {
		if count > 0 {
			c.challengeCount = count
		}
		if size > 0 {
			c.challengeSize = size
		}
		if difficulty > 0 {
			c.challengeDifficulty = difficulty
		}
	}
}

// 配置质询(令牌)过期时间
//   - {expires} 过期时间/秒. 默认600秒(10分钟)
func WithChallengeExpires(expires int) CapOption {
	return func(c *Cap) {
		if expires > 0 {
			c.challengeExpires = time.Duration(expires) * time.Second
		}
	}
}

// 配置验证令牌过期时间
//   - {expires} 过期时间/秒. 默认1200秒(20分钟)
func WithTokenExpires(expires int) CapOption {
	return func(c *Cap) {
		if expires > 0 {
			c.tokenExpires = time.Duration(expires) * time.Second
		}
	}
}

// 配置验证令牌检查参数
//   - {isOnce} 是否一次性验证. 默认true
func WithTokenVerifyOnce(isOnce bool) CapOption {
	return func(c *Cap) {
		c.tokenVerifyOnce = isOnce
	}
}
