package canal

import (
	"context"
	"time"

	"pikachun/internal/database"
)

// EventType 事件类型
type EventType string

const (
	EventTypeInsert EventType = "INSERT"
	EventTypeUpdate EventType = "UPDATE"
	EventTypeDelete EventType = "DELETE"
)

// Position binlog位置信息
type Position struct {
	Name    string `json:"name"`
	Pos     uint32 `json:"pos"`
	GTIDSet string `json:"gtid_set,omitempty"`
}

// RowData 行数据
type RowData struct {
	Columns []Column `json:"columns"`
}

// Column 列数据
type Column struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Value   interface{} `json:"value"`
	IsNull  bool        `json:"is_null"`
	Updated bool        `json:"updated,omitempty"`
}

// Event 数据变更事件
type Event struct {
	ID         string    `json:"id"`
	Schema     string    `json:"schema"`
	Table      string    `json:"table"`
	EventType  EventType `json:"event_type"`
	Timestamp  time.Time `json:"timestamp"`
	Position   Position  `json:"position"`
	BeforeData *RowData  `json:"before_data,omitempty"`
	AfterData  *RowData  `json:"after_data,omitempty"`
	SQL        string    `json:"sql,omitempty"`
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event *Event) error
	GetName() string
}

// EventSink 事件接收器接口
type EventSink interface {
	Start(ctx context.Context) error
	Stop() error
	Subscribe(schema, table string, handler EventHandler) error
	Unsubscribe(schema, table string, handlerName string) error
}

// Parser binlog解析器接口
type Parser interface {
	Start(ctx context.Context) error
	Stop() error
	SetEventSink(sink EventSink)
	GetPosition() Position
	SetPosition(pos Position) error
}

// MetaManager 元数据管理器接口
type MetaManager interface {
	SavePosition(instanceID string, pos Position) error
	LoadPosition(instanceID string) (Position, error)
	SaveTableMeta(schema, table string, meta *TableMeta) error
	LoadTableMeta(schema, table string) (*TableMeta, error)
}

// TableMeta 表元数据
type TableMeta struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Types   []string `json:"types"`
}

// CanalInstance Canal实例接口
type CanalInstance interface {
	Start(ctx context.Context) error
	Stop() error
	// 停止某个实例
	StopInstance(instanceID uint) error
	// 更新某个实例
	UpdateInstance(instanceID uint, task *database.Task) error
	Subscribe(schema, table string, handler EventHandler) error
	Unsubscribe(schema, table string, handlerName string) error
	GetStatus() InstanceStatus
	GetStats() map[string]interface{}
}

// InstanceStatus 实例状态
type InstanceStatus struct {
	Running   bool      `json:"running"`
	Position  Position  `json:"position"`
	LastEvent time.Time `json:"last_event"`
	ErrorMsg  string    `json:"error_msg,omitempty"`
}

// BinlogSlave binlog 从库接口
type BinlogSlave interface {
	Start() error
	Stop() error
	AddWatchTable(schema, table string)
	RemoveWatchTable(schema, table string)
	SetEventTypes(eventTypes []EventType)
	GetBinlogPosition() Position
	IsRunning() bool
	GetStats() map[string]interface{}
	String() string
}

// EventLogger 事件日志接口
type EventLogger interface {
	CreateEventLog(taskID uint, database, table, eventType, data, status, errorMsg string) error
}
