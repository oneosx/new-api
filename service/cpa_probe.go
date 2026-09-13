package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type CodexRateLimitWindowInfo struct {
	UsedPercent      int    `json:"used_percent"`
	RemainingPercent int    `json:"remaining_percent"`
	ResetAfter       string `json:"reset_after"`
}

type CodexQuotaDetailedInfo struct {
	PlanType              string                    `json:"plan_type"`
	PrimaryWindow         *CodexRateLimitWindowInfo `json:"primary_window,omitempty"`
	SecondaryWindow       *CodexRateLimitWindowInfo `json:"secondary_window,omitempty"`
	AvailableResetCredits int                       `json:"available_reset_credits"`
}

type XaiQuotaDetailedInfo struct {
	WeeklyUsedPercent      int    `json:"weekly_used_percent"`
	WeeklyRemainingPercent int    `json:"weekly_remaining_percent"`
	WeeklyResetAfter       string `json:"weekly_reset_after,omitempty"`
	GrokBuildUsed          int    `json:"grok_build_used"`
	GrokBuildRemaining     int    `json:"grok_build_remaining"`
	GrokChatUsed           string `json:"grok_chat_used"`
}

type AntigravityQuotaBucket struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Window           string `json:"window,omitempty"`
	RemainingPercent int    `json:"remaining_percent"`
	ResetAfter       string `json:"reset_after,omitempty"`
	Description      string `json:"description,omitempty"`
}

type AntigravityQuotaGroup struct {
	ID          string                   `json:"id"`
	Label       string                   `json:"label"`
	Description string                   `json:"description,omitempty"`
	Buckets     []AntigravityQuotaBucket `json:"buckets,omitempty"`
}

type AntigravityQuotaDetailedInfo struct {
	Plan   string                  `json:"plan,omitempty"`
	Groups []AntigravityQuotaGroup `json:"groups,omitempty"`
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
	CodexDetail       *CodexQuotaDetailedInfo       `json:"codex_detail,omitempty"`
	XaiDetail         *XaiQuotaDetailedInfo         `json:"xai_detail,omitempty"`
	AntigravityDetail *AntigravityQuotaDetailedInfo `json:"antigravity_detail,omitempty"`
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
		ProjectID   string `json:"projectId"`
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

							quotaCtx, quotaCancel := context.WithTimeout(context.Background(), 25*time.Second)
							quotaClient := &http.Client{Timeout: 20 * time.Second}

							// 1. Fetch real-time Codex details if applicable
							if (providerLower == "codex" || typeLower == "codex") && f.AuthIndex != "" {
								accountID := f.Account
								if accountID == "" && f.Email != "" {
									accountID = f.Email
								}
								if detail := fetchCodexRealtimeQuota(quotaCtx, quotaClient, normURL, apiKey, f.AuthIndex, accountID); detail != nil {
									info.CodexDetail = detail
								}
							}

							// 2. Fetch real-time xAI details if applicable
							if (providerLower == "xai" || typeLower == "xai") && f.AuthIndex != "" {
								if xaiDetail := fetchXaiRealtimeQuota(quotaCtx, quotaClient, normURL, apiKey, f.AuthIndex); xaiDetail != nil {
									info.XaiDetail = xaiDetail
								}
							}

							// 3. Fetch real-time Antigravity details if applicable
							if (providerLower == "antigravity" || typeLower == "antigravity") && f.AuthIndex != "" {
								projID := f.ProjectId
								if projID == "" {
									projID = f.ProjectID
								}
								if antiDetail := fetchAntigravityRealtimeQuota(quotaCtx, quotaClient, normURL, apiKey, f.AuthIndex, projID); antiDetail != nil {
									info.AntigravityDetail = antiDetail
								}
							}

							quotaCancel()
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
		ProjectId string
		Email     string
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
			targetRaw.ProjectId = f.ProjectId
			if targetRaw.ProjectId == "" {
				targetRaw.ProjectId = f.ProjectID
			}
			targetRaw.Email = f.Email
			break
		}
	}

	if targetFile == nil {
		return nil, fmt.Errorf("credential %q not found on node", authFileID)
	}

	quotaCtx, quotaCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer quotaCancel()
	quotaClient := &http.Client{Timeout: 20 * time.Second}

	if targetRaw.Provider == "codex" && targetRaw.AuthIndex != "" {
		accountID := targetRaw.Account
		if accountID == "" && targetRaw.Email != "" {
			accountID = targetRaw.Email
		}
		targetFile.CodexDetail = fetchCodexRealtimeQuota(quotaCtx, quotaClient, normURL, apiKey, targetRaw.AuthIndex, accountID)
	} else if targetRaw.Provider == "xai" && targetRaw.AuthIndex != "" {
		targetFile.XaiDetail = fetchXaiRealtimeQuota(quotaCtx, quotaClient, normURL, apiKey, targetRaw.AuthIndex)
	} else if targetRaw.Provider == "antigravity" && targetRaw.AuthIndex != "" {
		targetFile.AntigravityDetail = fetchAntigravityRealtimeQuota(quotaCtx, quotaClient, normURL, apiKey, targetRaw.AuthIndex, targetRaw.ProjectId)
	}

	return targetFile, nil
}

