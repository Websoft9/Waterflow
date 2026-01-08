package temporal

import (
	"context"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.uber.org/zap"
)

// TaskQueueInfo represents task queue information
type TaskQueueInfo struct {
	Name           string
	Pollers        int
	HealthyPollers int
	TaskBacklog    int64
	LastUpdateTime time.Time
}

// DescribeTaskQueue queries Temporal for task queue status
// Returns poller information including connected workers and their health status
func (c *Client) DescribeTaskQueue(ctx context.Context, taskQueueName string) (*TaskQueueInfo, error) {
	request := &workflowservice.DescribeTaskQueueRequest{
		Namespace: c.config.Namespace,
		TaskQueue: &taskqueue.TaskQueue{
			Name: taskQueueName,
			Kind: enums.TASK_QUEUE_KIND_NORMAL,
		},
		TaskQueueType: enums.TASK_QUEUE_TYPE_ACTIVITY,
	}

	resp, err := c.client.WorkflowService().DescribeTaskQueue(ctx, request)
	if err != nil {
		return nil, err
	}

	info := &TaskQueueInfo{
		Name:           taskQueueName,
		Pollers:        len(resp.GetPollers()),
		LastUpdateTime: time.Now(),
	}

	// Count healthy pollers (last seen < 30s)
	healthyCount := 0
	for _, poller := range resp.GetPollers() {
		if poller.GetLastAccessTime() != nil {
			lastAccess := poller.GetLastAccessTime().AsTime()
			if time.Since(lastAccess) < 30*time.Second {
				healthyCount++
			}
		}
	}
	info.HealthyPollers = healthyCount

	// Get task backlog if available
	// nolint:staticcheck // Using deprecated API until Temporal provides replacement
	if resp.GetTaskQueueStatus() != nil {
		info.TaskBacklog = resp.GetTaskQueueStatus().GetBacklogCountHint() //nolint:staticcheck
	}

	return info, nil
}

// ListTaskQueues returns information for multiple task queues
func (c *Client) ListTaskQueues(ctx context.Context, taskQueues []string) (map[string]*TaskQueueInfo, error) {
	result := make(map[string]*TaskQueueInfo)

	for _, name := range taskQueues {
		info, err := c.DescribeTaskQueue(ctx, name)
		if err != nil {
			// Log error but continue with other queues
			c.logger.Warn("Failed to describe task queue",
				zap.String("task_queue", name),
				zap.Error(err),
			)
			continue
		}
		result[name] = info
	}

	return result, nil
}
