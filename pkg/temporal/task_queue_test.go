package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskQueueInfo(t *testing.T) {
	info := &TaskQueueInfo{
		Name:           "test-queue",
		Pollers:        5,
		HealthyPollers: 3,
		TaskBacklog:    100,
	}

	assert.Equal(t, "test-queue", info.Name)
	assert.Equal(t, 5, info.Pollers)
	assert.Equal(t, 3, info.HealthyPollers)
	assert.Equal(t, int64(100), info.TaskBacklog)
}

// Note: DescribeTaskQueue and ListTaskQueues require a real Temporal server
// and are tested in integration tests (test/integration/).
// These tests verify the data structures are properly defined.
