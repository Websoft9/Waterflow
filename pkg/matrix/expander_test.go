package matrix

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
)

// TestExpander_SimpleMatrix 测试简单矩阵展开
func TestExpander_SimpleMatrix(t *testing.T) {
	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"server": {"web1", "web2", "web3"},
			},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(job)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(instances))

	// 验证实例
	assert.Equal(t, 0, instances[0].Index)
	assert.Equal(t, "web1", instances[0].Matrix["server"])

	assert.Equal(t, 1, instances[1].Index)
	assert.Equal(t, "web2", instances[1].Matrix["server"])

	assert.Equal(t, 2, instances[2].Index)
	assert.Equal(t, "web3", instances[2].Matrix["server"])
}

// TestExpander_MultiDimension 测试多维矩阵展开 (笛卡尔积)
func TestExpander_MultiDimension(t *testing.T) {
	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"server": {"web1", "web2"},
				"env":    {"prod", "staging"},
			},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(job)

	assert.NoError(t, err)
	assert.Equal(t, 4, len(instances))

	// 验证笛卡尔积
	// {server: web1, env: prod}
	// {server: web1, env: staging}
	// {server: web2, env: prod}
	// {server: web2, env: staging}

	// 由于map迭代顺序不确定,验证所有组合存在
	combinations := make(map[string]bool)
	for _, inst := range instances {
		key := inst.Matrix["server"].(string) + "-" + inst.Matrix["env"].(string)
		combinations[key] = true
	}

	assert.True(t, combinations["web1-prod"])
	assert.True(t, combinations["web1-staging"])
	assert.True(t, combinations["web2-prod"])
	assert.True(t, combinations["web2-staging"])
}

// TestExpander_ThreeDimension 测试三维矩阵
func TestExpander_ThreeDimension(t *testing.T) {
	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"os":      {"ubuntu", "debian"},
				"arch":    {"amd64", "arm64"},
				"version": {1.20, 1.21},
			},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(job)

	assert.NoError(t, err)
	assert.Equal(t, 8, len(instances)) // 2 * 2 * 2 = 8

	// 验证至少一个完整组合
	found := false
	for _, inst := range instances {
		if inst.Matrix["os"] == "ubuntu" &&
			inst.Matrix["arch"] == "amd64" &&
			inst.Matrix["version"] == 1.20 {
			found = true
			break
		}
	}
	assert.True(t, found)
}

// TestExpander_DifferentTypes 测试不同数据类型
func TestExpander_DifferentTypes(t *testing.T) {
	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"version": {1.20, 1.21, 1.22},   // float64
				"os":      {"ubuntu", "debian"}, // string
				"enabled": {true, false},        // bool
			},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(job)

	assert.NoError(t, err)
	assert.Equal(t, 12, len(instances)) // 3 * 2 * 2 = 12

	// 验证数据类型
	for _, inst := range instances {
		assert.IsType(t, float64(0), inst.Matrix["version"])
		assert.IsType(t, "", inst.Matrix["os"])
		assert.IsType(t, true, inst.Matrix["enabled"])
	}
}

// TestExpander_EmptyMatrix 测试空矩阵错误
func TestExpander_EmptyMatrix(t *testing.T) {
	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"server": {}, // 空数组
			},
		},
	}

	expander := NewExpander(256)
	_, err := expander.Expand(job)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "matrix dimension 'server' is empty")
}

// TestExpander_CombinationsLimit 测试组合数限制
func TestExpander_CombinationsLimit(t *testing.T) {
	// 创建超过256组合的矩阵
	matrix := make(map[string][]interface{})

	// 10 * 10 * 3 = 300 > 256
	matrix["a"] = make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		matrix["a"][i] = i
	}

	matrix["b"] = make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		matrix["b"][i] = i
	}

	matrix["c"] = []interface{}{1, 2, 3}

	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: matrix,
		},
	}

	expander := NewExpander(256)
	_, err := expander.Expand(job)

	assert.Error(t, err)

	matrixErr, ok := err.(*MatrixError)
	assert.True(t, ok)
	assert.Equal(t, "matrix_combinations_exceed_limit", matrixErr.Type)
	assert.Equal(t, 300, matrixErr.Combinations)
	assert.Equal(t, 256, matrixErr.Limit)
	assert.Contains(t, matrixErr.Suggestion, "Reduce matrix dimensions")
}

// TestExpander_NoStrategy 测试无策略的Job返回单实例
func TestExpander_NoStrategy(t *testing.T) {
	job := &dsl.Job{
		Strategy: nil,
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(job)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(instances))
	assert.Equal(t, 0, instances[0].Index)
	assert.Nil(t, instances[0].Matrix)
}

// TestExpander_EmptyStrategyMatrix 测试空矩阵map返回单实例
func TestExpander_EmptyStrategyMatrix(t *testing.T) {
	job := &dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(job)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(instances))
}
