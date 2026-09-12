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

type CpaAuthFileInfo struct {
	Id             string         `json:"id"`
	Name           string         `json:"name"`
	Provider       string         `json:"provider"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	Disabled       bool           `json:"disabled"`
	Email          string         `json:"email,omitempty"`
	Account        string         `json:"account,omitempty"`
	PlanType       string         `json:"plan_type,omitempty"`
	SubscriptionTo string         `json:"subscription_to,omitempty"`
	QuotaSignals   map[string]any `json:"quota_signals,omitempty"`
	ModelQuotas    map[string]any `json:"model_quotas,omitempty"`
	Success        int64          `json:"success"`
	Failed         int64          `json:"failed"`
	LastRefresh    string         `json:"last_refresh,omitempty"`
}

type CpaProbeResult struct {
	Online          bool               `json:"online"`
	Latency         int64              `json:"latency"`
	HttpStatus      int                `json:"http_status"`
	Version         string             `json:"version"`
	ModelCount      int                `json:"model_count"`
	Models          []string           `json:"models"`
	AuthFilesCount  int                `json:"auth_files_count"`
	AuthFiles       []*CpaAuthFileInfo `json:"auth_files,omitempty"`
	Error           string             `json:"error"`
}

type openAIModelsResponse struct {
	Data []struct {
		Id string `json:"id"`
	} `json:"data"`
}

type rawAuthFilesResponse struct {
	Files []struct {
		Id          string `json:"id"`
		Name        string `json:"name"`
		Provider    string `json:"provider"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		Disabled    bool   `json:"disabled"`
		Email       string `json:"email"`
		Account     string `json:"account"`
		LastRefresh string `json:"last_refresh"`
		Success     int64  `json:"success"`
		Failed      int64  `json:"failed"`
		IdToken     struct {
			PlanType                       string `json:"plan_type"`
			ChatgptSubscriptionActiveUntil string `json:"chatgpt_subscription_active_until"`
		} `json:"id_token"`
		Quota struct {
			Signals map[string]any `json:"signals"`
		} `json:"quota"`
		ModelQuotas map[string]any `json:"model_quotas"`
	} `json:"files"`
}

func ProbeCpaNode(ctx context.Context, node *model.CpaNode) (*CpaProbeResult, error) {
	if node == nil {
		return nil, fmt.Errorf("cpa node is nil")
	}

	normURL := node.NormalizedUrl
	if normURL == "" {
		norm, err := model.NormalizeCpaBaseURL(node.BaseUrl)
		if err != nil {
			return nil, err
		}
		normURL = norm
	}
	normURL = strings.TrimRight(normURL, "/")

	reqURL := normURL + "/v1/models"
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	apiKey := strings.TrimSpace(node.ApiKey)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Disable open redirects for security
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		errMsg := sanitizeCpaError(err.Error(), apiKey)
		_ = node.UpdateProbeSnapshotWithAuth(false, latency, 0, "", "", errMsg, "")
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
		errMsg := sanitizeCpaError(err.Error(), apiKey)
		_ = node.UpdateProbeSnapshotWithAuth(false, latency, resp.StatusCode, version, "", errMsg, "")
		return &CpaProbeResult{
			Online:     false,
			Latency:    latency,
			HttpStatus: resp.StatusCode,
			Version:    version,
			Error:      errMsg,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		_ = node.UpdateProbeSnapshotWithAuth(false, latency, resp.StatusCode, version, "", errMsg, "")
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
		errMsg := "JSON parse error"
		_ = node.UpdateProbeSnapshotWithAuth(false, latency, resp.StatusCode, version, "", errMsg, "")
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

	// Fetch /v0/management/auth-files for credentials & quota
	var authFilesList []*CpaAuthFileInfo
	if apiKey != "" {
		authReqURL := normURL + "/v0/management/auth-files"
		authReq, authReqErr := http.NewRequestWithContext(reqCtx, http.MethodGet, authReqURL, nil)
		if authReqErr == nil {
			authReq.Header.Set("Authorization", "Bearer "+apiKey)
			authReq.Header.Set("Accept", "application/json")
			authResp, authDoErr := client.Do(authReq)
			if authDoErr == nil && authResp.StatusCode == http.StatusOK {
				defer authResp.Body.Close()
				authBody, _ := io.ReadAll(io.LimitReader(authResp.Body, 2<<20))
				var rawAuth rawAuthFilesResponse
				if err := common.Unmarshal(authBody, &rawAuth); err == nil {
					for _, f := range rawAuth.Files {
						info := &CpaAuthFileInfo{
							Id:             f.Id,
							Name:           f.Name,
							Provider:       f.Provider,
							Type:           f.Type,
							Status:         f.Status,
							Disabled:       f.Disabled,
							Email:          f.Email,
							Account:        f.Account,
							PlanType:       f.IdToken.PlanType,
							SubscriptionTo: f.IdToken.ChatgptSubscriptionActiveUntil,
							QuotaSignals:   f.Quota.Signals,
							ModelQuotas:    f.ModelQuotas,
							Success:        f.Success,
							Failed:         f.Failed,
							LastRefresh:    f.LastRefresh,
						}
						authFilesList = append(authFilesList, info)
					}
				}
			}
		}
	}

	modelsJoined := strings.Join(modelIDs, ",")
	authSummaryJSON := ""
	if len(authFilesList) > 0 {
		if b, err := common.Marshal(authFilesList); err == nil {
			authSummaryJSON = string(b)
		}
	}

	_ = node.UpdateProbeSnapshotWithAuth(true, latency, resp.StatusCode, version, modelsJoined, "", authSummaryJSON)

	return &CpaProbeResult{
		Online:         true,
		Latency:        latency,
		HttpStatus:     resp.StatusCode,
		Version:        version,
		ModelCount:     len(modelIDs),
		Models:         modelIDs,
		AuthFilesCount: len(authFilesList),
		AuthFiles:      authFilesList,
		Error:          "",
	}, nil
}

func sanitizeCpaError(msg, key string) string {
	if strings.TrimSpace(key) == "" {
		return msg
	}
	return strings.ReplaceAll(msg, strings.TrimSpace(key), "[REDACTED]")
}
