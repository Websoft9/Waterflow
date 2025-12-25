package dsl

import (
	"testing"
)

// expandMatrix 是内部 Matrix 展开函数（用于基准测试）
// 实际实现在 Temporal Workflow 中，这里提取核心逻辑用于性能测试
func expandMatrix(matrix map[string][]interface{}) []map[string]interface{} {
	if len(matrix) == 0 {
		return nil
	}

	// 获取所有维度和值
	var keys []string
	var values [][]interface{}
	for k, v := range matrix {
		keys = append(keys, k)
		values = append(values, v)
	}

	// 计算笛卡尔积
	result := []map[string]interface{}{}
	indices := make([]int, len(keys))

	for {
		// 创建当前组合
		combo := make(map[string]interface{})
		for i, key := range keys {
			combo[key] = values[i][indices[i]]
		}
		result = append(result, combo)

		// 递增索引（类似进位）
		pos := len(indices) - 1
		for pos >= 0 {
			indices[pos]++
			if indices[pos] < len(values[pos]) {
				break
			}
			indices[pos] = 0
			pos--
		}

		// 所有维度都已遍历完
		if pos < 0 {
			break
		}
	}

	return result
}

// BenchmarkMatrixExpansion 测试 Matrix 扩展性能
// AC5 要求: 256 个实例扩展 <10ms
func BenchmarkMatrixExpansion(b *testing.B) {
	tests := []struct {
		name     string
		matrix   map[string][]interface{}
		expected int
	}{
		{
			name: "small_matrix_2x2",
			matrix: map[string][]interface{}{
				"os":      {"ubuntu", "centos"},
				"version": {"20.04", "22.04"},
			},
			expected: 4,
		},
		{
			name: "medium_matrix_4x4x4",
			matrix: map[string][]interface{}{
				"os":      {"ubuntu", "centos", "debian", "alpine"},
				"version": {"20.04", "22.04", "23.04", "24.04"},
				"arch":    {"amd64", "arm64", "armv7", "ppc64le"},
			},
			expected: 64,
		},
		{
			name: "large_matrix_4x4x4x4_256_instances",
			matrix: map[string][]interface{}{
				"os":      {"ubuntu", "centos", "debian", "alpine"},
				"version": {"20.04", "22.04", "23.04", "24.04"},
				"arch":    {"amd64", "arm64", "armv7", "ppc64le"},
				"env":     {"prod", "staging", "dev", "test"},
			},
			expected: 256,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				combinations := expandMatrix(tt.matrix)
				if len(combinations) != tt.expected {
					b.Fatalf("expected %d combinations, got %d", tt.expected, len(combinations))
				}
			}
		})
	}
}

// BenchmarkMatrixExpansion256_WithAllocation 专门测试 256 实例扩展性能（含内存分配）
// AC5: 256 个实例扩展 <10ms
func BenchmarkMatrixExpansion256_WithAllocation(b *testing.B) {
	matrix := map[string][]interface{}{
		"os":      {"ubuntu", "centos", "debian", "alpine"},
		"version": {"20.04", "22.04", "23.04", "24.04"},
		"arch":    {"amd64", "arm64", "armv7", "ppc64le"},
		"env":     {"prod", "staging", "dev", "test"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combinations := expandMatrix(matrix)
		if len(combinations) != 256 {
			b.Fatalf("expected 256 combinations, got %d", len(combinations))
		}
	}
}

// BenchmarkMatrixExpansion256_SingleOp 测试单次 256 实例扩展性能
func BenchmarkMatrixExpansion256_SingleOp(b *testing.B) {
	matrix := map[string][]interface{}{
		"os":      {"ubuntu", "centos", "debian", "alpine"},
		"version": {"20.04", "22.04", "23.04", "24.04"},
		"arch":    {"amd64", "arm64", "armv7", "ppc64le"},
		"env":     {"prod", "staging", "dev", "test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = expandMatrix(matrix)
	}
}

// BenchmarkMatrixCombinationBuilding 测试组合构建性能
func BenchmarkMatrixCombinationBuilding(b *testing.B) {
	matrix := map[string][]interface{}{
		"server":  {"web1", "web2", "web3", "web4"},
		"version": {"1.0", "2.0", "3.0", "4.0"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combinations := expandMatrix(matrix)
		for _, combo := range combinations {
			// Access combo values
			_ = combo["server"]
			_ = combo["version"]
		}
	}
}
