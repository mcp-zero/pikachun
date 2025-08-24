package canal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLBinlogSlave 纯粹的 MySQL Binlog 从库实现
// 借鉴 Canal 思想，但只使用 go-mysql-org/go-mysql 的 replication 包
type MySQLBinlogSlave struct {
	config    MySQLConfig
	eventSink *DefaultEventSink
	logger    *log.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex

	// 监听配置
	watchTables map[string]bool    // schema.table -> enabled
	eventTypes  map[EventType]bool // 监听的事件类型

	// 运行状态
	running    bool
	instanceID string

	// go-mysql replication 组件
	syncer   *replication.BinlogSyncer
	streamer *replication.BinlogStreamer

	// binlog 位置信息
	binlogPos mysql.Position
	gtidSet   mysql.GTIDSet

	// 重连和容错机制
	reconnectInterval time.Duration
	maxReconnectCount int
	reconnectCount    int
	lastEventTime     time.Time

	// 表结构缓存
	tableSchemas map[string]*TableSchema // schema.table -> TableSchema

	// 性能统计
	eventCounter  map[EventType]int64
	lastStatsTime time.Time

	// 元数据管理器（用于断点续传）
	metaManager MetaManager
}

// TableSchema 表结构信息
type TableSchema struct {
	Schema    string
	Table     string
	Columns   []ColumnInfo
	PKColumns []int // 主键列索引
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	IsPK     bool
}

// NewMySQLBinlogSlave 创建 MySQL binlog 从库
func NewMySQLBinlogSlave(config MySQLConfig, eventSink *DefaultEventSink, logger *log.Logger) (*MySQLBinlogSlave, error) {
	return NewMySQLBinlogSlaveWithMeta(config, eventSink, logger, nil)
}

// NewMySQLBinlogSlaveWithMeta 创建带元数据管理器的 MySQL binlog 从库
func NewMySQLBinlogSlaveWithMeta(config MySQLConfig, eventSink *DefaultEventSink, logger *log.Logger, metaManager MetaManager) (*MySQLBinlogSlave, error) {
	logger.Printf("🔧 Creating MySQL binlog slave for %s:%d (serverID: %d, database: %s)", config.Host, config.Port, config.ServerID, config.Database)

	instanceID := fmt.Sprintf("mysql-slave-%s-%d-%d", config.Host, config.Port, config.ServerID)

	slave := &MySQLBinlogSlave{
		config:            config,
		eventSink:         eventSink,
		logger:            logger,
		instanceID:        instanceID,
		watchTables:       make(map[string]bool),
		eventTypes:        make(map[EventType]bool),
		tableSchemas:      make(map[string]*TableSchema),
		eventCounter:      make(map[EventType]int64),
		reconnectInterval: 5 * time.Second,
		maxReconnectCount: 10,
		lastEventTime:     time.Now(),
		lastStatsTime:     time.Now(),
		metaManager:       metaManager,
		binlogPos:         mysql.Position{Name: "mysql-bin.000001", Pos: 4},
	}

	logger.Printf("🔧 Initialized binlog position: %s:%d", "mysql-bin.000001", 4)

	// 默认监听所有事件类型
	slave.eventTypes[EventTypeInsert] = true
	slave.eventTypes[EventTypeUpdate] = true
	slave.eventTypes[EventTypeDelete] = true

	logger.Printf("🔧 Set default event types: INSERT, UPDATE, DELETE")

	// 初始化 binlog 同步器
	logger.Printf("🔧 Initializing binlog syncer...")
	if err := slave.initBinlogSyncer(); err != nil {
		logger.Printf("❌ Failed to initialize binlog syncer: %v", err)
		return nil, fmt.Errorf("failed to initialize binlog syncer: %v", err)
	}

	logger.Printf("✅ MySQL binlog slave created successfully for %s:%d", config.Host, config.Port)
	return slave, nil
}

