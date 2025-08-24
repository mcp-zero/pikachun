/*
 * @Author: lucklidi@126.com
 * @Date: 2025-08-21 17:03:03
 * @LastEditTime: 2025-08-21 17:03:13
 * @LastEditors: lucklidi@126.com
 * @Description:
 * Copyright (c) 2023 by pikachun
 */
package server

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"pikachun/internal/database"
)

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string `json:"name" binding:"required"`
	Database    string `json:"database" binding:"required"`
	Table       string `json:"table" binding:"required"`
	EventTypes  string `json:"event_types" binding:"required"`
	CallbackURL string `json:"callback_url" binding:"required"`
}

// ToTask 转换为Task模型
func (r *CreateTaskRequest) ToTask() *database.Task {
	return &database.Task{
		Name:        r.Name,
		Database:    r.Database,
		Table:       r.Table,
		EventTypes:  r.EventTypes,
		CallbackURL: r.CallbackURL,
		Status:      "active",
	}
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name        *string `json:"name,omitempty"`
	Database    *string `json:"database,omitempty"`
	Table       *string `json:"table,omitempty"`
	EventTypes  *string `json:"event_types,omitempty"`
	CallbackURL *string `json:"callback_url,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// ToTask 转换为Task模型
func (r *UpdateTaskRequest) ToTask() *database.Task {
	task := &database.Task{}
	if r.Name != nil {
		task.Name = *r.Name
	}
	if r.Database != nil {
		task.Database = *r.Database
	}
	if r.Table != nil {
		task.Table = *r.Table
	}
	if r.EventTypes != nil {
		task.EventTypes = *r.EventTypes
	}
	if r.CallbackURL != nil {
		task.CallbackURL = *r.CallbackURL
	}
	if r.Status != nil {
		task.Status = *r.Status
	}
	return task
}

// parseIntDefault 解析整数，失败时返回默认值
func parseIntDefault(s string, defaultValue int) (int, error) {
	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}
	return defaultValue, nil
}

// parseUintDefault 解析无符号整数，失败时返回默认值
func parseUintDefault(s string, defaultValue uint) (uint, error) {
	if i, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint(i), nil
	}
	return defaultValue, nil
}

// parseUintParam 从URL参数解析无符号整数
func parseUintParam(c *gin.Context, param string) (uint, error) {
	s := c.Param(param)
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(i), nil
}
