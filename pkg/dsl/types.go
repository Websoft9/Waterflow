package dsl

import "fmt"

// Workflow 工作流定义
type Workflow struct {
	Name string                 `yaml:"name" json:"name"`
	On   interface{}            `yaml:"on" json:"on"` // string 或 TriggerConfig
	Vars map[string]interface{} `yaml:"vars,omitempty" json:"vars,omitempty"`
	Env  map[string]string      `yaml:"env,omitempty" json:"env,omitempty"`
	Jobs map[string]*Job        `yaml:"jobs" json:"jobs"`

	// 元数据 (内部使用)
	SourceFile string         `yaml:"-" json:"-"`
	LineMap    map[string]int `yaml:"-" json:"-"` // 字段 → 行号映射
}

// Job 任务定义
type Job struct {
	RunsOn          string            `yaml:"runs-on" json:"runs_on"`
	TimeoutMinutes  int               `yaml:"timeout-minutes,omitempty" json:"timeout_minutes,omitempty"`
	Needs           []string          `yaml:"needs,omitempty" json:"needs,omitempty"`
	If              string            `yaml:"if,omitempty" json:"if,omitempty"`             // Story 1.5: Job级if条件
	Strategy        *Strategy         `yaml:"strategy,omitempty" json:"strategy,omitempty"` // Story 1.6: Matrix策略
	Env             map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Steps           []*Step           `yaml:"steps" json:"steps"`
	ContinueOnError bool              `yaml:"continue-on-error,omitempty" json:"continue_on_error,omitempty"`
	Outputs         map[string]string `yaml:"outputs,omitempty" json:"outputs,omitempty"` // Story 1.5: Job输出

	// 内部字段
	Name    string `yaml:"-" json:"name"` // Job key
	LineNum int    `yaml:"-" json:"-"`
}

// Strategy Matrix 策略
type Strategy struct {
	Matrix      map[string][]interface{} `yaml:"matrix" json:"matrix"`
	MaxParallel int                      `yaml:"max-parallel,omitempty" json:"max_parallel,omitempty"`
	FailFast    *bool                    `yaml:"fail-fast,omitempty" json:"fail_fast,omitempty"` // 默认 true

	// 预留字段 (MVP 不实现)
	Include []map[string]interface{} `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []map[string]interface{} `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// Step 步骤定义
type Step struct {
	ID              string                 `yaml:"id,omitempty" json:"id,omitempty"` // Story 1.5: Step ID用于输出引用
	Name            string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Uses            string                 `yaml:"uses" json:"uses"` // node@version
	With            map[string]interface{} `yaml:"with,omitempty" json:"with,omitempty"`
	TimeoutMinutes  int                    `yaml:"timeout-minutes,omitempty" json:"timeout_minutes,omitempty"` // Story 1.7: Step超时
	ContinueOnError bool                   `yaml:"continue-on-error,omitempty" json:"continue_on_error,omitempty"`
	If              string                 `yaml:"if,omitempty" json:"if,omitempty"`                         // Story 1.5
	RetryStrategy   *RetryStrategy         `yaml:"retry-strategy,omitempty" json:"retry_strategy,omitempty"` // Story 1.7: 重试策略
	Env             map[string]string      `yaml:"env,omitempty" json:"env,omitempty"`

	// 内部字段
	Index   int `yaml:"-" json:"index"`
	LineNum int `yaml:"-" json:"-"`
}

// RetryStrategy 重试策略 (Story 1.7)
type RetryStrategy struct {
	MaxAttempts        int     `yaml:"max-attempts,omitempty" json:"max_attempts,omitempty"`               // 最大尝试次数 (默认 3)
	InitialInterval    string  `yaml:"initial-interval,omitempty" json:"initial_interval,omitempty"`       // 首次重试间隔 (默认 1s)
	BackoffCoefficient float64 `yaml:"backoff-coefficient,omitempty" json:"backoff_coefficient,omitempty"` // 退避系数 (默认 2.0)
	MaxInterval        string  `yaml:"max-interval,omitempty" json:"max_interval,omitempty"`               // 最大间隔 (默认 60s)
}

// TriggerConfig 触发器配置 (简化版)
type TriggerConfig struct {
	Push     *PushTrigger     `yaml:"push,omitempty" json:"push,omitempty"`
	Schedule *ScheduleTrigger `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Webhook  *WebhookTrigger  `yaml:"webhook,omitempty" json:"webhook,omitempty"`
}

// PushTrigger Push 触发器
type PushTrigger struct {
	Branches []string `yaml:"branches,omitempty" json:"branches,omitempty"`
}

// ScheduleTrigger 定时触发器
type ScheduleTrigger struct {
	Cron string `yaml:"cron" json:"cron"`
}

// WebhookTrigger Webhook 触发器
type WebhookTrigger struct {
	Events []string `yaml:"events" json:"events"`
}

// MatrixInstance Matrix 实例
type MatrixInstance struct {
	Index  int                    // 实例索引 (0-based)
	Matrix map[string]interface{} // Matrix 变量
}

// MatrixError Matrix 错误
type MatrixError struct {
	Type         string
	Combinations int
	Limit        int
	Suggestion   string
}

func (e *MatrixError) Error() string {
	return fmt.Sprintf("matrix combinations %d exceed limit %d", e.Combinations, e.Limit)
}