// initBinlogSyncer 初始化 binlog 同步器
func (m *MySQLBinlogSlave) initBinlogSyncer() error {
	m.logger.Printf("🔧 Initializing binlog syncer for %s:%d with ServerID: %d", m.config.Host, m.config.Port, m.config.ServerID)

	cfg := replication.BinlogSyncerConfig{
		ServerID: m.config.ServerID,
		Flavor:   "mysql",
		Host:     m.config.Host,
		Port:     uint16(m.config.Port),
		User:     m.config.Username,
		Password: m.config.Password,

		// 启用校验和验证
		UseDecimal:     true,
		VerifyChecksum: true,

		// 心跳和超时配置
		HeartbeatPeriod: 30 * time.Second,
		ReadTimeout:     90 * time.Second,

		// 启用 GTID 支持
		ParseTime: true,

		// 设置字符集
		Charset: "utf8mb4",
	}

	m.logger.Printf("🔧 Binlog syncer config: Host=%s, Port=%d, ServerID=%d, User=%s",
		m.config.Host, m.config.Port, m.config.ServerID, m.config.Username)

	m.syncer = replication.NewBinlogSyncer(cfg)
	m.logger.Printf("✅ MySQL Binlog Syncer initialized with ServerID: %d", m.config.ServerID)
	return nil
}

// Start 启动 MySQL binlog 从库
func (m *MySQLBinlogSlave) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Printf("⚠️ MySQL binlog slave is already running")
		return fmt.Errorf("mysql binlog slave is already running")
	}

	m.logger.Printf("🔧 Starting MySQL Binlog Slave...")
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.running = true

	m.logger.Printf("🚀 Starting MySQL Binlog Slave")
	m.logger.Printf("📡 MySQL Server: %s:%d", m.config.Host, m.config.Port)
	m.logger.Printf("🆔 Server ID: %d", m.config.ServerID)
	m.logger.Printf("🏗️ Implementation: Pure go-mysql-org/go-mysql replication")

	// 测试连接到 MySQL 服务器
	m.logger.Printf("🔧 Testing connection to MySQL server...")
	if m.metaManager == nil {
		m.logger.Printf("🔧 No meta manager, testing direct connection...")
		if err := m.testConnection(); err != nil {
			m.logger.Printf("❌ Failed to connect to MySQL server: %v", err)
			m.running = false
			return fmt.Errorf("failed to connect to MySQL server: %v", err)
		}
		m.logger.Printf("✅ Successfully connected to MySQL server")
	} else {
		m.logger.Printf("🔧 Using meta manager for connection")
	}

	// 获取当前 binlog 位置
	m.logger.Printf("🔧 Getting current binlog position...")
	if err := m.getCurrentPosition(); err != nil {
		m.logger.Printf("⚠️ Failed to get current position, using default: %v", err)
		// 如果没有元数据管理器且获取位置失败，返回错误
		if m.metaManager == nil {
			m.logger.Printf("❌ Failed to get current position and no meta manager available: %v", err)
			m.running = false
			return fmt.Errorf("failed to get current position and no meta manager available: %v", err)
		}
		m.binlogPos = mysql.Position{Name: "mysql-bin.000001", Pos: 4}
		m.logger.Printf("🔧 Using default binlog position: %s:%d", m.binlogPos.Name, m.binlogPos.Pos)
	} else {
		m.logger.Printf("✅ Current binlog position: %s:%d", m.binlogPos.Name, m.binlogPos.Pos)
	}

	// 启动 binlog 流处理
	m.logger.Printf("🔧 Starting binlog stream processing goroutine...")
	m.wg.Add(1)
	go m.runBinlogStream()

	// 启动监控协程
	m.logger.Printf("🔧 Starting monitor goroutine...")
	m.wg.Add(1)
	go m.monitor()

	// 启动统计协程
	m.logger.Printf("🔧 Starting stats reporter goroutine...")
	m.wg.Add(1)
	go m.statsReporter()

	m.logger.Printf("✅ MySQL Binlog Slave started successfully")
	return nil
}

// Stop 停止 binlog 从库
func (m *MySQLBinlogSlave) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.logger.Printf("🛑 Stopping MySQL Binlog Slave")

	// 取消上下文
	if m.cancel != nil {
		m.cancel()
	}

	// 关闭 binlog 流 (BinlogStreamer 没有 Close 方法，设置为 nil)
	if m.streamer != nil {
		m.streamer = nil
	}

	// 关闭 binlog 同步器
	if m.syncer != nil {
		m.syncer.Close()
	}

	// 等待所有协程结束
	m.wg.Wait()

	m.running = false
	m.logger.Printf("✅ MySQL Binlog Slave stopped")
	return nil
}