func fetchCodexRealtimeQuota(ctx context.Context, client *http.Client, baseURL, apiKey, authIndex, accountID string) *CodexQuotaDetailedInfo {
	header := map[string]string{
		"Authorization": "Bearer $TOKEN$",
		"Accept":        "application/json",
		"OpenAI-Beta":   "codex-1",
		"Originator":    "Codex Desktop",
	}
	if accountID != "" {
		header["chatgpt-account-id"] = accountID
	}

	statusCode, bodyBytes, err := executeCpaManagementApiCall(
		ctx, client, baseURL, apiKey, authIndex,
		http.MethodGet, "https://chatgpt.com/backend-api/wham/usage",
		header, "",
	)
	if err != nil || statusCode != http.StatusOK {
		return nil
	}

	type whamApiResp struct {
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
	}

	var parsed whamApiResp
	if err := common.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil
	}

	primUsed := parsed.RateLimit.PrimaryWindow.UsedPercent
	secUsed := parsed.RateLimit.SecondaryWindow.UsedPercent

	return &CodexQuotaDetailedInfo{
		PlanType: parsed.PlanType,
		PrimaryWindow: &CodexRateLimitWindowInfo{
			UsedPercent:      primUsed,
			RemainingPercent: int(math.Max(0, float64(100-primUsed))),
			ResetAfter:       formatSecondsToFriendly(parsed.RateLimit.PrimaryWindow.ResetAfterSeconds),
		},
		SecondaryWindow: &CodexRateLimitWindowInfo{
			UsedPercent:      secUsed,
			RemainingPercent: int(math.Max(0, float64(100-secUsed))),
			ResetAfter:       formatSecondsToFriendly(parsed.RateLimit.SecondaryWindow.ResetAfterSeconds),
		},
		AvailableResetCredits: parsed.RateLimitResetCredits.AvailableCount,
	}
}

func fetchXaiRealtimeQuota(ctx context.Context, client *http.Client, baseURL, apiKey, authIndex string) *XaiQuotaDetailedInfo {
	header := map[string]string{
		"Authorization":         "Bearer $TOKEN$",
		"x-xai-token-auth":      "xai-grok-cli",
		"x-grok-client-version": "0.2.91",
		"accept":                "*/*",
		"user-agent":            "grok-pager/0.2.91 grok-shell/0.2.91 (macos; aarch64)",
	}

	statusCode, bodyBytes, err := executeCpaManagementApiCall(
		ctx, client, baseURL, apiKey, authIndex,
		http.MethodGet, "https://cli-chat-proxy.grok.com/v1/billing?format=credits",
		header, "",
	)
	if err != nil || statusCode != http.StatusOK {
		return nil
	}

	type xaiConfig struct {
		CreditUsagePercent float64 `json:"creditUsagePercent"`
		ProductUsage       []struct {
			Product      string  `json:"product"`
			UsagePercent float64 `json:"usagePercent"`
		} `json:"productUsage"`
		CurrentPeriod struct {
			End string `json:"end"`
		} `json:"currentPeriod"`
	}

	var parsed struct {
		Config xaiConfig `json:"config"`
	}
	if err := common.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil
	}

	weeklyUsed := int(parsed.Config.CreditUsagePercent)
	grokBuildUsed := 0
	for _, p := range parsed.Config.ProductUsage {
		if strings.EqualFold(p.Product, "GrokBuild") {
			grokBuildUsed = int(p.UsagePercent)
		}
	}

	return &XaiQuotaDetailedInfo{
		WeeklyUsedPercent:      weeklyUsed,
		WeeklyRemainingPercent: int(math.Max(0, float64(100-weeklyUsed))),
		WeeklyResetAfter:       parsed.Config.CurrentPeriod.End,
		GrokBuildUsed:          grokBuildUsed,
		GrokBuildRemaining:     int(math.Max(0, float64(100-grokBuildUsed))),
		GrokChatUsed:           "--",
	}
}

