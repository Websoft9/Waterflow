package dsl

// VarsMerger 三层参数合并器
// 根据 ADR-0009，vars 支持三层覆盖机制：
// 1. YAML 默认值 (优先级最低)
// 2. 触发器绑定参数 (Schedule/Webhook 创建时指定)
// 3. 执行时参数 (API 调用或 Webhook payload 传入，优先级最高)
type VarsMerger struct{}

// NewVarsMerger 创建参数合并器
func NewVarsMerger() *VarsMerger {
	return &VarsMerger{}
}

// Merge 合并三层参数
// 优先级: executionVars > triggerVars > yamlVars
//
// 示例:
//
//	yamlVars     = { "env": "dev", "timeout": "30m" }
//	triggerVars  = { "env": "staging" }
//	executionVars = { "env": "prod", "debug": true }
//	结果         = { "env": "prod", "timeout": "30m", "debug": true }
func (m *VarsMerger) Merge(
	yamlVars map[string]interface{}, // Layer 1: YAML 默认值
	triggerVars map[string]interface{}, // Layer 2: 触发器绑定参数
	executionVars map[string]interface{}, // Layer 3: 执行时参数
) map[string]interface{} {
	merged := make(map[string]interface{})

	// Layer 1: YAML 默认值
	for k, v := range yamlVars {
		merged[k] = v
	}

	// Layer 2: 触发器绑定参数覆盖
	for k, v := range triggerVars {
		merged[k] = v
	}

	// Layer 3: 执行时参数覆盖（优先级最高）
	for k, v := range executionVars {
		merged[k] = v
	}

	return merged
}

// MergeWithDefaults 合并并提供默认值
// 如果某个 key 在所有三层中都不存在，则使用 defaults 中的值
func (m *VarsMerger) MergeWithDefaults(
	yamlVars map[string]interface{},
	triggerVars map[string]interface{},
	executionVars map[string]interface{},
	defaults map[string]interface{},
) map[string]interface{} {
	merged := make(map[string]interface{})

	// Layer 0: 默认值（最低优先级）
	for k, v := range defaults {
		merged[k] = v
	}

	// Layer 1: YAML 默认值
	for k, v := range yamlVars {
		merged[k] = v
	}

	// Layer 2: 触发器绑定参数
	for k, v := range triggerVars {
		merged[k] = v
	}

	// Layer 3: 执行时参数（最高优先级）
	for k, v := range executionVars {
		merged[k] = v
	}

	return merged
}

// ExtractLayer 提取指定优先级的变量（用于调试）
func (m *VarsMerger) ExtractLayer(layer int, yamlVars, triggerVars, executionVars map[string]interface{}) map[string]interface{} {
	switch layer {
	case 1:
		return copyMap(yamlVars)
	case 2:
		return copyMap(triggerVars)
	case 3:
		return copyMap(executionVars)
	default:
		return make(map[string]interface{})
	}
}

// copyMap 深拷贝 map
func copyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