// getCurrentPosition 获取当前 binlog 位置
func (m *MySQLBinlogSlave) getCurrentPosition() error {
	m.logger.Printf("🔧 Getting current binlog position...")

	// 如果有元数据管理器，尝试从中恢复位置
	if m.metaManager != nil {
		m.logger.Printf("🔧 Trying to restore position from metadata manager...")
		if pos, err := m.metaManager.LoadPosition(m.instanceID); err == nil {
			m.binlogPos = mysql.Position{
				Name: pos.Name,
				Pos:  pos.Pos,
			}
			m.logger.Printf("📍 Restored binlog position from metadata: %s:%d", m.binlogPos.Name, m.binlogPos.Pos)
			return nil
		} else {
			m.logger.Printf("⚠️ Failed to load position from metadata: %v", err)
		}
	} else {
		m.logger.Printf("🔧 No metadata manager available, using default position")
	}

	// 使用默认位置
	m.binlogPos = mysql.Position{Name: "", Pos: 4}
	m.logger.Printf("📍 Starting from default binlog position: %s:%d", m.binlogPos.Name, m.binlogPos.Pos)
	return nil
}

// runBinlogStream 运行 binlog 流处理
func (m *MySQLBinlogSlave) runBinlogStream() {
	defer m.wg.Done()

	m.logger.Printf("🔥 Starting MySQL binlog stream processing...")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Printf("🛑 Binlog stream processing stopped")
			return
		default:
			if err := m.processBinlogStream(); err != nil {
				m.logger.Printf("❌ Binlog stream error: %v", err)
				m.handleReconnect("Binlog stream failed")

				// 等待一段时间后重试
				select {
				case <-m.ctx.Done():
					return
				case <-time.After(m.reconnectInterval):
					continue
				}
			}
		}
	}
}

// processBinlogStream 处理 binlog 流
func (m *MySQLBinlogSlave) processBinlogStream() error {
	// 创建 binlog 流
	streamer, err := m.syncer.StartSync(m.binlogPos)
	if err != nil {
		return fmt.Errorf("failed to start sync: %v", err)
	}
	m.streamer = streamer

	m.logger.Printf("📡 Binlog stream started from position: %s:%d", m.binlogPos.Name, m.binlogPos.Pos)

	for {
		select {
		case <-m.ctx.Done():
			return nil
		default:
			// 读取 binlog 事件
			ev, err := streamer.GetEvent(m.ctx)
			if err != nil {
				return fmt.Errorf("failed to get binlog event: %v", err)
			}

			// 更新最后事件时间
			m.lastEventTime = time.Now()

			// 处理事件
			if err := m.handleBinlogEvent(ev); err != nil {
				m.logger.Printf("❌ Failed to handle binlog event: %v", err)
			}

			// 更新位置
			m.updatePosition(ev)
		}
	}
}

// handleBinlogEvent 处理 binlog 事件
func (m *MySQLBinlogSlave) handleBinlogEvent(ev *replication.BinlogEvent) error {
	switch e := ev.Event.(type) {
	case *replication.RowsEvent:
		return m.handleRowsEvent(ev.Header, e)
	case *replication.QueryEvent:
		return m.handleQueryEvent(ev.Header, e)
	case *replication.XIDEvent:
		return m.handleXIDEvent(ev.Header, e)
	case *replication.GTIDEvent:
		return m.handleGTIDEvent(ev.Header, e)
	case *replication.RotateEvent:
		return m.handleRotateEvent(ev.Header, e)
	case *replication.TableMapEvent:
		return m.handleTableMapEvent(ev.Header, e)
	default:
		// 忽略其他类型的事件
		return nil
	}
}

