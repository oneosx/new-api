package controller

import (
	"context"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

type cpaProbeHandler struct{}

func (cpaProbeHandler) Type() string { return model.SystemTaskTypeCpaProbe }

func (cpaProbeHandler) Enabled() bool { return true }

func (cpaProbeHandler) Interval() time.Duration {
	return 3 * time.Minute
}

func (cpaProbeHandler) NewPayload() any { return nil }

func (cpaProbeHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	nodes, err := model.GetEnabledCpaNodes()
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}

	if len(nodes) == 0 {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, map[string]any{
			"total":   0,
			"online":  0,
			"offline": 0,
			"message": "no enabled cpa nodes",
		}, nil)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	var mu sync.Mutex
	onlineCount := 0
	offlineCount := 0

	for _, n := range nodes {
		node := n
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := service.ProbeCpaNode(ctx, node)
			mu.Lock()
			defer mu.Unlock()
			if err == nil && res != nil && res.Online {
				onlineCount++
			} else {
				offlineCount++
			}
		}()
	}

	wg.Wait()

	result := map[string]any{
		"total":   len(nodes),
		"online":  onlineCount,
		"offline": offlineCount,
		"time":    time.Now().Format(time.RFC3339),
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, result, nil)
}
