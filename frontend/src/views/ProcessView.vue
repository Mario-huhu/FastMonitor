<template>
  <div class="process-view">
    <!-- 顶部统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: linear-gradient(135deg, #61afef 0%, #4596d9 100%);">
              <el-icon :size="24"><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">活跃进程</div>
              <div class="stat-value">{{ processTotal }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: linear-gradient(135deg, #98c379 0%, #7eb368 100%);">
              <el-icon :size="24"><Upload /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">总发送流量</div>
              <div class="stat-value">{{ formatBytes(totalSent) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: linear-gradient(135deg, #e5c07b 0%, #d5ad65 100%);">
              <el-icon :size="24"><Download /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">总接收流量</div>
              <div class="stat-value">{{ formatBytes(totalRecv) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: linear-gradient(135deg, #c678dd 0%, #b562cc 100%);">
              <el-icon :size="24"><Connection /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">总连接数</div>
              <div class="stat-value">{{ totalConnections }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 标签页 -->
    <el-card shadow="hover" style="margin-top: 20px;">
      <el-tabs v-model="activeTab">
        <!-- Top 10 流量排名 -->
        <el-tab-pane label="Top 10 流量排名" name="top10">
          <div class="tab-header">
            <el-button type="primary" size="small" @click="loadTopProcesses" :icon="Refresh">
              刷新
            </el-button>
          </div>
          
          <el-table
            :data="topProcesses"
            style="width: 100%"
            stripe
            :default-sort="{ prop: 'bytes_sent', order: 'descending' }"
          >
            <el-table-column type="index" label="#" width="40" />
            <el-table-column prop="pid" label="PID" width="60" sortable />
            <el-table-column prop="name" label="进程名" width="120" sortable show-overflow-tooltip>
              <template #default="{ row }">
                <span class="process-name-text">{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="exe" label="可执行文件" min-width="200" show-overflow-tooltip />
            <el-table-column prop="username" label="用户" width="80" sortable show-overflow-tooltip />
            <el-table-column prop="bytes_sent" label="发送" width="85" sortable align="right">
              <template #default="{ row }">
                <span class="bytes-clickable bytes-sent" @click="showPackets(row, 'sent')">
                  {{ formatBytes(row.bytes_sent) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="bytes_recv" label="接收" width="85" sortable align="right">
              <template #default="{ row }">
                <span class="bytes-clickable bytes-recv" @click="showPackets(row, 'recv')">
                  {{ formatBytes(row.bytes_recv) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="总流量" width="85" sortable align="right" :sort-by="row => row.bytes_sent + row.bytes_recv">
              <template #default="{ row }">
                <span class="bytes-clickable bytes-total" @click="showPackets(row, 'all')">
                  {{ formatBytes(row.bytes_sent + row.bytes_recv) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="connections" label="连接" width="60" sortable align="right" />
          </el-table>
        </el-tab-pane>

        <!-- 所有进程列表 -->
        <el-tab-pane label="所有进程监控" name="all">
          <div class="tab-header">
            <el-button type="danger" size="small" @click="clearStats" :icon="Delete">
              清空统计
            </el-button>
            <el-button type="primary" size="small" @click="loadAllProcesses" :icon="Refresh">
              刷新
            </el-button>
          </div>
          
          <el-table
            :data="allProcesses"
            style="width: 100%"
            stripe
            v-loading="loading"
            :default-sort="{ prop: 'last_seen', order: 'descending' }"
            :expand-row-keys="expandedRows"
            row-key="pid"
          >
            <el-table-column type="expand">
              <template #default="{ row }">
                <div class="process-detail">
                  <el-descriptions :column="2" border size="small">
                    <el-descriptions-item label="进程ID (PID)">{{ row.pid }}</el-descriptions-item>
                    <el-descriptions-item label="进程名称">{{ row.name }}</el-descriptions-item>
                    <el-descriptions-item label="可执行文件" :span="2">
                      <el-text size="small" truncated style="max-width: 600px;" :title="row.exe">
                        {{ row.exe || '未知' }}
                      </el-text>
                    </el-descriptions-item>
                    <el-descriptions-item label="用户名">{{ row.username || '未知' }}</el-descriptions-item>
                    <el-descriptions-item label="连接数">{{ row.connections }}</el-descriptions-item>
                    <el-descriptions-item label="发送数据包">{{ row.packets_sent.toLocaleString() }}</el-descriptions-item>
                    <el-descriptions-item label="接收数据包">{{ row.packets_recv.toLocaleString() }}</el-descriptions-item>
                    <el-descriptions-item label="发送流量">
                      <span style="color: #e5c07b; font-weight: 600;">{{ formatBytes(row.bytes_sent) }}</span>
                    </el-descriptions-item>
                    <el-descriptions-item label="接收流量">
                      <span style="color: #61afef; font-weight: 600;">{{ formatBytes(row.bytes_recv) }}</span>
                    </el-descriptions-item>
                    <el-descriptions-item label="总流量">
                      <span style="color: #98c379; font-weight: 700; font-size: 14px;">
                        {{ formatBytes(row.bytes_sent + row.bytes_recv) }}
                      </span>
                    </el-descriptions-item>
                    <el-descriptions-item label="首次活动">{{ formatTimestamp(row.first_seen) }}</el-descriptions-item>
                    <el-descriptions-item label="最后活动">{{ formatTimestamp(row.last_seen) }}</el-descriptions-item>
                    <el-descriptions-item label="活动时长">
                      {{ calcDuration(row.first_seen, row.last_seen) }}
                    </el-descriptions-item>
                  </el-descriptions>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="pid" label="PID" width="60" sortable />
            <el-table-column prop="name" label="进程名" width="120" sortable show-overflow-tooltip>
              <template #default="{ row }">
                <span class="process-name-text">{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="exe" label="可执行文件" min-width="180" show-overflow-tooltip />
            <el-table-column prop="username" label="用户" width="80" sortable show-overflow-tooltip />
            <el-table-column prop="packets_sent" label="发送包" width="75" sortable align="right">
              <template #default="{ row }">
                <span class="bytes-clickable" @click="showPackets(row, 'sent')">
                  {{ row.packets_sent.toLocaleString() }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="packets_recv" label="接收包" width="75" sortable align="right">
              <template #default="{ row }">
                <span class="bytes-clickable" @click="showPackets(row, 'recv')">
                  {{ row.packets_recv.toLocaleString() }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="bytes_sent" label="发送" width="80" sortable align="right">
              <template #default="{ row }">
                <span class="bytes-clickable bytes-sent" @click="showPackets(row, 'sent')">
                  {{ formatBytes(row.bytes_sent) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="bytes_recv" label="接收" width="80" sortable align="right">
              <template #default="{ row }">
                <span class="bytes-clickable bytes-recv" @click="showPackets(row, 'recv')">
                  {{ formatBytes(row.bytes_recv) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="connections" label="连接" width="55" sortable align="right" />
            <el-table-column prop="last_seen" label="最后活动" width="85" sortable>
              <template #default="{ row }">
                {{ formatShortTime(row.last_seen) }}
              </template>
            </el-table-column>
          </el-table>
          
          <!-- 分页 -->
          <div class="pagination">
            <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :page-sizes="[20, 50, 100]"
              :total="processTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadAllProcesses"
              @current-change="loadAllProcesses"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    
    <!-- 数据包详情弹窗 -->
    <el-dialog
      v-model="packetDialogVisible"
      :title="packetDialogTitle"
      width="800px"
      @close="stopPacketAutoRefresh"
    >
      <template #header>
        <div class="dialog-header">
          <span class="dialog-title">{{ packetDialogTitle }}</span>
          <div class="dialog-actions">
            <el-button 
              :type="packetAutoRefresh ? 'primary' : 'default'" 
              size="small" 
              @click="togglePacketAutoRefresh"
              :icon="packetAutoRefresh ? VideoPause : Refresh"
            >
              {{ packetAutoRefresh ? '停止刷新' : '自动刷新' }}
            </el-button>
          </div>
        </div>
      </template>
      <div v-if="selectedPackets.length === 0" class="no-packets">
        <el-empty description="暂无缓存的数据包记录" />
        <p class="hint">数据包缓存仅保留最近10条记录，需要在抓包过程中产生新的数据包</p>
      </div>
      <el-table v-else :data="filteredPackets" stripe size="small" max-height="400">
        <el-table-column prop="timestamp" label="时间" width="160">
          <template #default="{ row }">
            {{ formatPacketTime(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column label="方向" width="60" align="center">
          <template #default="{ row }">
            <el-tag :type="row.is_sent ? 'warning' : 'primary'" size="small">
              {{ row.is_sent ? '发送' : '接收' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="protocol" label="协议" width="60" />
        <el-table-column label="源地址" min-width="150">
          <template #default="{ row }">
            {{ row.src_ip }}:{{ row.src_port }}
          </template>
        </el-table-column>
        <el-table-column label="目标地址" min-width="150">
          <template #default="{ row }">
            {{ row.dst_ip }}:{{ row.dst_port }}
          </template>
        </el-table-column>
        <el-table-column prop="size" label="大小" width="80" align="right">
          <template #default="{ row }">
            {{ formatBytes(row.size) }}
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Monitor, Upload, Download, Connection, TrendCharts, List, Refresh, Delete, VideoPause 
} from '@element-plus/icons-vue'
import { GetProcessStats, GetTopProcessesByTraffic, ClearProcessStats, GetProcessPackets } from '../../wailsjs/go/server/App'

const topProcesses = ref<any[]>([])
const allProcesses = ref<any[]>([])
const processTotal = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const expandedRows = ref<number[]>([])
const activeTab = ref('top10')

// 数据包弹窗相关
const packetDialogVisible = ref(false)
const packetDialogTitle = ref('')
const selectedPackets = ref<any[]>([])
const packetFilter = ref<'all' | 'sent' | 'recv'>('all')
const packetAutoRefresh = ref(false)
const currentProcessExe = ref('')

let refreshTimer: any = null
let packetRefreshTimer: any = null

// 计算总统计
const totalSent = computed(() => {
  return allProcesses.value.reduce((sum, p) => sum + (p.bytes_sent || 0), 0)
})

const totalRecv = computed(() => {
  return allProcesses.value.reduce((sum, p) => sum + (p.bytes_recv || 0), 0)
})

const totalConnections = computed(() => {
  return allProcesses.value.reduce((sum, p) => sum + (p.connections || 0), 0)
})

// 过滤后的数据包
const filteredPackets = computed(() => {
  if (packetFilter.value === 'all') {
    return selectedPackets.value
  }
  return selectedPackets.value.filter(p => 
    packetFilter.value === 'sent' ? p.is_sent : !p.is_sent
  )
})

onMounted(() => {
  loadTopProcesses()
  loadAllProcesses()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

async function loadTopProcesses() {
  try {
    const result = await GetTopProcessesByTraffic(10)
    topProcesses.value = result || []
  } catch (error) {
    console.error('Load top processes failed:', error)
    ElMessage.error(`加载Top进程失败: ${error}`)
  }
}

async function loadAllProcesses() {
  loading.value = true
  try {
    const result = await GetProcessStats(currentPage.value, pageSize.value)
    allProcesses.value = result.data || []
    processTotal.value = result.total || 0
  } catch (error) {
    console.error('Load all processes failed:', error)
    ElMessage.error(`加载进程列表失败: ${error}`)
  } finally {
    loading.value = false
  }
}

async function clearStats() {
  try {
    await ElMessageBox.confirm(
      '确定要清空所有进程统计数据吗？此操作不可恢复！',
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    await ClearProcessStats()
    ElMessage.success('进程统计已清空')
    
    // 刷新数据
    topProcesses.value = []
    allProcesses.value = []
    processTotal.value = 0
    loadTopProcesses()
    loadAllProcesses()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(`清空失败: ${error}`)
    }
  }
}

// 显示数据包详情
function showPackets(row: any, filter: 'all' | 'sent' | 'recv') {
  const packets = row.recent_packets || []
  selectedPackets.value = packets
  packetFilter.value = filter
  currentProcessExe.value = row.exe
  
  const filterText = filter === 'sent' ? '发送' : filter === 'recv' ? '接收' : '全部'
  packetDialogTitle.value = `${row.name} - 最近${filterText}数据包 (缓存${packets.length}条)`
  packetDialogVisible.value = true
}

// 切换数据包自动刷新
function togglePacketAutoRefresh() {
  packetAutoRefresh.value = !packetAutoRefresh.value
  
  if (packetAutoRefresh.value) {
    startPacketAutoRefresh()
  } else {
    stopPacketAutoRefresh()
  }
}

// 开始数据包自动刷新
function startPacketAutoRefresh() {
  if (packetRefreshTimer) return
  
  packetRefreshTimer = setInterval(async () => {
    if (!currentProcessExe.value || !packetDialogVisible.value) {
      stopPacketAutoRefresh()
      return
    }
    
    try {
      const packets = await GetProcessPackets(currentProcessExe.value)
      if (packets) {
        selectedPackets.value = packets
        // 更新标题中的数量
        const filterText = packetFilter.value === 'sent' ? '发送' : packetFilter.value === 'recv' ? '接收' : '全部'
        const processName = packetDialogTitle.value.split(' - ')[0]
        packetDialogTitle.value = `${processName} - 最近${filterText}数据包 (缓存${packets.length}条)`
      }
    } catch (error) {
      console.error('Refresh packets failed:', error)
    }
  }, 1000) // 每秒刷新
}

// 停止数据包自动刷新
function stopPacketAutoRefresh() {
  if (packetRefreshTimer) {
    clearInterval(packetRefreshTimer)
    packetRefreshTimer = null
  }
  packetAutoRefresh.value = false
}

// 格式化数据包时间
function formatPacketTime(timestamp: any): string {
  if (!timestamp) return '-'
  
  let date: Date
  if (typeof timestamp === 'string') {
    date = new Date(timestamp)
  } else if (typeof timestamp === 'number') {
    date = new Date(timestamp * 1000)
  } else {
    return '-'
  }
  
  if (isNaN(date.getTime())) return '-'
  
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

function startAutoRefresh() {
  refreshTimer = setInterval(() => {
    loadTopProcesses()
    loadAllProcesses()
  }, 5000) // 每5秒刷新一次
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

function formatTimestamp(timestamp: any): string {
  if (!timestamp) return '-'
  
  // 处理不同格式的时间戳
  let date: Date
  if (typeof timestamp === 'string') {
    // ISO 字符串格式 (Go time.Time 序列化后的格式)
    date = new Date(timestamp)
  } else if (typeof timestamp === 'number') {
    // Unix 时间戳（秒）
    date = new Date(timestamp * 1000)
  } else {
    return '-'
  }
  
  // 检查日期是否有效
  if (isNaN(date.getTime()) || date.getFullYear() < 2000) {
    return '-'
  }
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatShortTime(timestamp: any): string {
  if (!timestamp) return '-'
  
  // 处理不同格式的时间戳
  let date: Date
  if (typeof timestamp === 'string') {
    date = new Date(timestamp)
  } else if (typeof timestamp === 'number') {
    date = new Date(timestamp * 1000)
  } else {
    return '-'
  }
  
  // 检查日期是否有效
  if (isNaN(date.getTime()) || date.getFullYear() < 2000) {
    return '-'
  }
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '时前'
  
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

function formatDuration(value: any): string {
  // 处理两个时间戳的差值
  let seconds: number
  
  if (typeof value === 'object' && value !== null) {
    // 如果传入的是两个时间对象
    return '-'
  } else if (typeof value === 'number') {
    seconds = value
  } else {
    return '-'
  }
  
  if (seconds < 0 || isNaN(seconds)) seconds = 0
  
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  
  const parts = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}小时`)
  if (minutes > 0) parts.push(`${minutes}分钟`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs}秒`)
  
  return parts.join(' ')
}

// 计算两个时间戳之间的时长
function calcDuration(start: any, end: any): string {
  if (!start || !end) return '-'
  
  let startDate: Date, endDate: Date
  
  if (typeof start === 'string') {
    startDate = new Date(start)
  } else if (typeof start === 'number') {
    startDate = new Date(start * 1000)
  } else {
    return '-'
  }
  
  if (typeof end === 'string') {
    endDate = new Date(end)
  } else if (typeof end === 'number') {
    endDate = new Date(end * 1000)
  } else {
    return '-'
  }
  
  if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
    return '-'
  }
  
  const seconds = Math.floor((endDate.getTime() - startDate.getTime()) / 1000)
  return formatDuration(seconds)
}
</script>

<style scoped lang="scss">
.process-view {
  padding: 16px;
  height: 100%;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.process-detail {
  padding: 12px 16px;
  background: var(--el-fill-color-lighter);
  
  :deep(.el-descriptions) {
    background: var(--el-fill-color-blank);
    
    .el-descriptions__label {
      font-weight: 600;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      background: var(--el-fill-color-light);
    }
    
    .el-descriptions__content {
      font-size: 12px;
      color: var(--el-text-color-primary);
    }
  }
}

.stats-row {
  margin-bottom: 16px;
  flex-shrink: 0;
}

.stat-card {
  transition: all 0.3s ease;
  border-radius: 10px;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--el-box-shadow);
  }
  
  :deep(.el-card__body) {
    padding: 14px;
  }
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stat-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tab-header {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  justify-content: flex-end;
}

.pagination {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}

:deep(.el-card) {
  border-radius: 10px;
  border: 1px solid var(--el-border-color-light);
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  
  .el-card__body {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  
  .el-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;
    
    .el-tabs__content {
      flex: 1;
      overflow: hidden;
    }
    
    .el-tab-pane {
      height: 100%;
      display: flex;
      flex-direction: column;
    }
  }
}

:deep(.el-table) {
  flex: 1;
  --el-table-border-color: var(--el-border-color-lighter);
  background-color: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  font-size: 12px;
  
  .el-table__header {
    font-weight: 600;
    color: var(--el-text-color-secondary);
    background-color: var(--el-fill-color-light);
  }
  
  .el-table__row:hover {
    background-color: var(--el-fill-color-light);
  }
  
  .el-table__cell {
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 5px 0;
  }
}

.process-name-text {
  color: var(--el-color-success);
  font-weight: 500;
}

.bytes-sent {
  color: #e5c07b;
}

.bytes-recv {
  color: #61afef;
}

.bytes-total {
  color: #98c379;
  font-weight: 600;
}

.bytes-clickable {
  cursor: pointer;
  transition: all 0.2s;
  
  &:hover {
    text-decoration: underline;
    opacity: 0.8;
  }
}

.no-packets {
  text-align: center;
  padding: 20px;
  
  .hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    margin-top: 10px;
  }
}

.dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 40px;
  
  .dialog-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  
  .dialog-actions {
    display: flex;
    gap: 8px;
  }
}
</style>