// handleRowsEvent 处理行变更事件
func (m *MySQLBinlogSlave) handleRowsEvent(header *replication.EventHeader, e *replication.RowsEvent) error {
	m.logger.Printf("📥 Processing rows event: %s", header.EventType.String())

	// 获取表信息
	schemaName := string(e.Table.Schema)
	tableName := string(e.Table.Table)
	tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)

	m.logger.Printf("📋 Table info: schema=%s, table=%s, tableKey=%s", schemaName, tableName, tableKey)

	// 检查是否需要监听此表
	m.mu.RLock()
	shouldWatch := m.watchTables[tableKey]
	if len(m.watchTables) > 0 && !shouldWatch {
		m.mu.RUnlock()
		return nil // 不在监听列表中，忽略
	}
	m.mu.RUnlock()

	// 根据事件类型处理
	var eventType EventType
	switch header.EventType {
	case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
		eventType = EventTypeInsert
	case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
		eventType = EventTypeUpdate
	case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
		eventType = EventTypeDelete
	default:
		return nil
	}

	// 检查是否监听此事件类型
	m.mu.RLock()
	shouldHandleEventType := m.eventTypes[eventType]
	m.mu.RUnlock()

	if !shouldHandleEventType {
		return nil // 不监听此事件类型
	}

	// 获取表结构
	m.logger.Printf("🔍 Getting table schema for %s.%s", schemaName, tableName)
	tableSchema := m.getTableSchema(schemaName, tableName, e.Table)
	m.logger.Printf("✅ Got table schema with %d columns", len(tableSchema.Columns))

	// 处理每一行数据
	m.logger.Printf("🔄 Processing %d rows", len(e.Rows))
	for i, row := range e.Rows {
		m.logger.Printf("📝 Processing row %d/%d", i+1, len(e.Rows))
		event := m.createCanalEvent(header, tableSchema, eventType, row, i, e.Rows)
		m.logger.Printf("🔧 Created canal event: %s.%s %s", event.Schema, event.Table, event.EventType)

		if err := m.eventSink.SendEvent(event); err != nil {
			m.logger.Printf("❌ Failed to send event: %v", err)
			return fmt.Errorf("failed to send event: %v", err)
		}
		m.logger.Printf("✅ Event sent to sink successfully")

		// 更新统计
		m.mu.Lock()
		m.eventCounter[eventType]++
		m.mu.Unlock()

		m.logger.Printf("🔥 MYSQL BINLOG EVENT PROCESSED:")
		m.logger.Printf("   📋 Table: %s.%s", event.Schema, event.Table)
		m.logger.Printf("   🎯 Event Type: %s", event.EventType)
		m.logger.Printf("   📍 Position: %s:%d", event.Position.Name, event.Position.Pos)
		m.logger.Printf("   🆔 Event ID: %s", event.ID)
		m.logger.Printf("   📊 Data: %v", m.formatEventData(event))
	}
	m.logger.Printf("✅ Finished processing %d rows", len(e.Rows))

	return nil
}

// getTableSchema 获取表结构
func (m *MySQLBinlogSlave) getTableSchema(schema, table string, tableInfo *replication.TableMapEvent) *TableSchema {
	tableKey := fmt.Sprintf("%s.%s", schema, table)

	m.mu.RLock()
	if ts, exists := m.tableSchemas[tableKey]; exists {
		m.mu.RUnlock()
		return ts
	}
	m.mu.RUnlock()

	// 创建基本的表结构信息
	ts := &TableSchema{
		Schema:  schema,
		Table:   table,
		Columns: make([]ColumnInfo, len(tableInfo.ColumnType)),
	}

	// 填充列信息（这里简化处理，实际应该查询 information_schema）
	for i, colType := range tableInfo.ColumnType {
		ts.Columns[i] = ColumnInfo{
			Name:     fmt.Sprintf("col_%d", i), // 实际应该查询真实列名
			Type:     m.getColumnTypeName(colType),
			Nullable: true,  // 实际应该查询真实的 nullable 信息
			IsPK:     false, // 实际应该查询主键信息
		}
	}

	// 缓存表结构
	m.mu.Lock()
	m.tableSchemas[tableKey] = ts
	m.mu.Unlock()

	return ts
}

