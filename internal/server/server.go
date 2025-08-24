package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pikachun/internal/config"
	"pikachun/internal/database"
	"pikachun/internal/service"
)

// Server HTTP服务器
type Server struct {
	config           *config.Config
	taskService      *service.TaskService
	canalService     service.CanalServiceInterface
	enhancedHandlers *EnhancedHandlers
	// enhancedCanalService *service.EnhancedCanalService
	router *gin.Engine
}

// CanalServiceAdapter Canal服务适配器
type CanalServiceAdapter struct {
	enhanced *service.EnhancedCanalService
}

// Start 启动服务
func (a *CanalServiceAdapter) Start(ctx context.Context) error {
	// 增强服务已经在main中启动，这里返回nil
	return a.enhanced.Start(ctx)
}

// Stop 停止服务
func (a *CanalServiceAdapter) Stop() error {
	// 增强服务会在main中停止，这里不需要操作
	return a.enhanced.Stop()
}

// Stop Instance 停止指定实例
func (a *CanalServiceAdapter) StopInstance(instanceID uint) error {
	return a.enhanced.StopInstance(instanceID)
}

// UpdateInstance 更新指定实例
func (a *CanalServiceAdapter) UpdateInstance(instanceID uint, task *database.Task) error {
	return a.enhanced.UpdateInstance(instanceID, task)
}

// CreateTask 创建任务
func (a *CanalServiceAdapter) CreateTask(task *database.Task) error {
	return a.enhanced.CreateTask(task)
}

// GetStatus 获取状态
func (a *CanalServiceAdapter) GetStatus() map[string]interface{} {
	return a.enhanced.GetStatus()
}

// New 创建服务器实例
// New 创建服务器实例
func New(cfg *config.Config, taskService *service.TaskService, canalService service.CanalServiceInterface) *Server {
	// 创建增强处理器
	var enhancedHandlers *EnhancedHandlers

	// 检查是否为直接的EnhancedCanalService
	if enhancedCanalService, ok := canalService.(*service.EnhancedCanalService); ok {
		fmt.Printf("DEBUG: Using direct EnhancedCanalService\n")
		enhancedHandlers = NewEnhancedHandlers(enhancedCanalService)
	} else if adapter, ok := canalService.(*CanalServiceAdapter); ok {
		// 检查是否为CanalServiceAdapter
		fmt.Printf("DEBUG: Using CanalServiceAdapter\n")
		enhancedHandlers = NewEnhancedHandlers(adapter.enhanced)
	} else {
		fmt.Printf("DEBUG: Using basic canal service, type: %T\n", canalService)
	}

	return &Server{
		config:           cfg,
		taskService:      taskService,
		canalService:     canalService,
		enhancedHandlers: enhancedHandlers,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.setupRouter()
	addr := s.config.Server.Host + ":" + s.config.Server.Port
	return s.router.Run(addr)
}

// setupRouter 设置路由
func (s *Server) setupRouter() {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.Default()

	// 静态文件服务
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("web/templates/*")

	// 首页
	s.router.GET("/", s.indexHandler)

	// API路由组
	api := s.router.Group("/api")
	{
		// 任务管理
		tasks := api.Group("/tasks")
		{
			tasks.GET("", s.getTasksHandler)
			tasks.POST("", s.createTaskHandler)
			tasks.GET("/:id", s.getTaskHandler)
			tasks.PUT("/:id", s.updateTaskHandler)
			tasks.DELETE("/:id", s.deleteTaskHandler)
		}

		// 事件日志
		api.GET("/logs", s.getEventLogsHandler)
		api.GET("/logs/:id", s.getEventLogHandler)

		// 系统状态
		api.GET("/status", s.getStatusHandler)

		// 增强功能 API
		api.GET("/metrics", s.getPerformanceMetricsHandler)
	}
}

// indexHandler 首页处理器
func (s *Server) indexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Pikachun - 数据库监听服务",
	})
}

// getTasksHandler 获取任务列表
func (s *Server) getTasksHandler(c *gin.Context) {
	page := 1
	pageSize := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := parseIntDefault(p, 1); err == nil {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := parseIntDefault(ps, 10); err == nil {
			pageSize = parsed
		}
	}

	tasks, total, err := s.taskService.GetTasks(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取任务列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"tasks":     tasks,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// createTaskHandler 创建任务
func (s *Server) createTaskHandler(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	task := req.ToTask()
	if err := s.taskService.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建任务失败: " + err.Error(),
		})
		return
	}

	// 启动Canal实例来监听binlog
	if err := s.canalService.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "启动Canal监听失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": task,
	})
}

// getTaskHandler 获取单个任务
func (s *Server) getTaskHandler(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的任务ID",
		})
		return
	}

	task, err := s.taskService.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": task,
	})
}

