package router

import (
	"net/http"

	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerCpaNodeRoutes(apiRouter *gin.RouterGroup) {
	cpaRoute := apiRouter.Group("/cpa-node")
	cpaRoute.Use(middleware.AdminAuth())

	for _, route := range cpaPermissionRoutes {
		cpaRoute.Handle(route.method, route.path,
			middleware.RequirePermission(route.permission),
			route.handler,
		)
	}
}

var cpaPermissionRoutes = []permissionRoute{
	{method: http.MethodGet, path: "/", permission: authz.ChannelRead, handler: controller.GetAllCpaNodes},
	{method: http.MethodGet, path: "/:id", permission: authz.ChannelRead, handler: controller.GetCpaNode},
	{method: http.MethodPost, path: "/", permission: authz.ChannelSensitiveWrite, handler: controller.CreateCpaNode},
	{method: http.MethodPut, path: "/:id", permission: authz.ChannelSensitiveWrite, handler: controller.UpdateCpaNode},
	{method: http.MethodDelete, path: "/:id", permission: authz.ChannelSensitiveWrite, handler: controller.DeleteCpaNode},
	{method: http.MethodPost, path: "/:id/test", permission: authz.ChannelOperate, handler: controller.ProbeCpaNode},
	{method: http.MethodPost, path: "/:id/reset-codex-quota", permission: authz.ChannelOperate, handler: controller.ResetCodexCredentialQuota},
	{method: http.MethodPost, path: "/:id/refresh-credential-quota", permission: authz.ChannelOperate, handler: controller.RefreshSingleCpaCredentialQuota},
}
