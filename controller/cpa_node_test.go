package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func TestCpaNodeLifecycleAndProbe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock CPA HTTP server exposing /v1/models
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-cpa-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}

		w.Header().Set("X-CPA-VERSION", "v7.2.156")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"list","data":[{"id":"gemini-3.8-flash-high"},{"id":"claude-sonnet-4-6"}]}`))
	}))
	defer server.Close()

	// 1. Test NormalizeCpaBaseURL
	norm, err := model.NormalizeCpaBaseURL(server.URL + "/v1///")
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/v1", norm)

	normRoot, err := model.NormalizeCpaBaseURL(server.URL + "/")
	require.NoError(t, err)
	assert.Equal(t, server.URL, normRoot)

	// 2. Test Probe with invalid key
	nodeInvalid := &model.CpaNode{
		Id:      1001,
		Name:    "Test-CPA-Invalid",
		BaseUrl: server.URL,
		ApiKey:  "bad-token",
	}
	resInvalid, err := service.ProbeCpaNode(context.Background(), nodeInvalid)
	require.NoError(t, err)
	assert.False(t, resInvalid.Online)
	assert.Equal(t, http.StatusUnauthorized, resInvalid.HttpStatus)
	assert.Contains(t, resInvalid.Error, "HTTP 401")

	// 3. Test Probe with valid key
	nodeValid := &model.CpaNode{
		Id:      1002,
		Name:    "Test-CPA-Valid",
		BaseUrl: server.URL,
		ApiKey:  "valid-cpa-token",
	}
	resValid, err := service.ProbeCpaNode(context.Background(), nodeValid)
	require.NoError(t, err)
	assert.True(t, resValid.Online)
	assert.Equal(t, 2, resValid.ModelCount)
	assert.Equal(t, "v7.2.156", resValid.Version)
	assert.Contains(t, resValid.Models, "gemini-3.8-flash-high")
	assert.Contains(t, resValid.Models, "claude-sonnet-4-6")
}
