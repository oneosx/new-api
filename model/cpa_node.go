package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	CpaNodeStatusEnabled  = 1
	CpaNodeStatusDisabled = 2
)

type CpaNode struct {
	Id            int    `json:"id" gorm:"primaryKey"`
	Name          string `json:"name" gorm:"type:varchar(64);not null;uniqueIndex"`
	BaseUrl       string `json:"base_url" gorm:"type:varchar(255);not null;uniqueIndex"`
	NormalizedUrl string `json:"-" gorm:"column:normalized_url;type:varchar(255);not null;uniqueIndex"`
	ApiKey        string `json:"-" gorm:"type:varchar(255)"`
	Status        int    `json:"status" gorm:"default:1;index"` // 1: enabled, 2: disabled
	Weight        int    `json:"weight" gorm:"default:0"`
	Description   string `json:"description" gorm:"type:varchar(255)"`
	CreatedTime   int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime   int64  `json:"updated_time" gorm:"bigint"`

	// Probe snapshot fields
	IsOnline    bool   `json:"is_online" gorm:"default:false"`
	Latency     int64  `json:"latency" gorm:"bigint;default:0"`
	HttpStatus  int    `json:"http_status" gorm:"default:0"`
	Version     string `json:"version" gorm:"type:varchar(64);default:''"`
	ModelCount  int    `json:"model_count" gorm:"default:0"`
	Models      string `json:"models" gorm:"type:text"`
	LastError   string `json:"last_error" gorm:"type:varchar(255);default:''"`
	AuthFilesSummary string `json:"auth_files_summary" gorm:"type:text"`
	LastCheckAt int64  `json:"last_check_at" gorm:"bigint;default:0;index"`
}

func NormalizeCpaBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("base_url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("base_url must start with http:// or https://")
	}
	if u.Host == "" {
		return "", errors.New("base_url missing host")
	}
	u.Scheme = scheme
	u.Host = strings.ToLower(u.Host)
	u.User = nil
	u.Fragment = ""
	u.RawQuery = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

func GetAllCpaNodes() ([]*CpaNode, error) {
	var nodes []*CpaNode
	err := DB.Order("weight desc, id desc").Find(&nodes).Error
	return nodes, err
}

func GetEnabledCpaNodes() ([]*CpaNode, error) {
	var nodes []*CpaNode
	err := DB.Where("status = ?", CpaNodeStatusEnabled).Order("weight desc, id desc").Find(&nodes).Error
	return nodes, err
}

func GetCpaNodeById(id int) (*CpaNode, error) {
	var node CpaNode
	err := DB.First(&node, id).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (node *CpaNode) Insert() error {
	norm, err := NormalizeCpaBaseURL(node.BaseUrl)
	if err != nil {
		return err
	}
	node.NormalizedUrl = norm
	now := time.Now().Unix()
	node.CreatedTime = now
	node.UpdatedTime = now
	if node.Status == 0 {
		node.Status = CpaNodeStatusEnabled
	}
	return DB.Create(node).Error
}

func (node *CpaNode) Update(updateKey bool) error {
	norm, err := NormalizeCpaBaseURL(node.BaseUrl)
	if err != nil {
		return err
	}
	node.NormalizedUrl = norm
	node.UpdatedTime = time.Now().Unix()

	return DB.Transaction(func(tx *gorm.DB) error {
		lockedNode, err := lockCpaNodeForUpdate(tx, node.Id)
		if err != nil {
			return err
		}
		if lockedNode == nil {
			return errors.New("cpa node not found")
		}

		updates := map[string]any{
			"name":           node.Name,
			"base_url":       node.BaseUrl,
			"normalized_url": node.NormalizedUrl,
			"status":         node.Status,
			"weight":         node.Weight,
			"description":    node.Description,
			"updated_time":   node.UpdatedTime,
		}
		if updateKey {
			updates["api_key"] = node.ApiKey
		}
		return tx.Model(&CpaNode{}).Where("id = ?", node.Id).Updates(updates).Error
	})
}

func (node *CpaNode) Delete() error {
	if node.Id == 0 {
		return errors.New("id is invalid")
	}
	return DB.Delete(&CpaNode{}, node.Id).Error
}

func (node *CpaNode) UpdateProbeSnapshot(isOnline bool, latency int64, httpStatus int, version, models, lastError string) error {
	return node.UpdateProbeSnapshotWithAuth(isOnline, latency, httpStatus, version, models, lastError, "")
}

func (node *CpaNode) UpdateProbeSnapshotWithAuth(isOnline bool, latency int64, httpStatus int, version, models, lastError, authSummary string) error {
	modelsTrimmed := strings.TrimSpace(models)
	modelCount := 0
	if modelsTrimmed != "" {
		parts := strings.Split(modelsTrimmed, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				modelCount++
			}
		}
	}
	node.IsOnline = isOnline
	node.Latency = latency
	node.HttpStatus = httpStatus
	node.Version = version
	node.ModelCount = modelCount
	node.Models = modelsTrimmed
	node.LastError = lastError
	node.AuthFilesSummary = authSummary
	node.LastCheckAt = time.Now().Unix()
	if DB == nil {
		return nil
	}

	updates := map[string]any{
		"is_online":          isOnline,
		"latency":            latency,
		"http_status":        httpStatus,
		"version":            version,
		"model_count":        modelCount,
		"models":             modelsTrimmed,
		"last_error":         lastError,
		"auth_files_summary": authSummary,
		"last_check_at":      time.Now().Unix(),
	}
	return DB.Model(&CpaNode{}).Where("id = ?", node.Id).Updates(updates).Error
}

func lockCpaNodeForUpdate(tx *gorm.DB, id int) (*CpaNode, error) {
	var node CpaNode
	q := lockForUpdate(tx.Where("id = ?", id))
	if err := q.First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}
