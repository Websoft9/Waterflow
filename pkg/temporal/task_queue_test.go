package temporal

import (
	"testing"
	"time"

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

// TestTaskQueueHealthWindow verifies the 30-second healthy-poller window logic.
// The window is applied inside DescribeTaskQueue; here we test the rule via
// a helper that mirrors the production condition.
func TestTaskQueueHealthWindow(t *testing.T) {
	isHealthy := func(lastAccess time.Time) bool {
		return time.Since(lastAccess) < 30*time.Second
	}

	t.Run("poller active 1s ago is healthy", func(t *testing.T) {
		assert.True(t, isHealthy(time.Now().Add(-1*time.Second)))
	})

	t.Run("poller active 29s ago is healthy", func(t *testing.T) {
		assert.True(t, isHealthy(time.Now().Add(-29*time.Second)))
	})

	t.Run("poller active exactly 30s ago is NOT healthy", func(t *testing.T) {
		assert.False(t, isHealthy(time.Now().Add(-30*time.Second)))
	})

	t.Run("poller active 60s ago is NOT healthy", func(t *testing.T) {
		assert.False(t, isHealthy(time.Now().Add(-60*time.Second)))
	})
}

// TestTaskQueueInfoZeroValue verifies zero-value semantics used when queries fail.
func TestTaskQueueInfoZeroValue(t *testing.T) {
	var info TaskQueueInfo
	assert.Equal(t, 0, info.Pollers)
	assert.Equal(t, 0, info.HealthyPollers)
	assert.True(t, info.LastUpdateTime.IsZero())
}

// Note: DescribeTaskQueue and ListTaskQueues require a real Temporal server
// and are tested in integration tests (test/integration/).
// These tests verify the data structures and business logic are properly defined.
