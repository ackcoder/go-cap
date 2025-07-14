package gocap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultChallengeTokenSize  = 25
	defaultChallengeCount      = 50
	defaultChallengeSize       = 32
	defaultChallengeDifficulty = 4
	defaultChallengeExpires    = 10 * time.Minute

	defaultTokenSize       = 15
	defaultTokenIdSize     = 8
	defaultTokenExpires    = 20 * time.Minute
	defaultTokenVerifyOnce = true

	defaultHttpHandleLimitRPS   = 10
	defaultHttpHandleLimitBurst = 50
)

var (
	ErrInvalidChallenge  = errors.New("invalid challenge body")        //非法质询参数
	ErrChallengeExpired  = errors.New("challenge expired")             //质询令牌过期
	ErrInvalidSolutions  = errors.New("invalid solutions")             //非法解方案组参数
	ErrGenerateFailed    = errors.New("generate random string failed") //生成随机串失败
	ErrStorageNotDefined = errors.New("storage not defined")           //未定义存储实例
)

type (
	Cap struct {
		challengeTokenSize  int           //质询令牌长度. 默认25
		challengeCount      int           //质询生成数量. 默认50
		challengeSize       int           //质询项字节大小. 默认32
		challengeDifficulty int           //质询难度级别. 默认4
		challengeExpires    time.Duration //质询(令牌)过期时间. 默认600秒(10分钟)

		tokenSize       int           //验证令牌(主要哈希)长度. 默认15
		tokenIdSize     int           //验证令牌ID长度. 默认8
		tokenExpires    time.Duration //验证令牌过期时间. 默认20分钟
		tokenVerifyOnce bool          //验证令牌一次性检查. 默认true

		limiterRPS   int //限流器每秒通过数. 默认10次/秒
		limiterBurst int //限流器最大突发容量数. 默认50次突发

		limiter [3]*rate.Limiter //令牌桶限流器组, 在使用 Handlexxx 方法时生效
		storage Storage
	}

	// 质询数据内容
	ChallengeData struct {
		Challenge ChallengeItem `json:"challenge"`
		Expires   int64         `json:"expires"` //过期时间,秒级时间戳
		Token     string        `json:"token"`   //质询令牌
	}
	ChallengeItem struct {
		C int `json:"c"` //质询数量
		S int `json:"s"` //质询大小
		D int `json:"d"` //质询难度
	}
	// 验证令牌内容
	TokenData struct {
		Expires int64  `json:"expires,omitzero"` //过期时间,秒级时间戳
		Token   string `json:"token,omitzero"`   //验证令牌
	}

	// 前端组件传入的参数
	//
	// 注: 用于 RedeemChallenge() 或 ValidateToken()
	VerificationParams struct {
		Token     string  `json:"token"`              //质询令牌
		Solutions []int64 `json:"solutions,omitzero"` //质询解方案组
	}
	// 返回前端组件的结果
	//
	// 注: 来自 RedeemChallenge() 或 ValidateToken()
	VerificationResult struct {
		*TokenData
		Success bool   `json:"success"`
		Message string `json:"message,omitzero"`
	}
)

