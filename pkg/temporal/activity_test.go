package temporal

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestNewActivities(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := node.NewRegistry()
	activities := NewActivities(logger, registry)

	assert.NotNil(t, activities)
	assert.NotNil(t, activities.logger)
	assert.NotNil(t, activities.nodeRegistry)
}

// Note: ExecuteStepActivity error handling tests (NonRetryableError) are covered
// in workflow_test.go TestToTemporalRetryPolicy_NonRetryableErrorTypes which验证
// the NonRetryableErrorTypes list configuration.
//
// Full integration tests with actual node execution require a real Temporal server
// and are located in test/integration/.