// getColumnTypeName 获取列类型名称
func (m *MySQLBinlogSlave) getColumnTypeName(colType byte) string {
	switch colType {
	case 1:
		return "tinyint"
	case 2:
		return "smallint"
	case 3:
		return "int"
	case 8:
		return "bigint"
	case 4:
		return "float"
	case 5:
		return "double"
	case 246:
		return "decimal"
	case 253, 254:
		return "varchar"
	case 12:
		return "datetime"
	case 10:
		return "date"
	case 11:
		return "time"
	case 7:
		return "timestamp"
	default:
		return fmt.Sprintf("unknown_%d", colType)
	}
}

// createCanalEvent 创建 Canal 事件
func (m *MySQLBinlogSlave) createCanalEvent(header *replication.EventHeader, tableSchema *TableSchema, eventType EventType, row []interface{}, rowIndex int, allRows [][]interface{}) *Event {
	event := &Event{
		ID:        fmt.Sprintf("mysql-binlog-%d-%d-%d", header.LogPos, header.Timestamp, rowIndex),
		Schema:    tableSchema.Schema,
		Table:     tableSchema.Table,
		EventType: eventType,
		Timestamp: time.Unix(int64(header.Timestamp), 0),
		Position: Position{
			Name: m.binlogPos.Name,
			Pos:  header.LogPos,
		},
	}

	// 设置 GTID
	if m.gtidSet != nil {
		event.Position.GTIDSet = m.gtidSet.String()
	}

	// 根据事件类型设置数据
	switch eventType {
	case EventTypeInsert:
		event.AfterData = m.convertRowToRowData(tableSchema, row)
	case EventTypeDelete:
		event.BeforeData = m.convertRowToRowData(tableSchema, row)
	case EventTypeUpdate:
		if rowIndex%2 == 0 && rowIndex+1 < len(allRows) {
			// UPDATE 事件的行数据是成对出现的：before, after
			event.BeforeData = m.convertRowToRowData(tableSchema, row)
			event.AfterData = m.convertRowToRowData(tableSchema, allRows[rowIndex+1])
		}
	}

	return event
}

// convertRowToRowData 将行数据转换为 RowData
func (m *MySQLBinlogSlave) convertRowToRowData(tableSchema *TableSchema, row []interface{}) *RowData {
	columns := make([]Column, len(tableSchema.Columns))

	for i, colInfo := range tableSchema.Columns {
		var value interface{}
		var isNull bool

		if i < len(row) {
			value = row[i]
			isNull = (value == nil)
		} else {
			isNull = true
		}

		columns[i] = Column{
			Name:   colInfo.Name,
			Type:   colInfo.Type,
			Value:  value,
			IsNull: isNull,
		}
	}

	return &RowData{
		Columns: columns,
	}
}

// formatEventData 格式化事件数据
func (m *MySQLBinlogSlave) formatEventData(event *Event) map[string]interface{} {
	result := make(map[string]interface{})

	if event.BeforeData != nil {
		before := make(map[string]interface{})
		for _, col := range event.BeforeData.Columns {
			if col.IsNull {
				before[col.Name] = nil
			} else {
				before[col.Name] = col.Value
			}
		}
		result["before"] = before
	}

	if event.AfterData != nil {
		after := make(map[string]interface{})
		for _, col := range event.AfterData.Columns {
			if col.IsNull {
				after[col.Name] = nil
			} else {
				after[col.Name] = col.Value
			}
		}
		result["after"] = after
	}

	return result
}

// handleQueryEvent 处理查询事件
func (m *MySQLBinlogSlave) handleQueryEvent(header *replication.EventHeader, e *replication.QueryEvent) error {
	m.logger.Printf("📝 DDL Query: %s", string(e.Query))
	return nil
}

// handleXIDEvent 处理事务提交事件
func (m *MySQLBinlogSlave) handleXIDEvent(header *replication.EventHeader, e *replication.XIDEvent) error {
	m.logger.Printf("💾 Transaction committed")
	return nil
}

// handleGTIDEvent 处理 GTID 事件
func (m *MySQLBinlogSlave) handleGTIDEvent(header *replication.EventHeader, e *replication.GTIDEvent) error {
	m.logger.Printf("🔗 GTID Event received")
	return nil
}

// handleRotateEvent 处理 binlog 轮转事件
func (m *MySQLBinlogSlave) handleRotateEvent(header *replication.EventHeader, e *replication.RotateEvent) error {
	m.logger.Printf("🔄 Binlog rotated to: %s", string(e.NextLogName))
	return nil
}

