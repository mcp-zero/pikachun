// 全局变量
let currentPage = 1;
let currentTab = 'tasks';

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    initializeTabs();
    loadTasks();
    loadSystemStatus();
    
    // 每30秒刷新一次状态
    setInterval(loadSystemStatus, 30000);
});

// 初始化标签页
function initializeTabs() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            const tabId = this.getAttribute('data-tab');
            
            // 移除所有活跃状态
            tabBtns.forEach(b => b.classList.remove('active'));
            tabContents.forEach(c => c.classList.remove('active'));
            
            // 添加活跃状态
            this.classList.add('active');
            document.getElementById(tabId).classList.add('active');
            
            currentTab = tabId;
            
            // 根据标签页加载相应数据
            switch(tabId) {
                case 'tasks':
                    loadTasks();
                    break;
                case 'logs':
                    loadEventLogs();
                    loadTasksForFilter();
                    break;
                case 'status':
                    loadSystemStatus();
                    break;
                // case 'binlog':
                //     loadBinlogInfo();
                //     break;
                case 'metrics':
                    loadMetrics();
                    break;
            }
        });
    });
}

// 加载任务列表
async function loadTasks(page = 1) {
    try {
        const response = await fetch(`/api/tasks?page=${page}&page_size=10`);
        const result = await response.json();
        
        if (response.ok) {
            renderTasksTable(result.data.tasks);
            renderPagination('tasksPagination', result.data.page, Math.ceil(result.data.total / result.data.page_size), loadTasks);
        } else {
            showError('加载任务列表失败: ' + result.error);
        }
    } catch (error) {
        showError('网络错误: ' + error.message);
    }
}