// updateTaskHandler 更新任务
func (s *Server) updateTaskHandler(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的任务ID",
		})
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	updates := req.ToTask()
	if err := s.taskService.UpdateTask(id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新任务失败: " + err.Error(),
		})
		return
	}

	// 日志记录
	fmt.Printf("Task %d updated, updating associated canal instance if exists", id)
	if s.canalService.UpdateInstance(id, updates) != nil {
		// 日错误志记录
		fmt.Printf("Error updating canal instance for updated task %d", id)
		// error
		fmt.Printf("Error updating canal instance for updated task: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新Canal任务失败: " + err.Error(),
		})
		return
	}

	//日志记录
	fmt.Printf("Canal instance for task %d updated", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "任务更新成功",
	})
}

// deleteTaskHandler 删除任务
func (s *Server) deleteTaskHandler(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的任务ID",
		})
		return
	}

	if err := s.taskService.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除任务失败: " + err.Error(),
		})
		return
	}

	// 日志记录
	fmt.Printf("Task %d deleted, removing associated canal instance if exists", id)
	if err := s.canalService.StopInstance(id); err != nil {
		// 错误日志
		fmt.Printf("Error stopping canal instance for deleted task %d: %s", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "停止Canal任务失败: " + err.Error(),
		})
		return
	}
	//日志记录
	fmt.Printf("Canal instance for task %d stopped", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "任务删除成功",
	})
}

// getEventLogsHandler 获取事件日志
func (s *Server) getEventLogsHandler(c *gin.Context) {
	page := 1
	pageSize := 20
	var taskID uint

	if p := c.Query("page"); p != "" {
		if parsed, err := parseIntDefault(p, 1); err == nil {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := parseIntDefault(ps, 20); err == nil {
			pageSize = parsed
		}
	}

	if tid := c.Query("task_id"); tid != "" {
		if parsed, err := parseUintDefault(tid, 0); err == nil {
			taskID = parsed
		}
	}

	logs, total, err := s.taskService.GetEventLogs(taskID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取事件日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// getEventLogHandler 获取单个事件日志
func (s *Server) getEventLogHandler(c *gin.Context) {
	id, err := parseUintDefault(c.Param("id"), 0)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的日志ID",
		})
		return
	}

	log, err := s.taskService.GetEventLog(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "日志不存在",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": log,
	})
}

// getStatusHandler 获取系统状态
func (s *Server) getStatusHandler(c *gin.Context) {
	// 获取活跃任务数量
	activeTasks, err := s.taskService.GetActiveTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取系统状态失败: " + err.Error(),
		})
		return
	}

	// 获取Canal服务状态
	canalStatus := "running"
	if s.canalService == nil {
		canalStatus = "stopped"
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":       canalStatus,
			"active_tasks": len(activeTasks),
			"version":      "1.0.0",
		},
	})
}

// getPerformanceMetricsHandler 获取性能指标
func (s *Server) getPerformanceMetricsHandler(c *gin.Context) {
	// 如果有增强的handlers，使用增强的实现
	if s.enhancedHandlers != nil {
		s.enhancedHandlers.getPerformanceMetricsHandler(c)
		return
	}

	// 否则使用基础实现
	status := s.canalService.GetStatus()

	// 从状态中提取或计算指标
	eventsProcessed := 0
	if instances, ok := status["instances"].(map[string]interface{}); ok {
		// 计算总处理事件数
		for _, instanceStatus := range instances {
			if statusMap, ok := instanceStatus.(map[string]interface{}); ok {
				if processed, ok := statusMap["processed_events"]; ok {
					if processedInt, ok := processed.(int); ok {
						eventsProcessed += processedInt
					} else if processedFloat, ok := processed.(float64); ok {
						eventsProcessed += int(processedFloat)
					}
				}
			}
		}
	}

	// 获取实例数量
	instanceCountFromStatus := 0
	if count, ok := status["instance_count"].(int); ok {
		instanceCountFromStatus = count
	}

	// 获取内存使用情况
	memoryUsage := "0 MB"
	if mem, ok := status["memory_usage"].(string); ok {
		memoryUsage = mem
	}

	// 获取连接池状态
	connectionPool := "0/0"
	if pool, ok := status["connection_pool"].(string); ok {
		connectionPool = pool
	}

	// 获取架构信息
	architecture := "x86_64" // 默认值

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"canal_status":      status,
			"events_processed":  eventsProcessed,
			"events_per_second": 123.45, // 这个值需要从服务层获取或计算
			"error_rate":        0.05,   // 这个值需要从服务层获取或计算
			"uptime_seconds":    3600,   // 这个值需要从服务层获取或计算
			"architecture":      architecture,
			"instance_count":    instanceCountFromStatus,
			"memory_usage":      memoryUsage,
			"connection_pool":   connectionPool,
		},
	})
}
