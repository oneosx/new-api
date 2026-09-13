package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	codexResetLockMap = make(map[string]struct{})
	codexResetMu      sync.Mutex
)

type CodexConsumeResetResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func ResetCodexAuthFileQuota(ctx context.Context, node *model.CpaNode, authFileID string) (*CodexConsumeResetResult, error) {
	if node == nil {
		return nil, fmt.Errorf("cpa node is nil")
	}
	authFileID = strings.TrimSpace(authFileID)
	if authFileID == "" {
		return nil, fmt.Errorf("auth_file_id is required")
	}

	lockKey := fmt.Sprintf("%d:%s", node.Id, authFileID)
	codexResetMu.Lock()
	if _, busy := codexResetLockMap[lockKey]; busy {
		codexResetMu.Unlock()
		return nil, fmt.Errorf("a reset operation is already in progress for this credential")
	}
	codexResetLockMap[lockKey] = struct{}{}
	codexResetMu.Unlock()

	defer func() {
		codexResetMu.Lock()
		delete(codexResetLockMap, lockKey)
		codexResetMu.Unlock()
	}()

	apiKey := strings.TrimSpace(node.ApiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("node api_key is not configured")
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

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Get Auth files to verify credential exists and find auth_index
	authReqURL := normURL + "/v0/management/auth-files"
	authReq, err := http.NewRequestWithContext(ctx, http.MethodGet, authReqURL, nil)
	if err != nil {
		return nil, err
	}
	authReq.Header.Set("Authorization", "Bearer "+apiKey)
	authReq.Header.Set("Accept", "application/json")

	authResp, err := client.Do(authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch node auth files: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query auth files from node: HTTP %d", authResp.StatusCode)
	}

	authBody, _ := io.ReadAll(io.LimitReader(authResp.Body, 2<<20))
	var rawAuth rawAuthFilesResponse
	if err := common.Unmarshal(authBody, &rawAuth); err != nil {
		return nil, fmt.Errorf("invalid auth files response: %w", err)
	}

	var targetAuthIndex string
	var chatgptAccountID string
	var isCodex bool
	for _, f := range rawAuth.Files {
		if f.Id == authFileID || f.Name == authFileID {
			if strings.ToLower(f.Provider) == "codex" || strings.ToLower(f.Type) == "codex" {
				isCodex = true
				targetAuthIndex = f.AuthIndex
				chatgptAccountID = f.Account
				break
			}
		}
	}

	if !isCodex || targetAuthIndex == "" {
		return nil, fmt.Errorf("credential %q is not a valid Codex account on this node", authFileID)
	}

	// 2. Precheck available reset credits
	precheckHeader := map[string]string{
		"Authorization": "Bearer $TOKEN$",
		"Accept":        "application/json",
		"OpenAI-Beta":   "codex-1",
		"Originator":    "Codex Desktop",
	}
	if chatgptAccountID != "" {
		precheckHeader["chatgpt-account-id"] = chatgptAccountID
	}

	precheckPayload := map[string]any{
		"authIndex": targetAuthIndex,
		"method":    "GET",
		"url":       "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits",
		"header":    precheckHeader,
	}
	precheckBytes, _ := common.Marshal(precheckPayload)

	apiCallURL := normURL + "/v0/management/api-call"
	callReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiCallURL, strings.NewReader(string(precheckBytes)))
	if err != nil {
		return nil, err
	}
	callReq.Header.Set("Authorization", "Bearer "+apiKey)
	callReq.Header.Set("Content-Type", "application/json")

	callResp, err := client.Do(callReq)
	if err != nil {
		return nil, fmt.Errorf("failed to precheck reset credits: %w", err)
	}
	defer callResp.Body.Close()

	if callResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reset-credit precheck request failed: HTTP %d", callResp.StatusCode)
	}
	callRespBytes, err := io.ReadAll(io.LimitReader(callResp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read reset-credit precheck: %w", err)
	}
	type resetCreditResp struct {
		AvailableCount int `json:"available_count"`
	}
	var precheckApiRes struct {
		StatusCode int             `json:"status_code"`
		Body       resetCreditResp `json:"body"`
	}
	if err := common.Unmarshal(callRespBytes, &precheckApiRes); err != nil {
		return nil, fmt.Errorf("invalid reset-credit precheck response: %w", err)
	}
	if precheckApiRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reset-credit precheck upstream returned HTTP %d", precheckApiRes.StatusCode)
	}
	if precheckApiRes.Body.AvailableCount <= 0 {
		return nil, fmt.Errorf("no rate limit reset credits available for this account")
	}

	// 3. Consume 1 reset credit
	redeemRequestId := common.GetUUID()
	consumePayload := map[string]any{
		"authIndex": targetAuthIndex,
		"method":    "POST",
		"url":       "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits/consume",
		"header":    precheckHeader,
		"data":      fmt.Sprintf(`{"redeem_request_id":%q}`, redeemRequestId),
	}
	consumeBytes, _ := common.Marshal(consumePayload)

	consumeReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiCallURL, strings.NewReader(string(consumeBytes)))
	if err != nil {
		return nil, err
	}
	consumeReq.Header.Set("Authorization", "Bearer "+apiKey)
	consumeReq.Header.Set("Content-Type", "application/json")

	consumeResp, err := client.Do(consumeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call consume api: %w", err)
	}
	defer consumeResp.Body.Close()

	consumeRespBytes, _ := io.ReadAll(io.LimitReader(consumeResp.Body, 1<<20))
	var consumeApiRes struct {
		StatusCode int `json:"status_code"`
		Body       any `json:"body"`
	}
	_ = common.Unmarshal(consumeRespBytes, &consumeApiRes)

	if consumeApiRes.StatusCode < 200 || consumeApiRes.StatusCode >= 300 {
		return nil, fmt.Errorf("consume upstream returned error: HTTP %d", consumeApiRes.StatusCode)
	}

	// Trigger immediate background probe to refresh signals
	go func() {
		defer func() { _ = recover() }()
		_, _ = ProbeCpaNode(context.Background(), node)
	}()

	return &CodexConsumeResetResult{
		Success: true,
		Message: "Rate limit reset credit consumed successfully",
	}, nil
}
