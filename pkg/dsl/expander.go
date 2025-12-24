package dsl

import (
	"fmt"
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

// cartesianProduct 计算笛卡尔积
func (e *Expander) cartesianProduct(matrix map[string][]interface{}) []*MatrixInstance {
	// 获取所有维度 (需要确定顺序)
	dimensions := make([]string, 0, len(matrix))
	for dim := range matrix {
		dimensions = append(dimensions, dim)
	}

	// 递归生成组合
	instances := make([]*MatrixInstance, 0)
	e.generateCombinations(matrix, dimensions, 0, make(map[string]interface{}), &instances)

	return instances
}

// generateCombinations 递归生成组合
func (e *Expander) generateCombinations(
	matrix map[string][]interface{},
	dimensions []string,
	dimIndex int,
	current map[string]interface{},
	instances *[]*MatrixInstance,
) {
	if dimIndex == len(dimensions) {
		// 完成一个组合
		combination := make(map[string]interface{})
		for k, v := range current {
			combination[k] = v
		}

		*instances = append(*instances, &MatrixInstance{
			Index:  len(*instances),
			Matrix: combination,
		})
		return
	}

	// 遍历当前维度的所有值
	dim := dimensions[dimIndex]
	for _, value := range matrix[dim] {
		current[dim] = value
		e.generateCombinations(matrix, dimensions, dimIndex+1, current, instances)
	}
}