// handleTableMapEvent 处理表映射事件
func (m *MySQLBinlogSlave) handleTableMapEvent(header *replication.EventHeader, e *replication.TableMapEvent) error {
	tableKey := fmt.Sprintf("%s.%s", string(e.Schema), string(e.Table))
	m.logger.Printf("🗺️ Table map event: %s", tableKey)
	return nil
}

// updatePosition 更新 binlog 位置
func (m *MySQLBinlogSlave) updatePosition(ev *replication.BinlogEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldPos := m.binlogPos
	m.binlogPos.Pos = ev.Header.LogPos

	if ev.Header.EventType == replication.ROTATE_EVENT {
		if rotateEvent, ok := ev.Event.(*replication.RotateEvent); ok {
			m.binlogPos.Name = string(rotateEvent.NextLogName)
			m.binlogPos.Pos = uint32(rotateEvent.Position)
		}
	}

	// 如果位置发生变化且有元数据管理器，保存位置
	if m.metaManager != nil && (oldPos.Name != m.binlogPos.Name || oldPos.Pos != m.binlogPos.Pos) {
		pos := Position{
			Name: m.binlogPos.Name,
			Pos:  m.binlogPos.Pos,
		}
		if m.gtidSet != nil {
			pos.GTIDSet = m.gtidSet.String()
		}

		// 异步保存位置，避免阻塞事件处理
		go func() {
			if err := m.metaManager.SavePosition(m.instanceID, pos); err != nil {
				m.logger.Printf("❌ Failed to save binlog position: %v", err)
			}
		}()
	}
}

// monitor 监控协程
func (m *MySQLBinlogSlave) monitor() {
	m.logger.Printf("👀 Starting monitor goroutine")
	defer m.wg.Done()
	defer m.logger.Printf("👋 Monitor goroutine stopped")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Printf("🛑 Monitor context cancelled")
			return
		case <-ticker.C:
			m.logger.Printf("📊 Running periodic status check")
			m.logStatus()
			m.checkHealth()
			m.logger.Printf("✅ Periodic status check completed")
		}
	}
}

// statsReporter 统计报告协程
func (m *MySQLBinlogSlave) statsReporter() {
	m.logger.Printf("📈 Starting stats reporter goroutine")
	defer m.wg.Done()
	defer m.logger.Printf("👋 Stats reporter goroutine stopped")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Printf("🛑 Stats reporter context cancelled")
			return
		case <-ticker.C:
			m.logger.Printf("📊 Reporting statistics")
			m.reportStats()
			m.logger.Printf("✅ Statistics reported")
		}
	}
}

// logStatus 记录状态
func (m *MySQLBinlogSlave) logStatus() {
	m.mu.RLock()
	pos := m.binlogPos
	running := m.running
	lastEvent := m.lastEventTime
	m.mu.RUnlock()

	if running {
		m.logger.Printf("📊 Binlog Status: %s:%d, Running: %v, Last Event: %s",
			pos.Name, pos.Pos, running, lastEvent.Format("2006-01-02 15:04:05"))
	}
}

// reportStats 报告统计信息
func (m *MySQLBinlogSlave) reportStats() {
	m.mu.RLock()
	stats := make(map[EventType]int64)
	for k, v := range m.eventCounter {
		stats[k] = v
	}
	m.mu.RUnlock()

	m.logger.Printf("📈 Event Statistics:")
	for eventType, count := range stats {
		m.logger.Printf("   %s: %d", eventType, count)
	}
}

// checkHealth 检查健康状态
func (m *MySQLBinlogSlave) checkHealth() {
	// 检查是否长时间没有收到事件
	if time.Since(m.lastEventTime) > 5*time.Minute {
		m.logger.Printf("⚠️ No events received for %v", time.Since(m.lastEventTime))
	}
}

