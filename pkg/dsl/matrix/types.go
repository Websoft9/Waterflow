package matrix

import "fmt"

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
