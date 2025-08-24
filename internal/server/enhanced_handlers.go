package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pikachun/internal/service"
)

// EnhancedHandlers 增强的处理器
type EnhancedHandlers struct {
	enhancedCanalService *service.EnhancedCanalService
}

// NewEnhancedHandlers 创建增强处理器
func NewEnhancedHandlers(enhancedCanalService *service.EnhancedCanalService) *EnhancedHandlers {
	return &EnhancedHandlers{
		enhancedCanalService: enhancedCanalService,
	}
}

// getBinlogInfoHandler 获取binlog信息
func (h *EnhancedHandlers) getBinlogInfoHandler(c *gin.Context) {
	info, err := h.enhancedCanalService.GetBinlogInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取binlog信息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": info,
	})
}

// getEnhancedStatusHandler 获取增强状态信息
func (h *EnhancedHandlers) getEnhancedStatusHandler(c *gin.Context) {
	status := h.enhancedCanalService.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"data": status,
	})
}

// getPerformanceMetricsHandler 获取性能指标
func (h *EnhancedHandlers) getPerformanceMetricsHandler(c *gin.Context) {
	metrics := h.enhancedCanalService.GetPerformanceMetrics()

	c.JSON(http.StatusOK, gin.H{
		"data": metrics,
	})
}
