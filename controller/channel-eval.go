package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service/eval"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// ChannelEvalRequest represents the parameters for a channel evaluation run
type ChannelEvalRequest struct {
	Model     string         `json:"model"`
	ProbeType eval.ProbeKind `json:"probe_type"`
	Randomize bool           `json:"randomize"`
	Thinking  bool           `json:"thinking"`
}

// ChannelEvalResult is the response payload for a single channel evaluation
type ChannelEvalResult struct {
	Success         bool                    `json:"success"`
	ChannelID       int                     `json:"channel_id"`
	ChannelName     string                  `json:"channel_name"`
	ChannelType     int                     `json:"channel_type"`
	Model           string                  `json:"model"`
	ProbeType       eval.ProbeKind          `json:"probe_type"`
	ProbeName       string                  `json:"probe_name"`
	Category        string                  `json:"category"`
	Prompt          string                  `json:"prompt"`
	RawOutput       string                  `json:"raw_output"`
	ReasoningOutput string                  `json:"reasoning_output,omitempty"`
	ReasoningTokens int                     `json:"reasoning_tokens"`
	PromptTokens    int                     `json:"prompt_tokens"`
	TotalTokens     int                     `json:"total_tokens"`
	DurationMs      int64                   `json:"duration_ms"`
	Analysis        eval.EvalAnalysisResult `json:"analysis"`
	ErrorMessage    string                  `json:"error_message,omitempty"`
}

// EvalChannel handles POST /api/channel/:id/eval
func EvalChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req ChannelEvalRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		common.ApiError(c, err)
		return
	}

	testUserID, err := resolveChannelTestUserID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	result := runChannelEval(c.Request.Context(), channel, testUserID, req)
	c.JSON(http.StatusOK, gin.H{
		"success": result.Success,
		"data":    result,
	})
}

// MatrixEvalRequest represents parameters for cross-channel evaluation
type MatrixEvalRequest struct {
	Model     string         `json:"model" binding:"required"`
	ProbeType eval.ProbeKind `json:"probe_type"`
	Randomize bool           `json:"randomize"`
	Thinking  bool           `json:"thinking"`
}

// EvalMatrix handles POST /api/channel/matrix-eval via SSE streaming to prevent Cloudflare 524 timeouts
func EvalMatrix(c *gin.Context) {
	var req MatrixEvalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	testUserID, err := resolveChannelTestUserID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 1. Find all enabled channels supporting this model
	allChannels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var channels []*model.Channel
	targetModelLower := strings.ToLower(strings.TrimSpace(req.Model))
	for _, ch := range allChannels {
		if ch.Status != common.ChannelStatusEnabled {
			continue
		}
		for _, m := range ch.GetModels() {
			if strings.ToLower(strings.TrimSpace(m)) == targetModelLower {
				channels = append(channels, ch)
				break
			}
		}
	}

	probeDef := eval.GetProbeDefinition(req.ProbeType, req.Randomize)

	// Set SSE Headers immediately to maintain persistent HTTP connection with Cloudflare
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	sendSSE := func(eventType string, data any) {
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
		c.Writer.Flush()
	}

	sendSSE("init", gin.H{
		"model": req.Model,
		"probe": probeDef,
		"total": len(channels),
	})

	if len(channels) == 0 {
		sendSSE("done", gin.H{"total": 0})
		return
	}

	// Periodic heartbeat to prevent Cloudflare 524 timeout (100 seconds threshold)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				fmt.Fprintf(c.Writer, ": ping\n\n")
				c.Writer.Flush()
			}
		}
	}()

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, ch := range channels {
		wg.Add(1)
		go func(idx int, targetCh *model.Channel) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			evalReq := ChannelEvalRequest{
				Model:     req.Model,
				ProbeType: req.ProbeType,
				Randomize: false,
				Thinking:  req.Thinking,
			}
			res := runChannelEvalWithProbe(ctx, targetCh, testUserID, evalReq, probeDef)

			mu.Lock()
			sendSSE("result", res)
			mu.Unlock()
		}(i, ch)
	}

	wg.Wait()
	sendSSE("done", gin.H{"total": len(channels)})
}

func runChannelEval(ctx context.Context, channel *model.Channel, testUserID int, req ChannelEvalRequest) ChannelEvalResult {
	if req.ProbeType == "" {
		req.ProbeType = eval.ProbeKindCandy
	}
	probeDef := eval.GetProbeDefinition(req.ProbeType, req.Randomize)
	return runChannelEvalWithProbe(ctx, channel, testUserID, req, probeDef)
}

