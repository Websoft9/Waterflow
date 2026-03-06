package temporal

import (
	"context"
	"fmt"
	"sort"
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

// DescribeTaskQueue queries Temporal for task queue status.
// It queries both ACTIVITY and WORKFLOW poller types and merges the results so that
// agents which only register Workflow workers (no Activity workers) are not incorrectly
// reported as unavailable.
func (c *Client) DescribeTaskQueue(ctx context.Context, taskQueueName string) (*TaskQueueInfo, error) {
	queueTypes := []enums.TaskQueueType{
		enums.TASK_QUEUE_TYPE_ACTIVITY,
		enums.TASK_QUEUE_TYPE_WORKFLOW,
	}

	info := &TaskQueueInfo{
		Name:           taskQueueName,
		LastUpdateTime: time.Now(),
	}

	var lastErr error
	for _, qType := range queueTypes {
		request := &workflowservice.DescribeTaskQueueRequest{
			Namespace: c.config.Namespace,
			TaskQueue: &taskqueue.TaskQueue{
				Name: taskQueueName,
				Kind: enums.TASK_QUEUE_KIND_NORMAL,
			},
			TaskQueueType: qType,
		}

		resp, err := c.client.WorkflowService().DescribeTaskQueue(ctx, request)
		if err != nil {
			lastErr = err
			continue
		}

		info.Pollers += len(resp.GetPollers())

		// Count healthy pollers (last seen < 30s)
		for _, poller := range resp.GetPollers() {
			if poller.GetLastAccessTime() != nil {
				lastAccess := poller.GetLastAccessTime().AsTime()
				if time.Since(lastAccess) < 30*time.Second {
					info.HealthyPollers++
				}
			}
		}

		// Get task backlog from Activity queue (canonical source)
		// nolint:staticcheck // Using deprecated API until Temporal provides replacement
		if qType == enums.TASK_QUEUE_TYPE_ACTIVITY && resp.GetTaskQueueStatus() != nil {
			info.TaskBacklog = resp.GetTaskQueueStatus().GetBacklogCountHint() //nolint:staticcheck
		}
	}

	// Return error only if both queries failed
	if info.Pollers == 0 && lastErr != nil {
		return nil, lastErr
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

// DiscoverTaskQueues discovers unique task queue names from recent workflow executions.
// It queries Temporal workflow history to find all task queues that have been active,
// returning up to maxQueues unique names in sorted order.
func (c *Client) DiscoverTaskQueues(ctx context.Context, maxQueues int) ([]string, error) {
	seen := make(map[string]bool)
	var queues []string
	var nextPageToken []byte

	for {
		resp, err := c.client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     c.config.Namespace,
			PageSize:      100,
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list workflows for task queue discovery: %w", err)
		}

		for _, execution := range resp.GetExecutions() {
			tq := execution.GetTaskQueue()
			if tq != "" && !seen[tq] {
				seen[tq] = true
				queues = append(queues, tq)
			}
			if len(queues) >= maxQueues {
				sort.Strings(queues)
				return queues, nil
			}
		}

		nextPageToken = resp.GetNextPageToken()
		if len(nextPageToken) == 0 {
			break
		}
	}

	sort.Strings(queues)
	return queues, nil
}
