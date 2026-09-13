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

type CodexRateLimitWindowInfo struct {
	UsedPercent int    `json:"used_percent"`
	ResetAfter  string `json:"reset_after"`
}

type CodexQuotaDetailedInfo struct {
	PlanType              string                    `json:"plan_type"`
	PrimaryWindow         *CodexRateLimitWindowInfo `json:"primary_window,omitempty"`
	SecondaryWindow       *CodexRateLimitWindowInfo `json:"secondary_window,omitempty"`
	AvailableResetCredits int                       `json:"available_reset_credits"`
}

type XaiQuotaDetailedInfo struct {
	WeeklyUsedPercent int    `json:"weekly_used_percent"`
	WeeklyResetAfter  string `json:"weekly_reset_after,omitempty"`
	GrokBuildUsed     int    `json:"grok_build_used"`
	GrokChatUsed      string `json:"grok_chat_used"`
}

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
	AuthIndex      string         `json:"-"`
	// Real-time fetched rich quotas
	CodexDetail *CodexQuotaDetailedInfo `json:"codex_detail,omitempty"`
	XaiDetail   *XaiQuotaDetailedInfo   `json:"xai_detail,omitempty"`
}

type CpaProbeResult struct {
	Online         bool               `json:"online"`
	Latency        int64              `json:"latency"`
	HttpStatus     int                `json:"http_status"`
	Version        string             `json:"version"`
	ModelCount     int                `json:"model_count"`
	Models         []string           `json:"models"`
	AuthFilesCount int                `json:"auth_files_count"`
	AuthFiles      []*CpaAuthFileInfo `json:"auth_files,omitempty"`
	Error          string             `json:"error"`
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
		AuthIndex   string `json:"auth_index"`
		ProjectId   string `json:"project_id"`
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
			return http.ErrUseLastResponse
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
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
	authFilesFetched := false
	if apiKey != "" {
		authReqURL := normURL + "/v0/management/auth-files"
		authReq, authReqErr := http.NewRequestWithContext(reqCtx, http.MethodGet, authReqURL, nil)
		if authReqErr == nil {
			authReq.Header.Set("Authorization", "Bearer "+apiKey)
			authReq.Header.Set("Accept", "application/json")
			authResp, authDoErr := client.Do(authReq)
			if authDoErr == nil {
				defer authResp.Body.Close()
				if authResp.StatusCode == http.StatusOK {
					authBody, _ := io.ReadAll(io.LimitReader(authResp.Body, 2<<20))
					var rawAuth rawAuthFilesResponse
					if err := common.Unmarshal(authBody, &rawAuth); err == nil {
						authFilesFetched = true
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
								AuthIndex:      f.AuthIndex,
							}

							providerLower := strings.ToLower(f.Provider)
							typeLower := strings.ToLower(f.Type)

							// 1. Fetch real-time Codex details if applicable
							if (providerLower == "codex" || typeLower == "codex") && f.AuthIndex != "" {
								if detail := fetchCodexRealtimeQuota(reqCtx, client, normURL, apiKey, f.AuthIndex, f.Account); detail != nil {
									info.CodexDetail = detail
								}
							}

							// 2. Fetch real-time xAI details if applicable
							if (providerLower == "xai" || typeLower == "xai") && f.AuthIndex != "" {
								if xaiDetail := fetchXaiRealtimeQuota(reqCtx, client, normURL, apiKey, f.AuthIndex); xaiDetail != nil {
									info.XaiDetail = xaiDetail
								}
							}

							authFilesList = append(authFilesList, info)
						}
					}
				}
			}
		}
	}

	modelsJoined := strings.Join(modelIDs, ",")
	authSummaryJSON := ""
	if authFilesFetched {
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

// RefreshSingleAuthFileQuota refreshes only one specific auth file on the node
func RefreshSingleAuthFileQuota(ctx context.Context, node *model.CpaNode, authFileID string) (*CpaAuthFileInfo, error) {
	if node == nil {
		return nil, fmt.Errorf("cpa node is nil")
	}
	authFileID = strings.TrimSpace(authFileID)
	if authFileID == "" {
		return nil, fmt.Errorf("auth_file_id is required")
	}

	apiKey := strings.TrimSpace(node.ApiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("node api_key is empty")
	}

	normURL := strings.TrimRight(node.NormalizedUrl, "/")
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Get auth file list from node
	authReqURL := normURL + "/v0/management/auth-files"
	authReq, err := http.NewRequestWithContext(ctx, http.MethodGet, authReqURL, nil)
	if err != nil {
		return nil, err
	}
	authReq.Header.Set("Authorization", "Bearer "+apiKey)
	authReq.Header.Set("Accept", "application/json")

	authResp, err := client.Do(authReq)
	if err != nil {
		return nil, err
	}
	defer authResp.Body.Close()

	authBody, _ := io.ReadAll(io.LimitReader(authResp.Body, 2<<20))
	var rawAuth rawAuthFilesResponse
	if err := common.Unmarshal(authBody, &rawAuth); err != nil {
		return nil, err
	}

	var targetFile *CpaAuthFileInfo
	var targetRaw struct {
		AuthIndex string
		Account   string
		Provider  string
	}

	for _, f := range rawAuth.Files {
		if f.Id == authFileID || f.Name == authFileID {
			targetFile = &CpaAuthFileInfo{
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
				AuthIndex:      f.AuthIndex,
			}
			targetRaw.AuthIndex = f.AuthIndex
			targetRaw.Account = f.Account
			targetRaw.Provider = strings.ToLower(f.Provider)
			break
		}
	}

	if targetFile == nil {
		return nil, fmt.Errorf("credential %q not found on node", authFileID)
	}

	if targetRaw.Provider == "codex" && targetRaw.AuthIndex != "" {
		targetFile.CodexDetail = fetchCodexRealtimeQuota(ctx, client, normURL, apiKey, targetRaw.AuthIndex, targetRaw.Account)
	} else if targetRaw.Provider == "xai" && targetRaw.AuthIndex != "" {
		targetFile.XaiDetail = fetchXaiRealtimeQuota(ctx, client, normURL, apiKey, targetRaw.AuthIndex)
	}

	return targetFile, nil
}

func fetchCodexRealtimeQuota(ctx context.Context, client *http.Client, baseURL, apiKey, authIndex, accountID string) *CodexQuotaDetailedInfo {
	callURL := baseURL + "/v0/management/api-call"
	header := map[string]string{
		"Authorization": "Bearer $TOKEN$",
		"Accept":        "application/json",
		"OpenAI-Beta":   "codex-1",
		"Originator":    "Codex Desktop",
	}
	if accountID != "" {
		header["chatgpt-account-id"] = accountID
	}

	reqPayload := map[string]any{
		"authIndex": authIndex,
		"method":    "GET",
		"url":       "https://chatgpt.com/backend-api/wham/usage",
		"header":    header,
	}
	bodyBytes, err := common.Marshal(reqPayload)
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	type whamApiResp struct {
		StatusCode int `json:"status_code"`
		Body       struct {
			PlanType  string `json:"plan_type"`
			RateLimit struct {
				PrimaryWindow struct {
					UsedPercent       int `json:"used_percent"`
					ResetAfterSeconds int `json:"reset_after_seconds"`
				} `json:"primary_window"`
				SecondaryWindow struct {
					UsedPercent       int `json:"used_percent"`
					ResetAfterSeconds int `json:"reset_after_seconds"`
				} `json:"secondary_window"`
			} `json:"rate_limit"`
			RateLimitResetCredits struct {
				AvailableCount int `json:"available_count"`
			} `json:"rate_limit_reset_credits"`
		} `json:"body"`
	}

	var parsed whamApiResp
	if err := common.Unmarshal(respBytes, &parsed); err != nil || parsed.StatusCode != http.StatusOK {
		return nil
	}

	return &CodexQuotaDetailedInfo{
		PlanType: parsed.Body.PlanType,
		PrimaryWindow: &CodexRateLimitWindowInfo{
			UsedPercent: parsed.Body.RateLimit.PrimaryWindow.UsedPercent,
			ResetAfter:  formatSecondsToFriendly(parsed.Body.RateLimit.PrimaryWindow.ResetAfterSeconds),
		},
		SecondaryWindow: &CodexRateLimitWindowInfo{
			UsedPercent: parsed.Body.RateLimit.SecondaryWindow.UsedPercent,
			ResetAfter:  formatSecondsToFriendly(parsed.Body.RateLimit.SecondaryWindow.ResetAfterSeconds),
		},
		AvailableResetCredits: parsed.Body.RateLimitResetCredits.AvailableCount,
	}
}

func fetchXaiRealtimeQuota(ctx context.Context, client *http.Client, baseURL, apiKey, authIndex string) *XaiQuotaDetailedInfo {
	callURL := baseURL + "/v0/management/api-call"
	reqPayload := map[string]any{
		"authIndex": authIndex,
		"method":    "GET",
		"url":       "https://cli-chat-proxy.grok.com/v1/billing?format=credits",
		"header": map[string]string{
			"Authorization":         "Bearer $TOKEN$",
			"x-grok-client-version": "0.2.91",
			"user-agent":            "grok-pager/0.2.91 grok-shell/0.2.91 (macos; aarch64)",
		},
	}
	bodyBytes, err := common.Marshal(reqPayload)
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	type xaiApiResp struct {
		StatusCode int `json:"status_code"`
		Body       struct {
			Config struct {
				CreditUsagePercent float64 `json:"creditUsagePercent"`
				ProductUsage       []struct {
					Product      string  `json:"product"`
					UsagePercent float64 `json:"usagePercent"`
				} `json:"productUsage"`
				CurrentPeriod struct {
					End string `json:"end"`
				} `json:"currentPeriod"`
			} `json:"config"`
		} `json:"body"`
	}

	var parsed xaiApiResp
	if err := common.Unmarshal(respBytes, &parsed); err != nil || parsed.StatusCode != http.StatusOK {
		return nil
	}

	grokBuildUsed := 0
	for _, p := range parsed.Body.Config.ProductUsage {
		if strings.EqualFold(p.Product, "GrokBuild") {
			grokBuildUsed = int(p.UsagePercent)
		}
	}

	return &XaiQuotaDetailedInfo{
		WeeklyUsedPercent: int(parsed.Body.Config.CreditUsagePercent),
		WeeklyResetAfter:  parsed.Body.Config.CurrentPeriod.End,
		GrokBuildUsed:     grokBuildUsed,
		GrokChatUsed:      "--",
	}
}

func formatSecondsToFriendly(sec int) string {
	if sec <= 0 {
		return "OK"
	}
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	if sec < 3600 {
		return fmt.Sprintf("%dm", sec/60)
	}
	hours := sec / 3600
	mins := (sec % 3600) / 60
	if hours >= 24 {
		days := hours / 24
		remHours := hours % 24
		return fmt.Sprintf("%dd %dh", days, remHours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func sanitizeCpaError(msg, key string) string {
	if strings.TrimSpace(key) == "" {
		return msg
	}
	return strings.ReplaceAll(msg, strings.TrimSpace(key), "[REDACTED]")
}
