package controller

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

type CpaNodeDTO struct {
	Id               int                           `json:"id"`
	Name             string                        `json:"name"`
	BaseUrl          string                        `json:"base_url"`
	NormalizedUrl    string                        `json:"normalized_url"`
	HasApiKey        bool                          `json:"has_api_key"`
	Status           int                           `json:"status"`
	Weight           int                           `json:"weight"`
	Description      string                        `json:"description"`
	CreatedTime      int64                         `json:"created_time"`
	UpdatedTime      int64                         `json:"updated_time"`
	IsOnline         bool                          `json:"is_online"`
	Latency          int64                         `json:"latency"`
	HttpStatus       int                           `json:"http_status"`
	Version          string                        `json:"version"`
	ModelCount       int                           `json:"model_count"`
	Models           string                        `json:"models"`
	AuthFilesCount   int                           `json:"auth_files_count"`
	AuthFilesSummary []*service.CpaAuthFileInfo    `json:"auth_files_summary,omitempty"`
	LastError        string                        `json:"last_error"`
	LastCheckAt      int64                         `json:"last_check_at"`
	ChannelCount     int                           `json:"channel_count"`
	Channels         []*service.CpaNodeChannelInfo `json:"channels,omitempty"`
	Usage            *service.CpaNodeUsage         `json:"usage,omitempty"`
}

type CreateOrUpdateCpaNodeRequest struct {
	Name        string `json:"name"`
	BaseUrl     string `json:"base_url"`
	ApiKey      string `json:"api_key"`
	Status      int    `json:"status"`
	Weight      int    `json:"weight"`
	Description string `json:"description"`
}

func GetAllCpaNodes(c *gin.Context) {
	nodes, err := model.GetAllCpaNodes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	window := c.DefaultQuery("window", "today")
	fresh := c.Query("fresh") == "1"

	var dtos []*CpaNodeDTO
	var totalRequests int64
	var totalQuota int64
	onlineCount := 0

	for _, n := range nodes {
		dto := toCpaNodeDTO(n, false)
		channels, err := service.GetChannelsForCpaNode(n)
		if err == nil {
			dto.ChannelCount = len(channels)
			dto.Channels = channels
		}
		usage, err := service.GetCpaNodeUsage(n, window, fresh)
		if err == nil && usage != nil {
			dto.Usage = usage
			totalRequests += usage.Requests
			totalQuota += usage.Quota
		}
		if dto.IsOnline {
			onlineCount++
		}
		dtos = append(dtos, dto)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": dtos,
			"summary": gin.H{
				"total":          len(nodes),
				"online":         onlineCount,
				"total_requests": totalRequests,
				"total_quota":    totalQuota,
			},
		},
	})
}

func GetCpaNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的节点 ID"})
		return
	}
	node, err := model.GetCpaNodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点不存在: " + err.Error()})
		return
	}

	dto := toCpaNodeDTO(node, true)
	channels, _ := service.GetChannelsForCpaNode(node)
	dto.ChannelCount = len(channels)
	dto.Channels = channels

	window := c.DefaultQuery("window", "today")
	fresh := c.Query("fresh") == "1"
	usage, _ := service.GetCpaNodeUsage(node, window, fresh)
	dto.Usage = usage

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dto,
	})
}

func CreateCpaNode(c *gin.Context) {
	var req CreateOrUpdateCpaNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数解析失败: " + err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点名称不能为空"})
		return
	}
	baseURL := strings.TrimSpace(req.BaseUrl)
	if baseURL == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点 BaseURL 不能为空"})
		return
	}

	node := &model.CpaNode{
		Name:        name,
		BaseUrl:     baseURL,
		ApiKey:      strings.TrimSpace(req.ApiKey),
		Status:      req.Status,
		Weight:      req.Weight,
		Description: strings.TrimSpace(req.Description),
	}
	if node.Status == 0 {
		node.Status = model.CpaNodeStatusEnabled
	}

	if err := node.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建节点失败: " + err.Error()})
		return
	}

	// Immediate background probe
	go func() {
		defer func() {
			_ = recover()
		}()
		_, _ = service.ProbeCpaNode(context.Background(), node)
	}()

	recordManageAudit(c, "cpa_node.create", map[string]any{
		"id":   node.Id,
		"name": node.Name,
		"url":  node.NormalizedUrl,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    toCpaNodeDTO(node, false),
	})
}

func UpdateCpaNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的节点 ID"})
		return
	}
	var req CreateOrUpdateCpaNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数解析失败: " + err.Error()})
		return
	}

	node, err := model.GetCpaNodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点不存在: " + err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name != "" {
		node.Name = name
	}
	baseURL := strings.TrimSpace(req.BaseUrl)
	if baseURL != "" {
		node.BaseUrl = baseURL
	}
	if req.Status != 0 {
		node.Status = req.Status
	}
	node.Weight = req.Weight
	node.Description = strings.TrimSpace(req.Description)

	updateKey := false
	if strings.TrimSpace(req.ApiKey) != "" {
		node.ApiKey = strings.TrimSpace(req.ApiKey)
		updateKey = true
	}

	if err := node.Update(updateKey); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "更新节点失败: " + err.Error()})
		return
	}

	recordManageAudit(c, "cpa_node.update", map[string]any{
		"id":   node.Id,
		"name": node.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    toCpaNodeDTO(node, false),
	})
}

func DeleteCpaNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的节点 ID"})
		return
	}
	node, err := model.GetCpaNodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点不存在: " + err.Error()})
		return
	}

	if err := node.Delete(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "删除失败: " + err.Error()})
		return
	}

	recordManageAudit(c, "cpa_node.delete", map[string]any{
		"id":   id,
		"name": node.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

func ProbeCpaNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的节点 ID"})
		return
	}
	node, err := model.GetCpaNodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点不存在: " + err.Error()})
		return
	}

	res, err := service.ProbeCpaNode(c.Request.Context(), node)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "探测异常: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    res,
	})
}

type ResetCodexQuotaRequest struct {
	AuthFileId string `json:"auth_file_id"`
	Confirm    bool   `json:"confirm"`
}

func ResetCodexCredentialQuota(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的节点 ID"})
		return
	}
	node, err := model.GetCpaNodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点不存在: " + err.Error()})
		return
	}

	var req ResetCodexQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误: " + err.Error()})
		return
	}

	if !req.Confirm {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "必须确认消耗重置额度卡"})
		return
	}

	res, err := service.ResetCodexAuthFileQuota(c.Request.Context(), node, req.AuthFileId)
	if err != nil {
		recordManageAudit(c, "cpa_node.codex_reset_credits", map[string]any{
			"node_id":      node.Id,
			"auth_file_id": req.AuthFileId,
			"success":      false,
			"error":        err.Error(),
		})
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "重置失败: " + err.Error()})
		return
	}

	recordManageAudit(c, "cpa_node.codex_reset_credits", map[string]any{
		"node_id":      node.Id,
		"auth_file_id": req.AuthFileId,
		"success":      true,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "重置成功",
		"data":    res,
	})
}

func RefreshSingleCpaCredentialQuota(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的节点 ID"})
		return
	}
	node, err := model.GetCpaNodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "节点不存在: " + err.Error()})
		return
	}

	authFileId := c.Query("auth_file_id")
	if authFileId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "auth_file_id 不能为空"})
		return
	}

	res, err := service.RefreshSingleAuthFileQuota(c.Request.Context(), node, authFileId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "刷新失败: " + err.Error()})
		return
	}

	// Update node's cached auth summary with the freshly refreshed credential so the list reflects it immediately
	if node.AuthFilesSummary != "" && res != nil {
		var list []*service.CpaAuthFileInfo
		if err := common.UnmarshalJsonStr(node.AuthFilesSummary, &list); err == nil {
			updated := false
			for i, item := range list {
				if item.Id == res.Id || item.Name == res.Name {
					list[i] = res
					updated = true
					break
				}
			}
			if updated {
				if b, err := common.Marshal(list); err == nil {
					_ = node.UpdateProbeSnapshotWithAuth(node.IsOnline, node.Latency, node.HttpStatus, node.Version, node.Models, node.LastError, string(b))
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "刷新成功",
		"data":    res,
	})
}

func toCpaNodeDTO(node *model.CpaNode, includeDetails bool) *CpaNodeDTO {
	if node == nil {
		return nil
	}
	dto := &CpaNodeDTO{
		Id:            node.Id,
		Name:          node.Name,
		BaseUrl:       node.BaseUrl,
		NormalizedUrl: node.NormalizedUrl,
		HasApiKey:     strings.TrimSpace(node.ApiKey) != "",
		Status:        node.Status,
		Weight:        node.Weight,
		Description:   node.Description,
		CreatedTime:   node.CreatedTime,
		UpdatedTime:   node.UpdatedTime,
		IsOnline:      node.IsOnline,
		Latency:       node.Latency,
		HttpStatus:    node.HttpStatus,
		Version:       node.Version,
		ModelCount:    node.ModelCount,
		Models:        node.Models,
		LastError:     node.LastError,
		LastCheckAt:   node.LastCheckAt,
	}

	if node.AuthFilesSummary != "" {
		var authList []*service.CpaAuthFileInfo
		if err := common.UnmarshalJsonStr(node.AuthFilesSummary, &authList); err == nil {
			dto.AuthFilesCount = len(authList)
			dto.AuthFilesSummary = authList
		}
	}

	return dto
}