// handleReconnect 处理重连
func (m *MySQLBinlogSlave) handleReconnect(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reconnectCount++
	if m.reconnectCount > m.maxReconnectCount {
		m.logger.Printf("❌ Max reconnect attempts reached, stopping slave")
		return
	}

	m.logger.Printf("🔄 Reconnecting (attempt %d/%d) due to: %s", m.reconnectCount, m.maxReconnectCount, reason)

	// 等待重连间隔
	time.Sleep(m.reconnectInterval)

	// 重新初始化连接
	if err := m.initBinlogSyncer(); err != nil {
		m.logger.Printf("❌ Failed to reinitialize binlog syncer: %v", err)
	} else {
		// 重置重连计数
		m.reconnectCount = 0
	}
}

// AddWatchTable 添加监听表
func (m *MySQLBinlogSlave) AddWatchTable(schema, table string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s.%s", schema, table)
	m.watchTables[key] = true
	m.logger.Printf("📋 Added watch table: %s", key)
}

// RemoveWatchTable 移除监听表
func (m *MySQLBinlogSlave) RemoveWatchTable(schema, table string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s.%s", schema, table)
	delete(m.watchTables, key)
	m.logger.Printf("📋 Removed watch table: %s", key)
}

// SetEventTypes 设置监听的事件类型
func (m *MySQLBinlogSlave) SetEventTypes(eventTypes []EventType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空现有配置
	m.eventTypes = make(map[EventType]bool)

	// 设置新的事件类型
	for _, eventType := range eventTypes {
		m.eventTypes[eventType] = true
	}

	m.logger.Printf("🎯 Set event types: %v", eventTypes)
}

// testConnection 测试到 MySQL 服务器的连接
func (m *MySQLBinlogSlave) testConnection() error {
	m.logger.Printf("🔧 Testing MySQL connection to %s:%d with user %s", m.config.Host, m.config.Port, m.config.Username)

	// 在测试环境中，如果设置了 TEST_MYSQL_CONNECTION_FAIL 环境变量，则模拟连接失败
	if os.Getenv("TEST_MYSQL_CONNECTION_FAIL") == "true" {
		m.logger.Printf("❌ Simulated connection failure for testing")
		return fmt.Errorf("simulated connection failure for testing")
	}

	// 创建一个简单的连接来测试 MySQL 服务器是否可达
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
	)

	m.logger.Printf("🔧 DSN for connection test: %s:***@tcp(%s:%d)/?charset=utf8mb4",
		m.config.Username, m.config.Host, m.config.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		m.logger.Printf("❌ Failed to create connection to %s:%d: %v", m.config.Host, m.config.Port, err)
		return fmt.Errorf("failed to create connection to %s:%d: %v", m.config.Host, m.config.Port, err)
	}
	defer db.Close()

	// 尝试连接到数据库
	m.logger.Printf("🔧 Pinging MySQL server at %s:%d", m.config.Host, m.config.Port)
	// 尝试连接到数据库
	m.logger.Printf("🔧 Attempting to ping MySQL server at %s:%d", m.config.Host, m.config.Port)
	if err := db.Ping(); err != nil {
		m.logger.Printf("❌ Failed to ping MySQL server at %s:%d: %v", m.config.Host, m.config.Port, err)
		return fmt.Errorf("failed to ping MySQL server at %s:%d: %v", m.config.Host, m.config.Port, err)
	}

	m.logger.Printf("✅ Successfully connected to MySQL server at %s:%d", m.config.Host, m.config.Port)
	return nil
}

// GetBinlogPosition 获取当前 binlog 位置
func (m *MySQLBinlogSlave) GetBinlogPosition() Position {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Position{
		Name: m.binlogPos.Name,
		Pos:  m.binlogPos.Pos,
	}
}

// IsRunning 检查是否正在运行
func (m *MySQLBinlogSlave) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetStats 获取统计信息
func (m *MySQLBinlogSlave) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"running":         m.running,
		"position":        m.binlogPos,
		"last_event_time": m.lastEventTime,
		"reconnect_count": m.reconnectCount,
		"watched_tables":  len(m.watchTables),
		"event_counter":   m.eventCounter,
	}

	return stats
}

// String 实现 Stringer 接口
func (m *MySQLBinlogSlave) String() string {
	return fmt.Sprintf("MySQLBinlogSlave{host: %s:%d, serverID: %d, pure_replication: true}",
		m.config.Host, m.config.Port, m.config.ServerID)
}