func runChannelEvalWithProbe(ctx context.Context, channel *model.Channel, testUserID int, req ChannelEvalRequest, probeDef eval.ProbeDefinition) ChannelEvalResult {
	targetModel := strings.TrimSpace(req.Model)
	if targetModel == "" {
		if channel.TestModel != nil && *channel.TestModel != "" {
			targetModel = strings.TrimSpace(*channel.TestModel)
		} else {
			models := channel.GetModels()
			if len(models) > 0 {
				targetModel = strings.TrimSpace(models[0])
			}
			if targetModel == "" {
				targetModel = "gpt-4o"
			}
		}
	}

	result := ChannelEvalResult{
		ChannelID:   channel.Id,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Model:       targetModel,
		ProbeType:   probeDef.Kind,
		ProbeName:   probeDef.Name,
		Category:    probeDef.Category,
		Prompt:      probeDef.Prompt,
	}

	// Build chat request with the probe's prompt.
	// Enable Stream mode so upstream Cloudflare/reverse proxies immediately receive chunks,
	// avoiding the 100s timeout (HTTP 524) while models think for minutes.
	chatReq := &dto.GeneralOpenAIRequest{
		Model:  targetModel,
		Stream: lo.ToPtr(true),
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: probeDef.Prompt,
			},
		},
		StreamOptions: &dto.StreamOptions{
			IncludeUsage: true,
		},
	}

	// Allocate generous token budget for 3D/SVG or complex reasoning.
	// o-series reasoning models reject `max_tokens`; they need `max_completion_tokens`.
	maxOut := uint(4096)
	if probeDef.Category == "3d" {
		maxOut = 16384
	}
	if dto.IsOpenAIReasoningOModel(targetModel) {
		chatReq.MaxCompletionTokens = lo.ToPtr(maxOut)
	} else {
		chatReq.MaxTokens = lo.ToPtr(maxOut)
	}

	// Always enable high thinking/reasoning effort for evaluation
	chatReq.ReasoningEffort = "high"

	evalTimeout := 300 * time.Second
	evalCtx, evalCancel := context.WithTimeout(ctx, evalTimeout)
	defer evalCancel()

	// Prepare mock context for relay adaptation
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequestWithContext(evalCtx, http.MethodPost, "/v1/chat/completions", nil)
	ginCtx.Request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	ginCtx.Request.Header.Set("Accept", "text/event-stream, application/json, */*")

	cache, err := model.GetUserCache(testUserID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("GetUserCache failed: %v", err)
		return result
	}
	cache.WriteContext(ginCtx)
	ginCtx.Set("id", testUserID)
	ginCtx.Set("channel", channel.Type)
	ginCtx.Set("base_url", channel.GetBaseURL())
	group, _ := model.GetUserGroup(testUserID, false)
	ginCtx.Set("group", group)

	apiErr := middleware.SetupContextForSelectedChannel(ginCtx, channel, targetModel)
	if apiErr != nil {
		result.ErrorMessage = fmt.Sprintf("SetupContext failed: %v", apiErr)
		return result
	}

	info, err := relaycommon.GenRelayInfo(ginCtx, types.RelayFormatOpenAI, chatReq, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("GenRelayInfo failed: %v", err)
		return result
	}
	info.IsStream = true
	info.IsChannelTest = true
	info.InitChannelMeta(ginCtx)
	info.RelayMode = relayconstant.RelayModeChatCompletions
	info.OriginModelName = targetModel

	err = helper.ModelMappedHelper(ginCtx, info, chatReq)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("ModelMappedHelper failed: %v", err)
		return result
	}
	_ = helper.ApplyReasoningModelSuffix(ginCtx, info, chatReq)

	apiType, _ := common.ChannelType2APIType(channel.Type)
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		result.ErrorMessage = fmt.Sprintf("Unsupported api type: %d", apiType)
		return result
	}

	adaptor.Init(info)
	convertedReq, err := adaptor.ConvertOpenAIRequest(ginCtx, info, chatReq)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("ConvertOpenAIRequest failed: %v", err)
		return result
	}

	jsonData, err := common.Marshal(convertedReq)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Marshal request failed: %v", err)
		return result
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("ParamOverride failed: %v", err)
			return result
		}
	}

	startTime := time.Now()
	requestBody := bytes.NewBuffer(jsonData)
	ginCtx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))

	fullURL, _ := adaptor.GetRequestURL(info)
	common.SysError(fmt.Sprintf("[channel-eval] testing channel %d (%s), URL: %s, isStream: %v, relayMode: %d, bodyLen: %d, bodyPrefix: %s, reqHeaders: %+v",
		channel.Id, channel.Name, fullURL, info.IsStream, info.RelayMode, len(jsonData), string(jsonData[:min(100, len(jsonData))]), ginCtx.Request.Header))

	// Print the actual headers that will be sent
	testHeader := make(http.Header)
	_ = adaptor.SetupRequestHeader(ginCtx, &testHeader, info)
	common.SysError(fmt.Sprintf("[channel-eval] setupHeaders: %+v", testHeader))

	resp, err := adaptor.DoRequest(ginCtx, info, requestBody)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("DoRequest failed: %v", err)
		return result
	}

	httpResp, ok := resp.(*http.Response)
	if !ok || httpResp == nil {
		result.ErrorMessage = "Empty HTTP response from adaptor"
		return result
	}

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		result.ErrorMessage = fmt.Sprintf("Upstream returned HTTP %d: %s", httpResp.StatusCode, string(respBody))
		return result
	}

	usageObj, respErr := adaptor.DoResponse(ginCtx, httpResp, info)
	result.DurationMs = time.Since(startTime).Milliseconds()

	if respErr != nil {
		result.ErrorMessage = fmt.Sprintf("DoResponse failed: %v", respErr)
		return result
	}

	if usage, ok := usageObj.(*dto.Usage); ok && usage != nil {
		result.PromptTokens = usage.PromptTokens
		result.TotalTokens = usage.TotalTokens
		result.ReasoningTokens = usage.CompletionTokenDetails.ReasoningTokens
	}

	// Capture response body written into Recorder (handle both stream chunks and single JSON)
	respBytes := w.Body.Bytes()
	var outputBuilder strings.Builder
	var reasoningBuilder strings.Builder

	lines := bytes.Split(respBytes, []byte("\n"))
	isSSEStream := false
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("data: ")) {
			isSSEStream = true
			chunkData := bytes.TrimPrefix(trimmed, []byte("data: "))
			if bytes.Equal(chunkData, []byte("[DONE]")) {
				continue
			}
			var streamChunk dto.ChatCompletionsStreamResponse
			if err := json.Unmarshal(chunkData, &streamChunk); err == nil {
				if len(streamChunk.Choices) > 0 {
					delta := streamChunk.Choices[0].Delta
					outputBuilder.WriteString(delta.GetContentString())
					reasoningBuilder.WriteString(delta.GetReasoningContent())
				}
				if streamChunk.Usage != nil {
					if streamChunk.Usage.CompletionTokenDetails.ReasoningTokens > 0 {
						result.ReasoningTokens = streamChunk.Usage.CompletionTokenDetails.ReasoningTokens
					}
					if streamChunk.Usage.PromptTokens > 0 {
						result.PromptTokens = streamChunk.Usage.PromptTokens
						result.TotalTokens = streamChunk.Usage.TotalTokens
					}
				}
			}
		}
	}

	if isSSEStream {
		result.RawOutput = outputBuilder.String()
		result.ReasoningOutput = reasoningBuilder.String()
	} else {
		var parsedResp dto.OpenAITextResponse
		if err := json.Unmarshal(respBytes, &parsedResp); err == nil && len(parsedResp.Choices) > 0 {
			choice := parsedResp.Choices[0]
			if choice.Message.Content != nil {
				if strContent, ok := choice.Message.Content.(string); ok {
					result.RawOutput = strContent
				}
			}
			if choice.Message.ReasoningContent != nil {
				result.ReasoningOutput = *choice.Message.ReasoningContent
			} else if rc := choice.Message.GetReasoningContent(); rc != "" {
				result.ReasoningOutput = rc
			}
			if choice.ReasoningContent != nil && result.ReasoningOutput == "" {
				result.ReasoningOutput = *choice.ReasoningContent
			}
			if parsedResp.Usage.CompletionTokenDetails.ReasoningTokens > 0 {
				result.ReasoningTokens = parsedResp.Usage.CompletionTokenDetails.ReasoningTokens
			}
			if parsedResp.Usage.PromptTokens > 0 {
				result.PromptTokens = parsedResp.Usage.PromptTokens
				result.TotalTokens = parsedResp.Usage.TotalTokens
			}
		} else {
			result.RawOutput = string(respBytes)
		}
	}

	// Grade output
	analysis := eval.GradeProbeOutput(probeDef.Kind, result.RawOutput, probeDef.Expected)
	result.Analysis = analysis
	result.Success = analysis.Success

	return result
}
