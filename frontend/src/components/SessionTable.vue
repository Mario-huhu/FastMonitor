<template>
  <div class="session-table">
    <div class="table-header">
      <el-button :icon="Refresh" @click="$emit('refresh')" :loading="loading">
        刷新
      </el-button>
      <span class="session-count">共 {{ total }} 个会话</span>
    </div>

        <!-- DNS 表格 -->
        <el-table
          v-if="table === 'dns'"
          :data="data"
          stripe
          style="width: 100%"
          :expand-row-keys="expandedRows"
          row-key="id"
          @sort-change="handleSortChange"
          :default-sort="{ prop: 'timestamp', order: 'descending' }"
        >
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="session-detail">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="会话ID">{{ row.id }}</el-descriptions-item>
              <el-descriptions-item label="时间">{{ formatTimestamp(row.timestamp) }}</el-descriptions-item>
              <el-descriptions-item label="源地址">{{ row.five_tuple.src_ip }}:{{ row.five_tuple.src_port }}</el-descriptions-item>
              <el-descriptions-item label="目标地址">{{ row.five_tuple.dst_ip }}:{{ row.five_tuple.dst_port }}</el-descriptions-item>
              <el-descriptions-item label="协议">{{ row.five_tuple.protocol }}</el-descriptions-item>
              <el-descriptions-item label="域名">{{ row.domain }}</el-descriptions-item>
              <el-descriptions-item label="查询类型">{{ row.query_type }}</el-descriptions-item>
              <el-descriptions-item label="响应IP">{{ row.response_ip || '无' }}</el-descriptions-item>
              <el-descriptions-item label="数据大小">{{ row.payload_size }} 字节</el-descriptions-item>
              <el-descriptions-item label="过期时间">{{ formatTimestamp(row.ttl) }}</el-descriptions-item>
              <el-descriptions-item label="进程名称">{{ row.process_name || '未关联' }}</el-descriptions-item>
              <el-descriptions-item label="进程PID">{{ row.process_pid || '-' }}</el-descriptions-item>
              <el-descriptions-item label="进程路径" :span="2">{{ row.process_exe || '-' }}</el-descriptions-item>
            </el-descriptions>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="timestamp" label="时间" width="90" sortable="custom">
        <template #default="{ row }">
          {{ formatShortTimestamp(row.timestamp) }}
        </template>
      </el-table-column>
      <el-table-column prop="src_ip" label="源IP" width="130" show-overflow-tooltip sortable="custom" />
      <el-table-column prop="domain" label="域名" min-width="280" show-overflow-tooltip sortable="custom" />
      <el-table-column prop="query_type" label="查询类型" width="130" sortable="custom">
        <template #default="{ row }">
          <el-tag size="small">{{ row.query_type }}</el-tag>
          <span class="type-desc">{{ getDNSTypeDescription(row.query_type) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="response_ip" label="响应IP" width="130" show-overflow-tooltip sortable="custom" />
      <el-table-column prop="payload_size" label="大小" width="80" sortable="custom">
        <template #default="{ row }">
          {{ formatBytes(row.payload_size) }}
        </template>
      </el-table-column>
      <el-table-column prop="process_name" label="进程" min-width="100" sortable="custom">
        <template #default="{ row }">
          <span v-if="row.process_name" class="process-name">{{ row.process_name }}</span>
          <span v-else class="process-none">-</span>
        </template>
      </el-table-column>
    </el-table>

        <!-- HTTP 表格 -->
        <el-table
          v-else-if="table === 'http'"
          :data="data"
          stripe
          style="width: 100%"
          :expand-row-keys="expandedRows"
          row-key="id"
          @sort-change="handleSortChange"
          :default-sort="{ prop: 'timestamp', order: 'descending' }"
        >
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="session-detail">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="会话ID">{{ row.id }}</el-descriptions-item>
              <el-descriptions-item label="时间">{{ formatTimestamp(row.timestamp) }}</el-descriptions-item>
              <el-descriptions-item label="源地址">{{ row.five_tuple.src_ip }}:{{ row.five_tuple.src_port }}</el-descriptions-item>
              <el-descriptions-item label="目标地址">{{ row.five_tuple.dst_ip }}:{{ row.five_tuple.dst_port }}</el-descriptions-item>
              <el-descriptions-item label="协议">{{ row.five_tuple.protocol }}</el-descriptions-item>
              <el-descriptions-item label="请求方法">{{ row.method }}</el-descriptions-item>
              <el-descriptions-item label="主机" :span="2">{{ row.host }}</el-descriptions-item>
              <el-descriptions-item label="路径" :span="2">{{ row.path }}</el-descriptions-item>
              <el-descriptions-item label="状态码">{{ row.status_code || '无' }}</el-descriptions-item>
              <el-descriptions-item label="Content-Type">{{ row.content_type || '无' }}</el-descriptions-item>
              <el-descriptions-item label="数据大小">{{ formatBytes(row.payload_size) }}</el-descriptions-item>
              <el-descriptions-item label="过期时间">{{ formatShortTimestamp(row.ttl) }}</el-descriptions-item>
              <el-descriptions-item label="User-Agent" :span="2">{{ row.user_agent || '无' }}</el-descriptions-item>
              <el-descriptions-item label="进程名称">{{ row.process_name || '未关联' }}</el-descriptions-item>
              <el-descriptions-item label="进程PID">{{ row.process_pid || '-' }}</el-descriptions-item>
              <el-descriptions-item label="进程路径" :span="2">{{ row.process_exe || '-' }}</el-descriptions-item>
              <el-descriptions-item v-if="row.post_data" label="POST数据" :span="2">
                <div class="post-data-container">
                  <div class="post-data-info">
                    <span>{{ getPostDataInfo(row.post_data) }}</span>
                    <el-button 
                      v-if="!isOldBinaryHint(row.post_data)" 
                      size="small" 
                      text 
                      type="primary" 
                      @click="copyPostData(row.post_data)"
                    >复制</el-button>
                  </div>
                  <el-input 
                    v-if="!isOldBinaryHint(row.post_data)"
                    type="textarea" 
                    :value="formatPostData(row.post_data)" 
                    :rows="4" 
                    readonly 
                    class="post-data-textarea"
                  />
                  <div v-else class="binary-hint">
                    <el-icon><Warning /></el-icon>
                    <span>{{ row.post_data }}</span>
                    <p class="binary-hint-tip">提示：重新抓包后可查看二进制数据的十六进制内容</p>
                  </div>
                </div>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="timestamp" label="时间" width="90" sortable="custom">
        <template #default="{ row }">
          {{ formatShortTimestamp(row.timestamp) }}
        </template>
      </el-table-column>
      <el-table-column prop="method" label="方法" width="70" sortable="custom">
        <template #default="{ row }">
          <el-tag :type="getMethodType(row.method)" size="small">
            {{ row.method }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="host" label="主机" width="160" show-overflow-tooltip sortable="custom" />
      <el-table-column prop="path" label="路径" min-width="250" show-overflow-tooltip sortable="custom" />
      <el-table-column prop="status_code" label="状态" width="70" sortable="custom">
        <template #default="{ row }">
          <el-tag v-if="row.status_code" :type="getStatusType(row.status_code)" size="small">
            {{ row.status_code }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="payload_size" label="大小" width="80" sortable="custom">
        <template #default="{ row }">
          {{ formatBytes(row.payload_size) }}
        </template>
      </el-table-column>
      <el-table-column prop="process_name" label="进程" min-width="100" sortable="custom">
        <template #default="{ row }">
          <span v-if="row.process_name" class="process-name">{{ row.process_name }}</span>
          <span v-else class="process-none">-</span>
        </template>
      </el-table-column>
    </el-table>

        <!-- ICMP 表格 -->
        <el-table
          v-else-if="table === 'icmp'"
          :data="data"
          stripe
          style="width: 100%"
          :expand-row-keys="expandedRows"
          row-key="id"
          @sort-change="handleSortChange"
          :default-sort="{ prop: 'timestamp', order: 'descending' }"
        >
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="session-detail">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="会话ID">{{ row.id }}</el-descriptions-item>
              <el-descriptions-item label="时间">{{ formatTimestamp(row.timestamp) }}</el-descriptions-item>
              <el-descriptions-item label="源IP">{{ row.five_tuple.src_ip }}</el-descriptions-item>
              <el-descriptions-item label="目标IP">{{ row.five_tuple.dst_ip }}</el-descriptions-item>
              <el-descriptions-item label="协议">{{ row.five_tuple.protocol }}</el-descriptions-item>
              <el-descriptions-item label="ICMP类型">{{ row.icmp_type }} ({{ getICMPTypeName(row.icmp_type) }})</el-descriptions-item>
              <el-descriptions-item label="ICMP代码">{{ row.icmp_code }}</el-descriptions-item>
              <el-descriptions-item label="序列号">{{ row.icmp_seq }}</el-descriptions-item>
              <el-descriptions-item label="数据大小">{{ formatBytes(row.payload_size) }}</el-descriptions-item>
              <el-descriptions-item label="过期时间">{{ formatShortTimestamp(row.ttl) }}</el-descriptions-item>
              <el-descriptions-item label="进程名称">{{ row.process_name || '未关联' }}</el-descriptions-item>
              <el-descriptions-item label="进程PID">{{ row.process_pid || '-' }}</el-descriptions-item>
              <el-descriptions-item label="进程路径" :span="2">{{ row.process_exe || '-' }}</el-descriptions-item>
            </el-descriptions>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="timestamp" label="时间" width="90" sortable="custom">
        <template #default="{ row }">
          {{ formatShortTimestamp(row.timestamp) }}
        </template>
      </el-table-column>
      <el-table-column label="源IP" width="130" show-overflow-tooltip sortable="custom">
        <template #default="{ row }">
          {{ row.src_ip || row.five_tuple?.src_ip || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="目标IP" width="130" show-overflow-tooltip sortable="custom">
        <template #default="{ row }">
          {{ row.dst_ip || row.five_tuple?.dst_ip || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="icmp_type" label="类型" min-width="150" sortable="custom">
        <template #default="{ row }">
          <span class="icmp-type">{{ row.icmp_type }}</span>
          <span class="type-desc">{{ getICMPTypeDescription(row.icmp_type) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="icmp_code" label="代码" width="55" sortable="custom" />
      <el-table-column prop="icmp_seq" label="序列" width="60" sortable="custom" />
      <el-table-column prop="payload_size" label="大小" width="75" sortable="custom">
        <template #default="{ row }">
          {{ formatBytes(row.payload_size) }}
        </template>
      </el-table-column>
      <el-table-column prop="process_name" label="进程" min-width="90" sortable="custom">
        <template #default="{ row }">
          <span v-if="row.process_name" class="process-name">{{ row.process_name }}</span>
          <span v-else class="process-none">-</span>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        :current-page="currentPage"
        :page-size="pageSize"
        :page-sizes="[20, 50, 100, 200]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Connection, QuestionFilled, Warning } from '@element-plus/icons-vue'

const props = defineProps<{
  table: string
  data: any[]
  total: number
  loading: boolean
}>()

const emit = defineEmits(['refresh', 'page-change', 'size-change', 'sort-change'])

const currentPage = ref(1)
const pageSize = ref(50)
const expandedRows = ref<number[]>([])

// 监听数据变化，保持展开状态
watch(() => props.data, () => {
  // 数据更新时保持已展开的行
}, { deep: true })

function handlePageChange(page: number) {
  currentPage.value = page
  expandedRows.value = [] // 切换页面时清空展开状态
  emit('page-change', page)
}

function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  expandedRows.value = [] // 切换页大小时清空展开状态
  emit('size-change', size)
}

function handleSortChange({ prop, order }: any) {
  if (!prop) return
  
  // 转换为后端需要的格式
  const sortBy = prop
  const sortOrder = order === 'ascending' ? 'asc' : 'desc'
  
  // 重置到第一页并通知父组件
  currentPage.value = 1
  expandedRows.value = []
  emit('sort-change', { sortBy, sortOrder })
}

function formatTimestamp(timestamp: any): string {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

function formatShortTimestamp(timestamp: any): string {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()
  
  if (isToday) {
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    })
  } else {
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    })
  }
}

function getDNSTypeDescription(type: string): string {
  const descriptions: Record<string, string> = {
    'A': 'IPv4地址',
    'AAAA': 'IPv6地址',
    'CNAME': '别名记录',
    'MX': '邮件交换',
    'NS': '域名服务器',
    'PTR': '反向解析',
    'SOA': '授权起始',
    'TXT': '文本记录',
    'SRV': '服务记录',
    'ANY': '所有记录',
  }
  return descriptions[type] || ''
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

function getICMPTypeName(type: number): string {
  const types: Record<number, string> = {
    // ICMPv4 类型
    0: 'Echo Reply',
    3: 'Destination Unreachable',
    4: 'Source Quench',
    5: 'Redirect',
    8: 'Echo Request',
    9: 'Router Advertisement',
    10: 'Router Solicitation',
    11: 'Time Exceeded',
    12: 'Parameter Problem',
    13: 'Timestamp',
    14: 'Timestamp Reply',
    15: 'Information Request',
    16: 'Information Reply',
    17: 'Address Mask Request',
    18: 'Address Mask Reply',
    // ICMPv6 类型
    1: 'Destination Unreachable (v6)',
    2: 'Packet Too Big',
    128: 'Echo Request (v6)',
    129: 'Echo Reply (v6)',
    130: 'Multicast Listener Query',
    131: 'Multicast Listener Report',
    132: 'Multicast Listener Done',
    133: 'Router Solicitation',
    134: 'Router Advertisement',
    135: 'Neighbor Solicitation',
    136: 'Neighbor Advertisement',
    137: 'Redirect Message',
    143: 'MLDv2 Report'
  }
  return types[type] || `Type ${type}`
}

function getICMPTypeDescription(type: number): string {
  const descriptions: Record<number, string> = {
    // ICMPv4
    0: '回显应答',
    3: '目标不可达',
    4: '源端被关闭',
    5: '重定向',
    8: '回显请求',
    9: '路由器通告',
    10: '路由器请求',
    11: '超时',
    12: '参数问题',
    13: '时间戳请求',
    14: '时间戳应答',
    15: '信息请求',
    16: '信息应答',
    17: '地址掩码请求',
    18: '地址掩码应答',
    // ICMPv6
    1: '目标不可达(v6)',
    2: '包过大',
    128: '回显请求(v6)',
    129: '回显应答(v6)',
    130: '组播监听查询',
    131: '组播监听报告',
    132: '组播监听完成',
    133: '路由器请求',
    134: '路由器通告',
    135: '邻居请求',
    136: '邻居通告',
    137: '重定向消息',
    143: 'MLDv2报告'
  }
  return descriptions[type] || ''
}

function getMethodType(method: string): string {
  switch (method) {
    case 'GET':
      return 'success'
    case 'POST':
      return 'primary'
    case 'PUT':
      return 'warning'
    case 'DELETE':
      return 'danger'
    default:
      return 'info'
  }
}

function getStatusType(status: number): string {
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'info'
  if (status >= 400 && status < 500) return 'warning'
  if (status >= 500) return 'danger'
  return 'info'
}

// 检测是否为 Base64 编码的二进制数据
function isBase64Binary(data: string): boolean {
  return data && data.startsWith('[BASE64:')
}

// 检测是否为旧格式的二进制数据提示
function isOldBinaryHint(data: string): boolean {
  return data && data.startsWith('[二进制数据')
}

// 解析 Base64 数据
function parseBase64Data(data: string): { size: number, base64: string } | null {
  if (!data || !data.startsWith('[BASE64:')) return null
  const match = data.match(/^\[BASE64:(\d+)\](.+)$/)
  if (match) {
    return { size: parseInt(match[1]), base64: match[2] }
  }
  return null
}

// 获取 POST 数据信息
function getPostDataInfo(data: string): string {
  if (!data) return ''
  
  if (isBase64Binary(data)) {
    const parsed = parseBase64Data(data)
    if (parsed) {
      return `二进制数据 (${parsed.size} 字节)`
    }
  }
  
  if (isOldBinaryHint(data)) {
    return data.replace('[', '').replace(']', '')
  }
  
  return `文本数据 (${data.length} 字符)`
}

// 格式化 POST 数据显示
function formatPostData(data: string): string {
  if (!data) return ''
  
  // 处理 Base64 编码的二进制数据
  if (isBase64Binary(data)) {
    const parsed = parseBase64Data(data)
    if (parsed) {
      try {
        // 解码 Base64
        const binaryStr = atob(parsed.base64)
        const bytes: string[] = []
        const chars: string[] = []
        const maxLen = Math.min(binaryStr.length, 512)
        
        for (let i = 0; i < maxLen; i++) {
          const code = binaryStr.charCodeAt(i)
          bytes.push(code.toString(16).padStart(2, '0'))
          chars.push(code >= 32 && code <= 126 ? binaryStr[i] : '.')
        }
        
        // 格式化为每行16字节的十六进制显示
        let result = ''
        for (let i = 0; i < bytes.length; i += 16) {
          const hexPart = bytes.slice(i, i + 16).join(' ')
          const charPart = chars.slice(i, i + 16).join('')
          const offset = i.toString(16).padStart(4, '0')
          result += `${offset}: ${hexPart.padEnd(48)} | ${charPart}\n`
        }
        
        if (binaryStr.length > maxLen) {
          result += `\n... 还有 ${binaryStr.length - maxLen} 字节未显示`
        }
        
        return result
      } catch (e) {
        return `Base64 解码失败: ${e}`
      }
    }
  }
  
  // 旧格式的二进制数据提示
  if (isOldBinaryHint(data)) {
    return data
  }
  
  // 尝试解析 JSON 并格式化
  try {
    const parsed = JSON.parse(data)
    return JSON.stringify(parsed, null, 2)
  } catch {
    // 不是 JSON，尝试解析 URL 编码
    if (data.includes('=') && (data.includes('&') || data.includes('\n'))) {
      // 已经是格式化的键值对
      if (data.includes(': ')) {
        return data
      }
      try {
        const params = new URLSearchParams(data)
        let result = ''
        params.forEach((value, key) => {
          result += `${key}: ${decodeURIComponent(value)}\n`
        })
        return result.trim()
      } catch {
        // 解析失败，返回原始数据
      }
    }
  }
  
  return data
}

// 复制 POST 数据
function copyPostData(data: string) {
  if (!data) return
  
  // 如果是 Base64 编码，复制解码后的十六进制
  let copyText = data
  if (isBase64Binary(data)) {
    copyText = formatPostData(data)
  }
  
  navigator.clipboard.writeText(copyText).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}
</script>

<style scoped lang="scss">
.session-table {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  .table-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
    flex-shrink: 0;

    .session-count {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  :deep(.el-table) {
    flex: 1;
    
    .el-table__cell {
      padding: 6px 0;
    }
    
    .cell {
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }

  .session-detail {
    padding: 16px;
    background: var(--el-fill-color-light);
    border-radius: 6px;

    :deep(.el-descriptions__label) {
      width: 100px;
      font-size: 12px;
    }
    
    :deep(.el-descriptions__content) {
      font-size: 12px;
    }
  }

  .pagination {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
    flex-shrink: 0;
  }
}

.type-desc {
  margin-left: 6px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.icmp-type {
  color: #e6a23c;
  font-weight: 500;
}

.process-name {
  color: var(--el-color-success);
  font-size: 12px;
}

.process-none {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.post-data-container {
  width: 100%;
  
  .post-data-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
    font-size: 11px;
    color: var(--el-text-color-secondary);
  }
  
  .post-data-textarea {
    :deep(.el-textarea__inner) {
      font-family: 'SF Mono', Monaco, 'Courier New', monospace;
      font-size: 11px;
      line-height: 1.4;
      background: var(--el-fill-color-darker);
      color: var(--el-text-color-primary);
    }
  }
  
  .binary-hint {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
    padding: 12px;
    background: var(--el-fill-color-light);
    border-radius: 4px;
    border: 1px dashed var(--el-border-color);
    
    .el-icon {
      color: var(--el-color-warning);
      font-size: 16px;
    }
    
    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
    
    .binary-hint-tip {
      margin: 0;
      font-size: 11px;
      color: var(--el-text-color-placeholder);
    }
  }
}
</style>
