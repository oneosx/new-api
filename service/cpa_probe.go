package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type CpaProbeResult struct {
	Online     bool     `json:"online"`
	Latency    int64    `json:"latency"`
	HttpStatus int      `json:"http_status"`
	Version    string   `json:"version"`
	ModelCount int      `json:"model_count"`
	Models     []string `json:"models"`
	Error      string   `json:"error"`
}

type openAIModelsResponse struct {
	Data []struct {
		Id string `json:"id"`
	} `json:"data"`
}

func ProbeCpaNode(ctx context.Context, node *model.CpaNode) (*CpaProbeResult, error) {
	if node == nil {
		return nil, fmt.Errorf("cpa node is nil")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(node.BaseUrl), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("cpa base url is empty")
	}

	reqURL := baseURL + "/v1/models"
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(node.ApiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(node.ApiKey))
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		errMsg := sanitizeCpaError(err.Error(), node.ApiKey)
		_ = node.UpdateProbeSnapshot(false, latency, 0, "", "", errMsg)
		return &CpaProbeResult{
			Online:     false,
			Latency:    latency,
			HttpStatus: 0,
			Error:      errMsg,
		}, nil
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()
	version := resp.Header.Get("X-CPA-VERSION")
	if version == "" {
		version = resp.Header.Get("X-SERVER-VERSION")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB limit
	if err != nil {
		errMsg := sanitizeCpaError(err.Error(), node.ApiKey)
		_ = node.UpdateProbeSnapshot(false, latency, resp.StatusCode, version, "", errMsg)
		return &CpaProbeResult{
			Online:     false,
			Latency:    latency,
			HttpStatus: resp.StatusCode,
			Version:    version,
			Error:      errMsg,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		if len(errMsg) > 250 {
			errMsg = errMsg[:250] + "..."
		}
		errMsg = sanitizeCpaError(errMsg, node.ApiKey)
		_ = node.UpdateProbeSnapshot(false, latency, resp.StatusCode, version, "", errMsg)
		return &CpaProbeResult{
			Online:     false,
			Latency:    latency,
			HttpStatus: resp.StatusCode,
			Version:    version,
			Error:      errMsg,
		}, nil
	}

	var parsed openAIModelsResponse
	if err := common.Unmarshal(body, &parsed); err != nil {
		errMsg := sanitizeCpaError(fmt.Sprintf("JSON parse error: %s", err.Error()), node.ApiKey)
		_ = node.UpdateProbeSnapshot(false, latency, resp.StatusCode, version, "", errMsg)
		return &CpaProbeResult{
			Online:     false,
			Latency:    latency,
			HttpStatus: resp.StatusCode,
			Version:    version,
			Error:      errMsg,
		}, nil
	}

	var modelIDs []string
	seen := make(map[string]struct{})
	for _, m := range parsed.Data {
		id := strings.TrimSpace(m.Id)
		if id != "" {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				modelIDs = append(modelIDs, id)
			}
		}
	}

	modelsJoined := strings.Join(modelIDs, ",")
	_ = node.UpdateProbeSnapshot(true, latency, resp.StatusCode, version, modelsJoined, "")

	return &CpaProbeResult{
		Online:     true,
		Latency:    latency,
		HttpStatus: resp.StatusCode,
		Version:    version,
		ModelCount: len(modelIDs),
		Models:     modelIDs,
		Error:      "",
	}, nil
}

func sanitizeCpaError(msg, key string) string {
	if strings.TrimSpace(key) == "" {
		return msg
	}
	return strings.ReplaceAll(msg, strings.TrimSpace(key), "[REDACTED]")
}