// 渲染任务表格
function renderTasksTable(tasks) {
    const tbody = document.querySelector('#tasksTable tbody');
    tbody.innerHTML = '';
    
    if (!tasks || tasks.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: #666;">暂无数据</td></tr>';
        return;
    }
    
    tasks.forEach(task => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${task.id}</td>
            <td>${task.name}</td>
            <td>${task.database}</td>
            <td>${task.table}</td>
            <td>${task.event_types}</td>
            <td><span class="url-text" title="${task.callback_url}">${truncateUrl(task.callback_url)}</span></td>
            <td><span class="status-badge status-${task.status}">${getStatusText(task.status)}</span></td>
            <td>
                <button class="btn btn-small btn-secondary" onclick="editTask(${task.id})">编辑</button>
                <button class="btn btn-small btn-danger" onclick="deleteTask(${task.id})">删除</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// 加载事件日志
async function loadEventLogs(page = 1) {
    try {
        const taskId = document.getElementById('taskFilter').value;
        const url = `/api/logs?page=${page}&page_size=20${taskId ? '&task_id=' + taskId : ''}`;
        
        const response = await fetch(url);
        const result = await response.json();
        
        if (response.ok) {
            renderLogsTable(result.data.logs);
            renderPagination('logsPagination', result.data.page, Math.ceil(result.data.total / result.data.page_size), loadEventLogs);
        } else {
            showError('加载事件日志失败: ' + result.error);
        }
    } catch (error) {
        showError('网络错误: ' + error.message);
    }
}

// 渲染日志表格
function renderLogsTable(logs) {
    const tbody = document.querySelector('#logsTable tbody');
    tbody.innerHTML = '';
    
    if (!logs || logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: #666;">暂无数据</td></tr>';
        return;
    }
    
    logs.forEach(log => {
        const row = document.createElement('tr');
        row.className = 'log-entry';
        row.innerHTML = `
            <td>${log.id}</td>
            <td>${log.task ? log.task.name : '-'}</td>
            <td>${log.database}</td>
            <td>${log.table}</td>
            <td>${log.event_type}</td>
            <td><span class="status-badge status-${log.status}">${getStatusText(log.status)}</span></td>
            <td>${formatDateTime(log.created_at)}</td>
            <td>
                <button class="btn btn-small btn-secondary" onclick="viewLogDetail(${log.id})">详情</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// 加载系统状态
async function loadSystemStatus() {
    try {
        const response = await fetch('/api/status');
        const result = await response.json();
        
        if (response.ok) {
            document.getElementById('activeTasksCount').textContent = result.data.active_tasks;
            document.getElementById('systemStatus').textContent = result.data.status === 'running' ? '运行中' : '停止';
            document.getElementById('systemVersion').textContent = result.data.version;
            
            // 更新状态指示器
            const statusDot = document.querySelector('.status-dot');
            if (result.data.status === 'running') {
                statusDot.classList.add('active');
            } else {
                statusDot.classList.remove('active');
            }
        } else {
            showError('加载系统状态失败: ' + result.error);
        }
    } catch (error) {
        showError('网络错误: ' + error.message);
    }
}

// 加载任务列表用于过滤器
async function loadTasksForFilter() {
    try {
        const response = await fetch('/api/tasks?page=1&page_size=100');
        const result = await response.json();
        
        if (response.ok) {
            const select = document.getElementById('taskFilter');
            select.innerHTML = '<option value="">所有任务</option>';
            
            result.data.tasks.forEach(task => {
                const option = document.createElement('option');
                option.value = task.id;
                option.textContent = task.name;
                select.appendChild(option);
            });
        }
    } catch (error) {
        console.error('加载任务过滤器失败:', error);
    }
}

// 显示创建任务模态框
function showCreateTaskModal() {
    document.getElementById('createTaskModal').style.display = 'block';
    document.getElementById('createTaskForm').reset();
}

// 隐藏创建任务模态框
function hideCreateTaskModal() {
    document.getElementById('createTaskModal').style.display = 'none';
}

// 创建任务
async function createTask() {
    const form = document.getElementById('createTaskForm');
    const formData = new FormData(form);
    
    // 获取选中的事件类型
    const eventTypes = [];
    document.querySelectorAll('input[type="checkbox"]:checked').forEach(cb => {
        eventTypes.push(cb.value);
    });
    
    const taskData = {
        name: formData.get('name'),
        database: formData.get('database'),
        table: formData.get('table'),
        event_types: eventTypes.join(','),
        callback_url: formData.get('callback_url')
    };
    
    try {
        const response = await fetch('/api/tasks', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(taskData)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            hideCreateTaskModal();
            loadTasks();
            showSuccess('任务创建成功');
        } else {
            showError('创建任务失败: ' + result.error);
        }
    } catch (error) {
        showError('网络错误: ' + error.message);
    }
}

// 删除任务
async function deleteTask(id) {
    if (!confirm('确定要删除这个任务吗？')) {
        return;
    }
    
    try {
        const response = await fetch(`/api/tasks/${id}`, {
            method: 'DELETE'
        });
        
        const result = await response.json();
        
        if (response.ok) {
            loadTasks();
            showSuccess('任务删除成功');
        } else {
            showError('删除任务失败: ' + result.error);
        }
    } catch (error) {
        showError('网络错误: ' + error.message);
    }
}

// 编辑任务
async function editTask(id) {
    console.log('editTask called with id:', id);
    try {
        // 获取任务详细信息
        const response = await fetch(`/api/tasks/${id}`);
        const result = await response.json();
        
        if (!response.ok) {
            showError('获取任务信息失败: ' + result.error);
            return;
        }
        
        // 显示编辑表单
        showEditTaskForm(result.data);
    } catch (error) {
        showError('网络错误: ' + error.message);
    }
}

// 渲染分页
function renderPagination(containerId, currentPage, totalPages, loadFunction) {
    const container = document.getElementById(containerId);
    container.innerHTML = '';
    
    if (totalPages <= 1) return;
    
    // 上一页按钮
    const prevBtn = document.createElement('button');
    prevBtn.textContent = '上一页';
    prevBtn.disabled = currentPage === 1;
    prevBtn.onclick = () => loadFunction(currentPage - 1);
    container.appendChild(prevBtn);
    
    // 页码按钮
    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);
    
    for (let i = startPage; i <= endPage; i++) {
        const pageBtn = document.createElement('button');
        pageBtn.textContent = i;
        pageBtn.className = i === currentPage ? 'active' : '';
        pageBtn.onclick = () => loadFunction(i);
        container.appendChild(pageBtn);
    }
    
    // 下一页按钮
    const nextBtn = document.createElement('button');
    nextBtn.textContent = '下一页';
    nextBtn.disabled = currentPage === totalPages;
    nextBtn.onclick = () => loadFunction(currentPage + 1);
    container.appendChild(nextBtn);
}

// 工具函数
function truncateUrl(url, maxLength = 30) {
    return url.length > maxLength ? url.substring(0, maxLength) + '...' : url;
}

function getStatusText(status) {
    const statusMap = {
        'active': '活跃',
        'inactive': '停用',
        'pending': '等待中',
        'success': '成功',
        'failed': '失败'
    };
    return statusMap[status] || status;
}

function formatDateTime(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN');
}

function showSuccess(message) {
    showNotification(message, 'success');
}

function showError(message) {
    showNotification(message, 'error');
}

function showInfo(message) {
    showNotification(message, 'info');
}

function showNotification(message, type) {
    // 简单的通知实现
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        border-radius: 8px;
        color: white;
        font-weight: 500;
        z-index: 10000;
        animation: slideIn 0.3s ease;
    `;
    
    switch(type) {
        case 'success':
            notification.style.backgroundColor = '#27ae60';
            break;
        case 'error':
            notification.style.backgroundColor = '#e74c3c';
            break;
        case 'info':
            notification.style.backgroundColor = '#3498db';
            break;
    }
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

// 点击模态框外部关闭
window.onclick = function(event) {
    const modal = document.getElementById('createTaskModal');
    if (event.target === modal) {
        hideCreateTaskModal();
    }
}

// 添加CSS动画
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    
    .url-text {
        font-family: monospace;
        font-size: 12px;
    }
`;
document.head.appendChild(style);

// Binlog 监控功能
function loadBinlogInfo() {
    fetch('/api/binlog/info')
        .then(response => response.json())
        .then(data => {
            if (data.data) {
                document.getElementById('binlogStatus').textContent = data.data.log_bin || '-';
                document.getElementById('binlogFormat').textContent = data.data.binlog_format || '-';
                document.getElementById('currentFile').textContent = data.data.current_file || '-';
                document.getElementById('currentPos').textContent = data.data.current_pos || '-';
            }
        })
        .catch(error => {
            console.error('获取binlog信息失败:', error);
            showNotification('获取binlog信息失败', 'error');
        });
}

// 性能指标监控功能
function loadMetrics() {
    fetch('/api/metrics')
        .then(response => response.json())
        .then(data => {
            if (data.data) {
                // document.getElementById('eventsProcessed').textContent =
                //     data.data.events_processed || '-';
                // document.getElementById('eventsPerSecond').textContent =
                //     data.data.events_per_second || '-';
                // document.getElementById('errorRate').textContent =
                //     (data.data.error_rate * 100).toFixed(2) + '%' || '-';
                // document.getElementById('uptimeSeconds').textContent =
                //     Math.floor(data.data.uptime_seconds) || '-';
                
                // 更新架构信息
                if (data.data.architecture) {
                    document.getElementById('architectureType').textContent = data.data.architecture;
                }
                
                // 更新Canal状态详情
                if (data.data.canal_status) {
                    const canalStatus = data.data.canal_status;
                    
                    // 连接池信息
                    if (canalStatus.connection_pool) {
                        const pool = canalStatus.connection_pool;
                        document.getElementById('connectionPoolStatus').textContent =
                            `${pool.available}/${pool.max_size} (由${pool.managed_by}管理)`;
                    }
                    
                    // 实例数量
                    if (canalStatus.instance_count !== undefined) {
                        document.getElementById('instanceCount').textContent = canalStatus.instance_count;
                    }
                    
                    // 内存使用
                    if (canalStatus.memory_usage) {
                        const memory = canalStatus.memory_usage;
                        document.getElementById('memoryUsage').textContent =
                            `${memory.instances}个实例 (${memory.status})`;
                    }
                    
                    // 运行状态
                    document.getElementById('runningStatus').textContent =
                        canalStatus.running ? '运行中' : '已停止';
                        
                    // 更新实例详情表
                    console.log('准备更新实例详情表:', canalStatus.instances);
                    if (canalStatus.instances) {
                        updateInstancesTable(canalStatus.instances);
                    } else {
                        console.log('没有实例数据可更新');
                    }
                }
            }
        })
        .catch(error => {
            console.error('获取性能指标失败:', error);
            showNotification('获取性能指标失败', 'error');
        });
}

// 更新实例详情表
function updateInstancesTable(instances) {
    console.log('更新实例详情表:', instances);
    const tableBody = document.getElementById('instancesTableBody');
    tableBody.innerHTML = '';
    
    // 检查是否有实例数据
    if (!instances || Object.keys(instances).length === 0) {
        console.log('没有实例数据');
        return;
    }
    
    for (const [id, instance] of Object.entries(instances)) {
        console.log('处理实例:', id, instance);
        const row = document.createElement('tr');
        
        // 处理binlog位置
        let positionText = '-';
        if (instance.position) {
            positionText = `${instance.position.name}:${instance.position.pos}`;
        }
        
        // 处理最后事件时间
        let lastEventText = '-';
        if (instance.last_event) {
            try {
                // 尝试解析日期
                const date = new Date(instance.last_event);
                if (!isNaN(date.getTime())) {
                    lastEventText = date.toLocaleString('zh-CN');
                }
            } catch (e) {
                console.error('日期解析错误:', e);
                lastEventText = instance.last_event;
            }
        }
        
        row.innerHTML = `
            <td>${id}</td>
            <td>${instance.running ? '运行中' : '已停止'}</td>
            <td>${positionText}</td>
            <td>${lastEventText}</td>
        `;
        tableBody.appendChild(row);
    }
}
// 自动刷新监控数据
function startMonitoring() {
    // 每30秒自动刷新一次监控数据
    setInterval(() => {
        const activeTab = document.querySelector('.tab-btn.active').dataset.tab;
        // if (activeTab === 'binlog') {
        //     loadBinlogInfo();
        // } else
        if (activeTab === 'metrics') {
            loadMetrics();
        }
    }, 30000);
}

// 页面加载完成后启动监控
document.addEventListener('DOMContentLoaded', function() {
    startMonitoring();
});

// 显示编辑任务表单
function showEditTaskForm(task) {
    // 创建模态框
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content">
            <span class="close">&times;</span>
            <h2>编辑任务</h2>
            <form id="editTaskForm">
                <input type="hidden" id="editTaskId" value="${task.id}">
                <div class="form-group">
                    <label for="editTaskName">任务名称:</label>
                    <input type="text" id="editTaskName" value="${task.name}" required>
                </div>
                <div class="form-group">
                    <label for="editTaskDatabase">数据库:</label>
                    <input type="text" id="editTaskDatabase" value="${task.database}" required>
                </div>
                <div class="form-group">
                    <label for="editTaskTable">表名:</label>
                    <input type="text" id="editTaskTable" value="${task.table}" required>
                </div>
                <div class="form-group">
                    <label for="editTaskEventTypes">事件类型:</label>
                    <input type="text" id="editTaskEventTypes" value="${task.event_types}" required>
                </div>
                <div class="form-group">
                    <label for="editTaskCallbackURL">回调URL:</label>
                    <input type="text" id="editTaskCallbackURL" value="${task.callback_url}" required>
                </div>
                <div class="form-group">
                    <label for="editTaskStatus">状态:</label>
                    <select id="editTaskStatus">
                        <option value="active" ${task.status === 'active' ? 'selected' : ''}>活跃</option>
                        <option value="inactive" ${task.status === 'inactive' ? 'selected' : ''}>非活跃</option>
                    </select>
                </div>
                <button type="submit">保存</button>
            </form>
        </div>
    `;
    
    // 添加到页面
    document.body.appendChild(modal);
    
    // 显示模态框
    modal.style.display = 'block';
    
    // 关闭模态框
    const closeBtn = modal.querySelector('.close');
    closeBtn.onclick = function() {
        document.body.removeChild(modal);
    };
    
    // 点击模态框外部关闭
    window.onclick = function(event) {
        if (event.target === modal) {
            document.body.removeChild(modal);
        }
    };
    
    // 处理表单提交
    const form = modal.querySelector('#editTaskForm');
    form.onsubmit = async function(e) {
        e.preventDefault();
        
        const taskId = document.getElementById('editTaskId').value;
        const taskData = {
            name: document.getElementById('editTaskName').value,
            database: document.getElementById('editTaskDatabase').value,
            table: document.getElementById('editTaskTable').value,
            event_types: document.getElementById('editTaskEventTypes').value,
            callback_url: document.getElementById('editTaskCallbackURL').value,
            status: document.getElementById('editTaskStatus').value
        };
        
        try {
            const response = await fetch(`/api/tasks/${taskId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(taskData)
            });
            
            const result = await response.json();
            
            if (response.ok) {
                showSuccess('任务更新成功');
                document.body.removeChild(modal);
                loadTasks(); // 重新加载任务列表
            } else {
                showError('更新任务失败: ' + result.error);
            }
        } catch (error) {
            showError('网络错误: ' + error.message);
        }
    };
}

// 显示日志详情模态框
function showLogDetailModal(log) {
    // 创建模态框
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content">
            <span class="close">&times;</span>
            <h2>日志详情</h2>
            <div class="log-detail">
                <p><strong>ID:</strong> ${log.id}</p>
                <p><strong>任务ID:</strong> ${log.task_id}</p>
                <p><strong>数据库:</strong> ${log.database}</p>
                <p><strong>表名:</strong> ${log.table}</p>
                <p><strong>事件类型:</strong> ${log.event_type}</p>
                <p><strong>状态:</strong> ${log.status}</p>
                <p><strong>创建时间:</strong> ${new Date(log.created_at).toLocaleString()}</p>
                <p><strong>错误信息:</strong> ${log.error || '无'}</p>
                <div class="form-group">
                    <label for="logData">数据:</label>
                    <textarea id="logData" readonly>${log.data}</textarea>
                </div>
            </div>
        </div>
    `;
    
    // 添加到页面
    document.body.appendChild(modal);
    
    // 关闭模态框
    const closeBtn = modal.querySelector('.close');
    closeBtn.onclick = function() {
        document.body.removeChild(modal);
    };
    
    // 点击模态框外部关闭
    window.onclick = function(event) {
        if (event.target === modal) {
            document.body.removeChild(modal);
        }
    };
}

// 查看日志详情
function viewLogDetail(id) {
    fetch('/api/logs/' + id)
        .then(response => response.json())
        .then(data => {
            if (data.data) {
                showLogDetailModal(data.data);
            } else {
                showNotification('获取日志详情失败', 'error');
            }
        })
        .catch(error => {
            console.error('获取日志详情失败:', error);
            showNotification('获取日志详情失败: ' + error.message, 'error');
        });
}

// 显示日志详情模态框
function showLogDetailModal(log) {
    // 填充日志详情数据
    document.getElementById('logDetailId').textContent = log.id;
    document.getElementById('logDetailTaskName').textContent = log.task.name;
    document.getElementById('logDetailDatabase').textContent = log.database;
    document.getElementById('logDetailTable').textContent = log.table;
    document.getElementById('logDetailEventType').textContent = log.event_type;
    
    // 设置状态标签
    const statusElement = document.getElementById('logDetailStatus');
    statusElement.textContent = log.status;
    statusElement.className = 'status-tag status-' + log.status;
    
    document.getElementById('logDetailCreatedAt').textContent = new Date(log.created_at).toLocaleString('zh-CN');
    document.getElementById('logDetailData').textContent = log.data;
    
    // 处理错误信息显示
    const errorGroup = document.getElementById('logDetailErrorGroup');
    const errorElement = document.getElementById('logDetailError');
    if (log.error && log.error.trim() !== '') {
        errorElement.textContent = log.error;
        errorGroup.style.display = 'block';
    } else {
        errorGroup.style.display = 'none';
    }
    
    // 显示模态框
    document.getElementById('logDetailModal').style.display = 'block';
}

// 隐藏日志详情模态框
function hideLogDetailModal() {
    document.getElementById('logDetailModal').style.display = 'none';
}