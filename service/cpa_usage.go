package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

type CpaNodeUsage struct {
	Window           string  `json:"window"`
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Quota            int64   `json:"quota"`
	ErrorCount       int64   `json:"error_count"`
	ErrorRate        float64 `json:"error_rate"`
}

type CpaNodeChannelInfo struct {
	Id         int      `json:"id"`
	Name       string   `json:"name"`
	Type       int      `json:"type"`
	Status     int      `json:"status"`
	ModelCount int      `json:"model_count"`
	Models     []string `json:"models"`
}

type usageCacheItem struct {
	usage     CpaNodeUsage
	expiresAt time.Time
}

var (
	cpaUsageCache = make(map[string]usageCacheItem)
	cpaUsageMu    sync.RWMutex
)

// GetChannelsForCpaNode finds all channels whose normalized baseURL matches the node
func GetChannelsForCpaNode(node *model.CpaNode) ([]*CpaNodeChannelInfo, error) {
	if node == nil {
		return nil, fmt.Errorf("cpa node is nil")
	}
	targetNorm := node.NormalizedUrl
	if targetNorm == "" {
		norm, err := model.NormalizeCpaBaseURL(node.BaseUrl)
		if err != nil {
			return nil, err
		}
		targetNorm = norm
	}

	var channels []*model.Channel
	if err := model.DB.Select("id, name, type, status, base_url, other, models").Find(&channels).Error; err != nil {
		return nil, err
	}

	var matched []*CpaNodeChannelInfo
	for _, ch := range channels {
		if ch.BaseURL == nil {
			continue
		}
		rawBaseURL := strings.TrimSpace(*ch.BaseURL)
		if rawBaseURL == "" {
			continue
		}
		chNorm, err := model.NormalizeCpaBaseURL(rawBaseURL)
		if err != nil {
			continue
		}
		if chNorm == targetNorm {
			modelList := splitModels(ch.Models)
			matched = append(matched, &CpaNodeChannelInfo{
				Id:         ch.Id,
				Name:       ch.Name,
				Type:       ch.Type,
				Status:     ch.Status,
				ModelCount: len(modelList),
				Models:     modelList,
			})
		}
	}
	return matched, nil
}

// GetCpaNodeUsage calculates usage from LOG_DB for the given node and window ('today', '24h', '7d')
func GetCpaNodeUsage(node *model.CpaNode, window string, fresh bool) (*CpaNodeUsage, error) {
	if window == "" {
		window = "today"
	}

	cacheKey := fmt.Sprintf("%d:%s", node.Id, window)
	if !fresh {
		cpaUsageMu.RLock()
		if item, ok := cpaUsageCache[cacheKey]; ok && time.Now().Before(item.expiresAt) {
			cpaUsageMu.RUnlock()
			return &item.usage, nil
		}
		cpaUsageMu.RUnlock()
	}

	channels, err := GetChannelsForCpaNode(node)
	if err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		empty := &CpaNodeUsage{Window: window}
		return empty, nil
	}

	var channelIDs []int
	for _, ch := range channels {
		channelIDs = append(channelIDs, ch.Id)
	}

	var startTimestamp int64
	now := time.Now()
	switch window {
	case "24h":
		startTimestamp = now.Add(-24 * time.Hour).Unix()
	case "7d":
		startTimestamp = now.Add(-7 * 24 * time.Hour).Unix()
	case "today":
		fallthrough
	default:
		window = "today"
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		startTimestamp = todayStart.Unix()
	}

	type aggregateResult struct {
		Requests         int64
		PromptTokens     int64
		CompletionTokens int64
		Quota            int64
	}

	var agg aggregateResult
	// Query consume logs (type = 2)
	err = model.LOG_DB.Model(&model.Log{}).
		Select("COUNT(*) as requests, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens, COALESCE(SUM(quota), 0) as quota").
		Where("type = ? AND channel_id IN (?) AND created_at >= ?", model.LogTypeConsume, channelIDs, startTimestamp).
		Scan(&agg).Error
	if err != nil {
		return nil, err
	}

	// Query error logs (type = 5)
	var errorCount int64
	err = model.LOG_DB.Model(&model.Log{}).
		Where("type = ? AND channel_id IN (?) AND created_at >= ?", model.LogTypeError, channelIDs, startTimestamp).
		Count(&errorCount).Error
	if err != nil {
		return nil, err
	}

	totalAll := agg.Requests + errorCount
	var errorRate float64
	if totalAll > 0 {
		errorRate = float64(errorCount) / float64(totalAll)
	}

	usage := CpaNodeUsage{
		Window:           window,
		Requests:         agg.Requests,
		PromptTokens:     agg.PromptTokens,
		CompletionTokens: agg.CompletionTokens,
		Quota:            agg.Quota,
		ErrorCount:       errorCount,
		ErrorRate:        errorRate,
	}

	cpaUsageMu.Lock()
	cpaUsageCache[cacheKey] = usageCacheItem{
		usage:     usage,
		expiresAt: time.Now().Add(60 * time.Second),
	}
	cpaUsageMu.Unlock()

	return &usage, nil
}

func splitModels(modelsStr string) []string {
	var list []string
	parts := strings.Split(modelsStr, ",")
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return list
}
