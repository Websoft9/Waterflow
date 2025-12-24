package dsl

import (
	"fmt"
	"time"

	temporal "go.temporal.io/sdk/temporal"
)

// RetryPolicyResolver 重试策略解析器
type RetryPolicyResolver struct{}

// NewRetryPolicyResolver 创建重试策略解析器
func NewRetryPolicyResolver() *RetryPolicyResolver {
	return &RetryPolicyResolver{}
}

// Resolve 解析重试策略
func (r *RetryPolicyResolver) Resolve(strategy *RetryStrategy) (*ResolvedRetryPolicy, error) {
	if strategy == nil {
		return r.DefaultRetryPolicy(), nil
	}

	policy := &ResolvedRetryPolicy{
		MaxAttempts:        r.resolveMaxAttempts(strategy.MaxAttempts),
		BackoffCoefficient: r.resolveBackoffCoefficient(strategy.BackoffCoefficient),
	}

	// 解析初始间隔
	if strategy.InitialInterval != "" {
		duration, err := parseDuration(strategy.InitialInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid initial-interval: %w", err)
		}
		policy.InitialInterval = duration
	} else {
		policy.InitialInterval = 1 * time.Second
	}

	// 解析最大间隔
	if strategy.MaxInterval != "" {
		duration, err := parseDuration(strategy.MaxInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid max-interval: %w", err)
		}
		policy.MaxInterval = duration
	} else {
		policy.MaxInterval = 60 * time.Second
	}

	return policy, nil
}

// DefaultRetryPolicy 默认重试策略
func (r *RetryPolicyResolver) DefaultRetryPolicy() *ResolvedRetryPolicy {
	return &ResolvedRetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaxInterval:        60 * time.Second,
	}
}

// resolveMaxAttempts 解析最大尝试次数
func (r *RetryPolicyResolver) resolveMaxAttempts(maxAttempts int) int {
	if maxAttempts > 0 {
		return maxAttempts
	}
	return 3 // 默认 3 次
}

// resolveBackoffCoefficient 解析退避系数
func (r *RetryPolicyResolver) resolveBackoffCoefficient(coefficient float64) float64 {
	if coefficient > 0 {
		return coefficient
	}
	return 2.0 // 默认 2.0
}

// parseDuration 解析时间间隔字符串
// 支持格式: 1s, 30s, 5m, 1h
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("duration string is empty")
	}

	// 使用 Go 标准库解析
	duration, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}

	return duration, nil
}

// ValidateDuration 验证duration字符串格式 (独立函数,避免Resolve重复调用)
func ValidateDuration(s string) error {
	if s == "" {
		return nil // 空字符串表示使用默认值
	}
	_, err := parseDuration(s)
	return err
}

// ResolvedRetryPolicy 解析后的重试策略
type ResolvedRetryPolicy struct {
	MaxAttempts        int
	InitialInterval    time.Duration
	BackoffCoefficient float64
	MaxInterval        time.Duration
}

// CalculateNextRetryInterval 计算下次重试间隔
func (p *ResolvedRetryPolicy) CalculateNextRetryInterval(attemptNumber int) time.Duration {
	if attemptNumber <= 0 {
		return p.InitialInterval
	}

	// 指数退避: initialInterval * (backoffCoefficient ^ attemptNumber)
	interval := float64(p.InitialInterval)
	for i := 0; i < attemptNumber; i++ {
		interval *= p.BackoffCoefficient
	}

	result := time.Duration(interval)

	// 限制最大间隔
	if result > p.MaxInterval {
		return p.MaxInterval
	}

	return result
}

// ToTemporalRetryPolicy converts to Temporal SDK RetryPolicy
func (p *ResolvedRetryPolicy) ToTemporalRetryPolicy() *temporal.RetryPolicy {
	maxAttempts := p.MaxAttempts
	if maxAttempts > 2147483647 {
		maxAttempts = 2147483647 // int32 max value
	}
	return &temporal.RetryPolicy{
		InitialInterval:    p.InitialInterval,
		BackoffCoefficient: p.BackoffCoefficient,
		MaximumInterval:    p.MaxInterval,
		MaximumAttempts:    int32(maxAttempts), //nolint:gosec // checked above
	}
}
