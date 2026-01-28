package dsl

import (
	"fmt"
	"sort"
)

// Expander Matrix 展开器
type Expander struct {
	maxCombinations int
}

// NewExpander 创建展开器
func NewExpander(maxCombinations int) *Expander {
	return &Expander{
		maxCombinations: maxCombinations,
	}
}

// Expand 展开 Matrix 为多个实例
func (e *Expander) Expand(job *Job) ([]*MatrixInstance, error) {
	if job.Strategy == nil || len(job.Strategy.Matrix) == 0 {
		// 无 Matrix,返回单个实例
		return []*MatrixInstance{{
			Index:  0,
			Matrix: nil,
		}}, nil
	}

	// 1. 验证 Matrix
	if err := e.validateMatrix(job.Strategy.Matrix); err != nil {
		return nil, err
	}

	// 2. 计算笛卡尔积
	instances := e.cartesianProduct(job.Strategy.Matrix)

	// 3. 检查组合数限制
	if len(instances) > e.maxCombinations {
		return nil, &MatrixError{
			Type:         "matrix_combinations_exceed_limit",
			Combinations: len(instances),
			Limit:        e.maxCombinations,
			Suggestion:   "Reduce matrix dimensions or split into multiple jobs",
		}
	}

	return instances, nil
}

// validateMatrix 验证 Matrix 配置
func (e *Expander) validateMatrix(matrix map[string][]interface{}) error {
	for key, values := range matrix {
		if len(values) == 0 {
			return fmt.Errorf("matrix dimension '%s' is empty", key)
		}
	}
	return nil
}

// cartesianProduct 计算笛卡尔积 (高性能版本)
func (e *Expander) cartesianProduct(matrix map[string][]interface{}) []*MatrixInstance {
	// 获取所有维度并排序以保证顺序一致性
	dimensions := make([]string, 0, len(matrix))
	for dim := range matrix {
		dimensions = append(dimensions, dim)
	}
	sort.Strings(dimensions)

	// 预计算总组合数
	totalCombinations := 1
	for _, dim := range dimensions {
		totalCombinations *= len(matrix[dim])
	}

	// 预分配结果切片和所有 MatrixInstance
	instances := make([]*MatrixInstance, totalCombinations)

	// 预分配所有索引数组
	indices := make([]int, len(dimensions))

	idx := 0
	for {
		// 创建当前组合 - 直接分配固定大小的 map
		combination := make(map[string]interface{}, len(dimensions))
		for i, dim := range dimensions {
			combination[dim] = matrix[dim][indices[i]]
		}

		instances[idx] = &MatrixInstance{
			Index:  idx,
			Matrix: combination,
		}
		idx++

		// 递增索引（类似进位）
		pos := len(indices) - 1
		for pos >= 0 {
			indices[pos]++
			if indices[pos] < len(matrix[dimensions[pos]]) {
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

	return instances
}
