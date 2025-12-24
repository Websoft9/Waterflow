package dsl

import (
	"testing"
	"time"
)

// BenchmarkTimeoutResolution 测试超时解析性能
func BenchmarkTimeoutResolution(b *testing.B) {
	resolver := NewTimeoutResolver()
	step := &Step{TimeoutMinutes: 30}
	job := &Job{TimeoutMinutes: 60}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resolver.ResolveStepTimeout(step, job)
	}
}

// BenchmarkRetryPolicyResolution 测试重试策略解析性能
func BenchmarkRetryPolicyResolution(b *testing.B) {
	resolver := NewRetryPolicyResolver()
	strategy := &RetryStrategy{
		MaxAttempts:        5,
		InitialInterval:    "2s",
		BackoffCoefficient: 1.5,
		MaxInterval:        "30s",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(strategy)
	}
}

// BenchmarkErrorClassification 测试错误分类性能
func BenchmarkErrorClassification(b *testing.B) {
	classifier := NewErrorClassifier()
	testErrors := []error{
		ErrValidation("test error"),
		ErrNotFound("resource not found"),
		ErrPermissionDenied("access denied"),
		ErrNetworkTimeout("connection timeout"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, err := range testErrors {
			_ = classifier.ClassifyError(err)
			_ = classifier.IsRetryable(classifier.ClassifyError(err))
		}
	}
}

// BenchmarkRetryIntervalCalculation 测试重试间隔计算性能
func BenchmarkRetryIntervalCalculation(b *testing.B) {
	policy := &ResolvedRetryPolicy{
		MaxAttempts:        10,
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaxInterval:        60 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for attempt := 0; attempt < 10; attempt++ {
			_ = policy.CalculateNextRetryInterval(attempt)
		}
	}
}

// BenchmarkDurationValidation 测试duration验证性能
func BenchmarkDurationValidation(b *testing.B) {
	testDurations := []string{"1s", "30s", "5m", "1h", "invalid"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range testDurations {
			_ = ValidateDuration(d)
		}
	}
}
