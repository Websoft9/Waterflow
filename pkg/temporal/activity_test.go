package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestNewActivities(t *testing.T) {
	logger := zaptest.NewLogger(t)
	activities := NewActivities(logger)

	assert.NotNil(t, activities)
	assert.NotNil(t, activities.logger)
}

// Note: ExecuteStepActivity tests require Temporal test environment
// with proper context serialization. These tests should be run as
// integration tests with a running Temporal server.
// For unit testing, we test the individual components (ConditionEvaluator,
// WorkflowRenderer) separately in their respective test files.