func fetchAntigravityRealtimeQuota(ctx context.Context, client *http.Client, baseURL, apiKey, authIndex, projectID string) *AntigravityQuotaDetailedInfo {
	header := map[string]string{
		"Authorization": "Bearer $TOKEN$",
		"Content-Type":  "application/json",
		"User-Agent":    "antigravity/cli/1.0.13 (aidev_client; os_type=darwin; arch=arm64)",
	}

	// 1. Fetch Subscription Plan from loadCodeAssist
	plan := "free"
	subStatusCode, subBody, subErr := executeCpaManagementApiCall(
		ctx, client, baseURL, apiKey, authIndex,
		http.MethodPost, "https://daily-cloudcode-pa.googleapis.com/v1internal:loadCodeAssist",
		header, `{"metadata":{"ideType":"ANTIGRAVITY"}}`,
	)
	if subErr == nil && subStatusCode == http.StatusOK {
		var subPayload struct {
			CurrentTier struct {
				Id string `json:"id"`
			} `json:"currentTier"`
			PaidTier struct {
				Id string `json:"id"`
			} `json:"paidTier"`
		}
		if err := common.Unmarshal(subBody, &subPayload); err == nil {
			tierID := subPayload.PaidTier.Id
			if tierID == "" {
				tierID = subPayload.CurrentTier.Id
			}
			switch tierID {
			case "g1-pro-tier":
				plan = "pro"
			case "g1-ultra-tier":
				plan = "ultra"
			case "g1-ultra-lite-tier":
				plan = "ultra-lite"
			default:
				if tierID != "" {
					plan = tierID
				}
			}
		}
	}

	// 2. Fetch Quota Summary with project ID
	if projectID == "" {
		return &AntigravityQuotaDetailedInfo{Plan: plan}
	}

	projJSON, _ := common.Marshal(projectID)
	reqData := fmt.Sprintf(`{"project":%s}`, string(projJSON))
	quotaURLs := []string{
		"https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
		"https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
	}

	var quotaBody []byte
	for _, u := range quotaURLs {
		sc, b, err := executeCpaManagementApiCall(ctx, client, baseURL, apiKey, authIndex, http.MethodPost, u, header, reqData)
		if err == nil && sc == http.StatusOK && len(b) > 0 {
			quotaBody = b
			break
		}
	}
	if len(quotaBody) == 0 {
		return &AntigravityQuotaDetailedInfo{Plan: plan}
	}

	type summaryPayload struct {
		Groups []struct {
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Buckets     []struct {
				BucketID          string  `json:"bucketId"`
				DisplayName       string  `json:"displayName"`
				Window            string  `json:"window"`
				ResetTime         string  `json:"resetTime"`
				RemainingFraction float64 `json:"remainingFraction"`
				Description       string  `json:"description"`
			} `json:"buckets"`
		} `json:"groups"`
	}

	var parsed summaryPayload
	if err := common.Unmarshal(quotaBody, &parsed); err != nil {
		return &AntigravityQuotaDetailedInfo{Plan: plan}
	}

	var groups []AntigravityQuotaGroup
	for _, g := range parsed.Groups {
		grp := AntigravityQuotaGroup{
			ID:          strings.ToLower(strings.ReplaceAll(g.DisplayName, " ", "-")),
			Label:       g.DisplayName,
			Description: g.Description,
		}
		for _, b := range g.Buckets {
			remPercent := int(math.Round(b.RemainingFraction * 100))
			if remPercent < 0 {
				remPercent = 0
			} else if remPercent > 100 {
				remPercent = 100
			}
			resetFormatted := b.ResetTime
			if t, err := time.Parse(time.RFC3339, b.ResetTime); err == nil {
				dur := time.Until(t)
				if dur > 0 {
					resetFormatted = formatSecondsToFriendly(int(dur.Seconds()))
				}
			}

			grp.Buckets = append(grp.Buckets, AntigravityQuotaBucket{
				ID:               b.BucketID,
				Label:            b.DisplayName,
				Window:           b.Window,
				RemainingPercent: remPercent,
				ResetAfter:       resetFormatted,
				Description:      b.Description,
			})
		}
		groups = append(groups, grp)
	}

	return &AntigravityQuotaDetailedInfo{
		Plan:   plan,
		Groups: groups,
	}
}

// Helper to execute CPA management api-call and unwrap response body whether it is an object or JSON string
func executeCpaManagementApiCall(ctx context.Context, client *http.Client, baseURL, apiKey, authIndex, method, targetURL string, headers map[string]string, data string) (int, []byte, error) {
	callURL := strings.TrimRight(baseURL, "/") + "/v0/management/api-call"
	reqPayload := map[string]any{
		"authIndex": authIndex,
		"method":    method,
		"url":       targetURL,
		"header":    headers,
	}
	if data != "" {
		reqPayload["data"] = data
	}
	bodyBytes, err := common.Marshal(reqPayload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var rawEnvelope struct {
		StatusCode int             `json:"status_code"`
		Body       json.RawMessage `json:"body"`
	}
	if err := common.Unmarshal(respBytes, &rawEnvelope); err != nil {
		return resp.StatusCode, nil, err
	}

	bodyData := rawEnvelope.Body
	var bodyStr string
	if err := common.Unmarshal(bodyData, &bodyStr); err == nil {
		return rawEnvelope.StatusCode, []byte(bodyStr), nil
	}
	return rawEnvelope.StatusCode, bodyData, nil
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
