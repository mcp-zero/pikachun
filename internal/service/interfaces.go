/*
 * @Author: lucklidi@126.com
 * @Date: 2025-08-21 17:05:39
 * @LastEditTime: 2025-08-24 18:14:24
 * @LastEditors: lidi10@staff.weibo.com
 * @Description:
 * Copyright (c) 2023 by pikachun
 */
package service

import (
	"context"

	"pikachun/internal/database"
)

// CanalServiceInterface Canal服务接口
type CanalServiceInterface interface {
	Start(ctx context.Context) error
	Stop() error
	StopInstance(instanceID uint) error
	UpdateInstance(instanceID uint, task *database.Task) error
	CreateTask(task *database.Task) error
	GetStatus() map[string]interface{}
}
