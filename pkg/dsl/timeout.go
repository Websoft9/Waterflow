package dsl

import (
	"fmt"
	"time"
)

// TimeoutResolver 超时解析器
type TimeoutResolver struct {
	defaultJobTimeout  int // 默认 Job 超时 (分钟)
	defaultStepTimeout int // 默认 Step 超时 (分钟)
}

// NewTimeoutResolver 创建超时解析器
func NewTimeoutResolver() *TimeoutResolver {
	return &TimeoutResolver{
		defaultJobTimeout:  360, // 6 小时
		defaultStepTimeout: 360, // 6 小时
	}
}

// ResolveStepTimeout 解析 Step 超时时间
// 优先级: Step.TimeoutMinutes > Job.TimeoutMinutes > 默认值
func (r *TimeoutResolver) ResolveStepTimeout(step *Step, job *Job) time.Duration {
	timeoutMinutes := 0

	if step.TimeoutMinutes > 0 {
		// Step 显式配置超时
		timeoutMinutes = step.TimeoutMinutes
	} else if job.TimeoutMinutes > 0 {
		// 继承 Job 超时
		timeoutMinutes = job.TimeoutMinutes
	} else {
		// 使用默认值
		timeoutMinutes = r.defaultStepTimeout
	}

	return time.Duration(timeoutMinutes) * time.Minute
}

// ResolveJobTimeout 解析 Job 超时时间
func (r *TimeoutResolver) ResolveJobTimeout(job *Job) time.Duration {
	if job.TimeoutMinutes > 0 {
		return time.Duration(job.TimeoutMinutes) * time.Minute
	}
	return time.Duration(r.defaultJobTimeout) * time.Minute
}

// ValidateTimeout 验证超时配置
func (r *TimeoutResolver) ValidateTimeout(timeoutMinutes int, fieldName string) error {
	if timeoutMinutes < 0 {
		return fmt.Errorf("%s: timeout cannot be negative", fieldName)
	}
	if timeoutMinutes > 1440 {
		return fmt.Errorf("%s: timeout cannot exceed 1440 minutes (24 hours)", fieldName)
	}
	return nil
}