func New(opts ...CapOption) *Cap {
	c := &Cap{
		challengeTokenSize:  defaultChallengeTokenSize,
		challengeCount:      defaultChallengeCount,
		challengeSize:       defaultChallengeSize,
		challengeDifficulty: defaultChallengeDifficulty,
		challengeExpires:    defaultChallengeExpires,

		tokenSize:       defaultTokenSize,
		tokenIdSize:     defaultTokenIdSize,
		tokenExpires:    defaultTokenExpires,
		tokenVerifyOnce: defaultTokenVerifyOnce,

		limiterRPS:   defaultHttpHandleLimitRPS,
		limiterBurst: defaultHttpHandleLimitBurst,

		storage: NewMemoryStorage(),
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.limiterRPS+c.limiterBurst > 0 {
		c.limiter = [3]*rate.Limiter{
			rate.NewLimiter(rate.Every(time.Second/time.Duration(c.limiterRPS)), c.limiterBurst),
			rate.NewLimiter(rate.Every(time.Second/time.Duration(c.limiterRPS)), c.limiterBurst),
			rate.NewLimiter(rate.Every(time.Second/time.Duration(c.limiterRPS)), c.limiterBurst),
		}
	}
	return c
}

// CreateChallenge 创建质询数据
func (c *Cap) CreateChallenge(ctx context.Context) (*ChallengeData, error) {
	if c.storage == nil {
		return nil, ErrStorageNotDefined
	}

	data := &ChallengeData{
		Challenge: ChallengeItem{
			C: c.challengeCount,
			S: c.challengeSize,
			D: c.challengeDifficulty,
		},
		Expires: time.Now().Add(c.challengeExpires).Unix(),
	}

	token, err := generateRandomHex(c.challengeTokenSize)
	if err != nil {
		return nil, err
	}
	data.Token = token

	c.storage.SetChallenge(ctx, token, data.Expires)

	return data, nil
}

// RedeemChallenge 用工作量证明兑换验证令牌
//   - {token} 质询令牌, 即 ChallengeData.Token
//   - {solutions} 工作量证明/解方案组
func (c *Cap) RedeemChallenge(ctx context.Context, token string, solutions []int64) (*TokenData, error) {
	if c.storage == nil {
		return nil, ErrStorageNotDefined
	}
	if token == "" || len(token) != c.challengeTokenSize {
		return nil, ErrInvalidChallenge
	}
	if len(solutions) < c.challengeCount {
		return nil, ErrInvalidSolutions
	}

	expTs, exists := c.storage.GetChallenge(ctx, token, true)
	if !exists || expTs < time.Now().Unix() {
		return nil, ErrChallengeExpired
	}

	b := []byte(token)
	bl := len(b)
	for i := range c.challengeCount {
		b = b[:bl]
		b = strconv.AppendInt(b, int64(i+1), 10)
		b = append(b, 'd')

		// 生成校验参数
		target := prng(string(b), c.challengeDifficulty)
		salt := prng(string(b[:len(b)-1]), c.challengeSize)

		// 计算哈希
		hash := calculateHashHex(fmt.Sprintf("%s%d", salt, solutions[i]))

		if !strings.HasPrefix(hash, target) {
			return nil, ErrInvalidSolutions
		}
	}

	vertoken, err := generateRandomHex(c.tokenSize)
	if err != nil {
		return nil, errors.Join(ErrGenerateFailed, err)
	}
	id, err := generateRandomHex(c.tokenIdSize)
	if err != nil {
		return nil, errors.Join(ErrGenerateFailed, err)
	}
	hash := calculateHashHex(vertoken)
	expires := time.Now().Add(c.tokenExpires).Unix()

	// 保存验证令牌Key(id:hash)
	c.storage.SetToken(ctx, fmt.Sprintf("%s:%s", id, hash), expires)

	// 返回验证令牌(id:vertoken)
	return &TokenData{
		Expires: expires,
		Token:   fmt.Sprintf("%s:%s", id, vertoken),
	}, nil
}

// ValidateToken 检查验证令牌
//   - {token} 验证令牌, 即 TokenData.Token
func (c *Cap) ValidateToken(ctx context.Context, token string) bool {
	if c.storage == nil {
		return false //未定义存储实例
	}
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false //验证令牌入参错误
	}

	id, vertoken := parts[0], parts[1]
	hash := calculateHashHex(vertoken)
	key := fmt.Sprintf("%s:%s", id, hash)

	expTx, exists := c.storage.GetToken(ctx, key, c.tokenVerifyOnce)
	if !exists || expTx < time.Now().Unix() {
		return false //验证令牌已过期
	}

	return true
}

// 生成确定性的16进制字串
func prng(seed string, length int) string {
	h := fnv.New32a() // FNV-1a 哈希算法 32 位版本
	h.Write([]byte(seed))
	state := h.Sum32()

	var result strings.Builder
	result.Grow(length)
	for result.Len() < length {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		result.WriteString(fmt.Sprintf("%08x", state))
	}
	return result.String()[:length]
}

// 生成指定长度的16进制随机字串
func generateRandomHex(length int) (string, error) {
	b := make([]byte, (length+1)/2)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:length], nil
}

// 计算指定内容的16进制哈希字串
func calculateHashHex(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}
