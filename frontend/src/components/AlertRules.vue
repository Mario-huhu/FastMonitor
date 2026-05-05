<template>
  <div class="alert-rules-container">
    <div class="table-header">
      <el-button type="primary" @click="showAddDialog">
        <el-icon><Plus /></el-icon>
        新建规则
      </el-button>
      <el-button :icon="Refresh" @click="loadData" :loading="loading">
        刷新
      </el-button>
      <el-button type="success" @click="exportRules">
        <el-icon><Download /></el-icon>
        导出规则
      </el-button>
      <el-button type="warning" @click="importRules">
        <el-icon><Upload /></el-icon>
        导入规则
      </el-button>
      <el-button type="danger" @click="resetBuiltinRules" :loading="resetting">
        <el-icon><RefreshRight /></el-icon>
        重置内置规则
      </el-button>
      <input 
        ref="fileInputRef" 
        type="file" 
        accept=".json" 
        style="display: none;" 
        @change="handleFileSelect"
      />
      <el-select 
        v-model="filterType" 
        placeholder="按类型筛选" 
        clearable 
        style="width: 150px; margin-left: 12px;"
        @change="handleFilterChange"
      >
        <el-option label="目标IP" value="dst_ip" />
        <el-option label="DNS" value="dns" />
        <el-option label="HTTP" value="http" />
        <el-option label="ICMP" value="icmp" />
        <el-option label="进程" value="process" />
      </el-select>
      <el-select 
        v-model="filterEnabled" 
        placeholder="按状态筛选" 
        clearable 
        style="width: 150px; margin-left: 12px;"
        @change="handleFilterChange"
      >
        <el-option label="启用" :value="true" />
        <el-option label="禁用" :value="false" />
      </el-select>
      <span class="rule-count">共 {{ total }} 条规则</span>
    </div>

    <el-table
      :data="tableData"
      height="calc(100vh - 400px)"
      stripe
      style="width: 100%"
      :expand-row-keys="expandedRows"
      row-key="id"
    >
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="rule-detail">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="规则ID">{{ row.id }}</el-descriptions-item>
              <el-descriptions-item label="规则名称">
                <el-tag type="primary">{{ row.name }}</el-tag>
              </el-descriptions-item>
              
              <el-descriptions-item label="规则类型">
                <el-tag :type="getRuleTypeColor(row.rule_type)">{{ getRuleTypeText(row.rule_type) }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="告警级别">
                <el-tag :type="getAlertType(row.alert_level)" effect="dark">
                  {{ getAlertLevelText(row.alert_level) }}
                </el-tag>
              </el-descriptions-item>
              
              <el-descriptions-item label="条件字段">
                {{ getConditionFieldText(row.condition_field) }}
              </el-descriptions-item>
              <el-descriptions-item label="匹配方式">
                {{ getOperatorText(row.condition_operator) }}
              </el-descriptions-item>
              
              <el-descriptions-item label="匹配值" :span="2">
                <el-tag type="warning">{{ row.condition_value }}</el-tag>
              </el-descriptions-item>
              
              <el-descriptions-item label="规则描述" :span="2">
                {{ row.description || '-' }}
              </el-descriptions-item>
              
              <el-descriptions-item label="联网检测" v-if="row.rule_type === 'process'">
                <el-tag :type="row.require_network ? 'warning' : 'info'">
                  {{ row.require_network ? '需要联网' : '离线检测' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="联网检测" v-else>
                <el-tag type="info">不适用</el-tag>
              </el-descriptions-item>
              
              <el-descriptions-item label="创建时间">
                {{ formatTime(row.created_at) }}
              </el-descriptions-item>
              <el-descriptions-item label="更新时间">
                {{ formatTime(row.updated_at) }}
              </el-descriptions-item>
              
              <el-descriptions-item label="启用状态" :span="2">
                <el-switch
                  v-model="row.enabled"
                  @change="toggleEnabled(row)"
                  active-text="启用"
                  inactive-text="禁用"
                />
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </template>
      </el-table-column>
      
      <el-table-column prop="name" label="规则名称" width="200" show-overflow-tooltip />
      
      <el-table-column label="规则类型" width="100">
        <template #default="{ row }">
          <el-tag :type="getRuleTypeColor(row.rule_type)" size="small">
            {{ getRuleTypeText(row.rule_type) }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column label="匹配条件" min-width="300">
        <template #default="{ row }">
          <span style="color: #409eff; font-weight: 500;">{{ getConditionFieldText(row.condition_field) }}</span>
          <span style="margin: 0 8px; color: #909399;">{{ getOperatorText(row.condition_operator) }}</span>
          <el-tag type="warning" size="small">{{ row.condition_value }}</el-tag>
        </template>
      </el-table-column>
      
      <el-table-column label="告警级别" width="100">
        <template #default="{ row }">
          <el-tag :type="getAlertType(row.alert_level)" effect="dark" size="small">
            {{ getAlertLevelText(row.alert_level) }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-switch
            v-model="row.enabled"
            @change="toggleEnabled(row)"
            inline-prompt
            active-text="启用"
            inactive-text="禁用"
          />
        </template>
      </el-table-column>
      
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">
          {{ formatShortTime(row.created_at) }}
        </template>
      </el-table-column>
      
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" size="small" link @click="editRule(row)">
            编辑
          </el-button>
          <el-button type="danger" size="small" link @click="deleteRule(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[20, 50, 100, 200]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="isEdit ? '编辑告警规则' : '新建告警规则'" 
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form 
        ref="ruleFormRef" 
        :model="ruleForm" 
        :rules="rules" 
        label-width="100px"
      >
        <el-form-item label="规则名称" prop="name">
          <el-input 
            v-model="ruleForm.name" 
            placeholder="请输入规则名称"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item label="规则类型" prop="rule_type">
          <el-select 
            v-model="ruleForm.rule_type" 
            placeholder="请选择规则类型"
            @change="handleRuleTypeChange"
            style="width: 100%;"
          >
            <el-option label="目标IP告警" value="dst_ip">
              <div style="display: flex; flex-direction: column;">
                <span>目标IP告警</span>
                <span style="font-size: 12px; color: #999;">监控特定目标IP的连接</span>
              </div>
            </el-option>
            <el-option label="DNS请求告警" value="dns">
              <div style="display: flex; flex-direction: column;">
                <span>DNS请求告警</span>
                <span style="font-size: 12px; color: #999;">监控DNS域名查询</span>
              </div>
            </el-option>
            <el-option label="HTTP请求告警" value="http">
              <div style="display: flex; flex-direction: column;">
                <span>HTTP请求告警</span>
                <span style="font-size: 12px; color: #999;">监控HTTP请求的域名或URL</span>
              </div>
            </el-option>
            <el-option label="ICMP告警" value="icmp">
              <div style="display: flex; flex-direction: column;">
                <span>ICMP告警</span>
                <span style="font-size: 12px; color: #999;">监控ICMP数据包（ping等）</span>
              </div>
            </el-option>
            <el-option label="进程告警" value="process">
              <div style="display: flex; flex-direction: column;">
                <span>进程告警</span>
                <span style="font-size: 12px; color: #999;">监控特定进程的网络活动</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        
        <el-form-item label="条件字段" prop="condition_field">
          <el-select v-model="ruleForm.condition_field" placeholder="请选择条件字段" style="width: 100%;">
            <el-option 
              v-for="field in getAvailableFields()" 
              :key="field.value" 
              :label="field.label" 
              :value="field.value"
            />
          </el-select>
        </el-form-item>
        
        <el-form-item label="匹配方式" prop="condition_operator">
          <el-select v-model="ruleForm.condition_operator" placeholder="请选择匹配方式" style="width: 100%;">
            <el-option label="精确匹配" value="equals">
              <span>精确匹配</span>
              <span style="margin-left: 8px; font-size: 12px; color: #999;">(完全相等)</span>
            </el-option>
            <el-option label="包含匹配" value="contains">
              <span>包含匹配</span>
              <span style="margin-left: 8px; font-size: 12px; color: #999;">(部分包含)</span>
            </el-option>
            <el-option label="正则表达式" value="regex">
              <span>正则表达式</span>
              <span style="margin-left: 8px; font-size: 12px; color: #999;">(高级匹配)</span>
            </el-option>
          </el-select>
        </el-form-item>
        
        <el-form-item label="匹配值" prop="condition_value">
          <el-input 
            v-model="ruleForm.condition_value" 
            type="textarea"
            :rows="4"
            placeholder="请输入匹配值（支持多行或逗号分隔，将自动创建多条规则）"
            maxlength="2000"
            show-word-limit
          />
          <div style="margin-top: 8px; display: flex; align-items: center; gap: 8px;">
            <el-button 
              v-if="ruleForm.condition_operator === 'regex'" 
              size="small" 
              @click="showRegexHelp"
            >
              正则帮助
            </el-button>
            <el-tag v-if="getValueCount() > 1" type="info" size="small">
              将创建 {{ getValueCount() }} 条规则
            </el-tag>
          </div>
          <div style="margin-top: 8px; font-size: 12px; color: #999;">
            <div v-if="ruleForm.rule_type === 'dst_ip'">示例: 192.168.1.1, 192.168.1.2 或每行一个IP</div>
            <div v-else-if="ruleForm.rule_type === 'dns'">示例: baidu.com, qq.com 或 .*\.cn$ (正则)</div>
            <div v-else-if="ruleForm.rule_type === 'http'">示例: example.com, example.org 或 /api/login</div>
            <div v-else-if="ruleForm.rule_type === 'process'">示例: chrome.exe, firefox.exe 或每行一个</div>
            <div style="margin-top: 4px; color: #e6a23c;">
              💡 提示：支持换行或逗号分隔，批量创建多条规则
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="告警级别" prop="alert_level">
          <el-radio-group v-model="ruleForm.alert_level">
            <el-radio label="critical">
              <el-tag type="danger" effect="dark">严重</el-tag>
            </el-radio>
            <el-radio label="error">
              <el-tag type="danger">错误</el-tag>
            </el-radio>
            <el-radio label="warning">
              <el-tag type="warning">警告</el-tag>
            </el-radio>
            <el-radio label="info">
              <el-tag type="info">信息</el-tag>
            </el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item label="规则描述" prop="description">
          <el-input 
            v-model="ruleForm.description" 
            type="textarea" 
            :rows="3"
            placeholder="请输入规则描述（选填）"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item label="联网检测" v-if="ruleForm.rule_type === 'process'">
          <el-switch 
            v-model="ruleForm.require_network" 
            active-text="需要联网" 
            inactive-text="离线检测"
            :active-value="true"
            :inactive-value="false"
          />
          <div style="margin-top: 8px; font-size: 12px; color: #999;">
            <div>• 需要联网：只有在进程产生网络流量时才触发告警</div>
            <div>• 离线检测：只要进程运行就触发告警（不推荐，可能产生大量误报）</div>
          </div>
        </el-form-item>
        
        <el-form-item label="启用状态">
          <el-switch v-model="ruleForm.enabled" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">
          {{ isEdit ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 正则帮助对话框 -->
    <el-dialog v-model="regexHelpVisible" title="正则表达式帮助" width="600px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="匹配.cn域名">.*\\.cn$</el-descriptions-item>
        <el-descriptions-item label="匹配可执行文件">.*\\.(exe|msi|dmg)$</el-descriptions-item>
        <el-descriptions-item label="匹配压缩文件">.*\\.(zip|rar|7z|tar|gz)$</el-descriptions-item>
        <el-descriptions-item label="匹配IP段">^192\\.168\\..*</el-descriptions-item>
        <el-descriptions-item label="匹配特定端口">.*:80$ 或 .*:443$</el-descriptions-item>
      </el-descriptions>
      <div style="margin-top: 16px; padding: 12px; background: #f5f7fa; border-radius: 4px;">
        <div style="font-weight: bold; margin-bottom: 8px;">常用符号：</div>
        <div>. = 任意字符 | * = 0次或多次 | + = 1次或多次</div>
        <div>^ = 开头 | $ = 结尾 | \\ = 转义字符</div>
        <div>| = 或 | [abc] = a、b或c | [0-9] = 数字</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Download, Upload, RefreshRight } from '@element-plus/icons-vue'
import { 
  QueryAlertRules, 
  CreateAlertRule, 
  UpdateAlertRule, 
  DeleteAlertRule,
  ResetBuiltinRules
} from '../../wailsjs/go/server/App'

const loading = ref(false)
const submitting = ref(false)
const resetting = ref(false)
const dialogVisible = ref(false)
const regexHelpVisible = ref(false)
const isEdit = ref(false)
const tableData = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(50)
const ruleFormRef = ref()
const expandedRows = ref<number[]>([])
const fileInputRef = ref<HTMLInputElement>()

// 筛选条件
const filterType = ref('')
const filterEnabled = ref<boolean | undefined>(undefined)

// 自动刷新
let autoRefreshTimer: number | null = null
const AUTO_REFRESH_INTERVAL = 30000 // 30秒

const ruleForm = reactive({
  id: 0,
  name: '',
  rule_type: '',
  condition_field: '',
  condition_operator: 'equals',
  condition_value: '',
  alert_level: 'warning',
  description: '',
  enabled: true,
  require_network: false
})

const rules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  rule_type: [{ required: true, message: '请选择规则类型', trigger: 'change' }],
  condition_field: [{ required: true, message: '请选择条件字段', trigger: 'change' }],
  condition_operator: [{ required: true, message: '请选择匹配方式', trigger: 'change' }],
  condition_value: [{ required: true, message: '请输入匹配值', trigger: 'blur' }],
  alert_level: [{ required: true, message: '请选择告警级别', trigger: 'change' }]
}

onMounted(() => {
  loadData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

// 暴露刷新方法给父组件
defineExpose({
  refresh: loadData
})

async function loadData() {
  try {
    loading.value = true
    const query = {
      rule_type: filterType.value || undefined,
      enabled: filterEnabled.value,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }
    
    const result = await QueryAlertRules(query)
    tableData.value = result.data || []
    total.value = result.total || 0
  } catch (error) {
    console.error('加载规则列表失败:', error)
    ElMessage.error('加载规则列表失败')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(ruleForm, {
    id: 0,
    name: '',
    rule_type: '',
    condition_field: '',
    condition_operator: 'equals',
    condition_value: '',
    alert_level: 'warning',
    description: '',
    enabled: true,
    require_network: false
  })
  
  ruleFormRef.value?.clearValidate()
}

function showAddDialog() {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

function editRule(row: any) {
  isEdit.value = true
  Object.assign(ruleForm, row)
  dialogVisible.value = true
}

// 重置内置规则
async function resetBuiltinRules() {
  try {
    await ElMessageBox.confirm(
      '确定要重置内置规则吗？这将删除所有银狐病毒相关的内置规则并重新安装最新版本。',
      '确认重置',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    resetting.value = true
    await ResetBuiltinRules()
    ElMessage.success('内置规则重置成功')
    await loadData() // 重新加载数据
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('重置内置规则失败:', error)
      ElMessage.error('重置内置规则失败: ' + (error?.message || error))
    }
  } finally {
    resetting.value = false
  }
}

// 解析匹配值，支持多行或逗号分隔
function parseConditionValues(valueString: string): string[] {
  if (!valueString || !valueString.trim()) return []
  
  // 先按换行分割
  const lines = valueString.split('\n')
  const values: string[] = []
  
  for (const line of lines) {
    const trimmedLine = line.trim()
    if (!trimmedLine) continue
    
    // 如果包含逗号，进一步分割
    if (trimmedLine.includes(',')) {
      const parts = trimmedLine.split(',').map(p => p.trim()).filter(p => p)
      values.push(...parts)
    } else {
      values.push(trimmedLine)
    }
  }
  
  // 去重
  return [...new Set(values)]
}

// 获取将要创建的规则数量
function getValueCount(): number {
  const values = parseConditionValues(ruleForm.condition_value)
  return Math.max(values.length, 1)
}

async function submitForm() {
  try {
    await ruleFormRef.value.validate()
    submitting.value = true
    
    if (isEdit.value) {
      // 编辑模式：不支持批量，直接更新
      await UpdateAlertRule(ruleForm)
      ElMessage.success('规则更新成功')
    } else {
      // 新建模式：解析匹配值，支持批量创建
      const values = parseConditionValues(ruleForm.condition_value)
      
      if (values.length === 0) {
        ElMessage.error('请输入匹配值')
        return
      }
      
      if (values.length === 1) {
        // 单条规则
        await CreateAlertRule({ ...ruleForm, condition_value: values[0] })
        ElMessage.success('规则创建成功')
      } else {
        // 批量创建
        let successCount = 0
        let failCount = 0
        
        for (const value of values) {
          try {
            await CreateAlertRule({
              ...ruleForm,
              name: `${ruleForm.name} - ${value}`,
              condition_value: value
            })
            successCount++
          } catch (error) {
            console.error(`创建规则失败 (${value}):`, error)
            failCount++
          }
        }
        
        if (successCount > 0) {
          ElMessage.success(`成功创建 ${successCount} 条规则` + (failCount > 0 ? `，失败 ${failCount} 条` : ''))
        } else {
          ElMessage.error('批量创建失败')
        }
      }
    }
    
    dialogVisible.value = false
    loadData()
  } catch (error) {
    if (error !== false) {
      console.error('保存规则失败:', error)
      ElMessage.error('保存规则失败')
    }
  } finally {
    submitting.value = false
  }
}

async function deleteRule(row: any) {
  try {
    await ElMessageBox.confirm('确定要删除此规则吗？', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await DeleteAlertRule(row.id)
    ElMessage.success('规则删除成功')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除规则失败:', error)
      ElMessage.error('删除规则失败')
    }
  }
}

async function toggleEnabled(row: any) {
  try {
    await UpdateAlertRule(row)
    ElMessage.success(row.enabled ? '规则已启用' : '规则已禁用')
  } catch (error) {
    console.error('更新状态失败:', error)
    ElMessage.error('更新状态失败')
    row.enabled = !row.enabled
  }
}


function handleRuleTypeChange(value: string) {
  // 根据规则类型自动设置默认字段
  const defaultFields: any = {
    dst_ip: 'dst_ip',
    dns: 'domain',
    http: 'domain',
    icmp: 'dst_ip',
    process: 'process_name'
  }
  ruleForm.condition_field = defaultFields[value] || ''
}

function getAvailableFields() {
  const fieldMap: any = {
    dst_ip: [
      { label: '目标IP', value: 'dst_ip' }
    ],
    dns: [
      { label: '域名', value: 'domain' }
    ],
    http: [
      { label: '域名', value: 'domain' },
      { label: 'URL', value: 'url' }
    ],
    icmp: [
      { label: '源IP', value: 'src_ip' },
      { label: '目标IP', value: 'dst_ip' }
    ],
    process: [
      { label: '进程名称', value: 'process_name' },
      { label: '进程路径', value: 'process_exe' },
      { label: '进程PID', value: 'process_pid' }
    ]
  }
  return fieldMap[ruleForm.rule_type] || []
}

function showRegexHelp() {
  regexHelpVisible.value = true
}

// 导出规则为JSON
async function exportRules() {
  try {
    // 获取所有规则（不分页）
    const allRules = await QueryAlertRules({
      page: 1,
      page_size: 10000,
      rule_type: '',
      enabled: undefined
    })
    
    if (!allRules.rules || allRules.rules.length === 0) {
      ElMessage.warning('没有可导出的规则')
      return
    }
    
    // 移除id、created_at、updated_at等后端字段
    const exportData = allRules.rules.map((rule: any) => ({
      name: rule.name,
      rule_type: rule.rule_type,
      condition_field: rule.condition_field,
      condition_operator: rule.condition_operator,
      condition_value: rule.condition_value,
      alert_level: rule.alert_level,
      description: rule.description,
      enabled: rule.enabled
    }))
    
    // 创建Blob并下载
    const jsonStr = JSON.stringify(exportData, null, 2)
    const blob = new Blob([jsonStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `alert_rules_${new Date().getTime()}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    
    ElMessage.success(`已导出 ${exportData.length} 条规则`)
  } catch (error) {
    console.error('导出规则失败:', error)
    ElMessage.error('导出规则失败')
  }
}

// 触发文件选择
function importRules() {
  fileInputRef.value?.click()
}

// 处理文件选择
async function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  
  if (!file) return
  
  try {
    const text = await file.text()
    const rules = JSON.parse(text)
    
    if (!Array.isArray(rules)) {
      ElMessage.error('JSON格式错误：期望一个数组')
      return
    }
    
    // 确认导入
    await ElMessageBox.confirm(
      `即将导入 ${rules.length} 条规则，是否继续？`,
      '导入确认',
      {
        confirmButtonText: '导入',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    let successCount = 0
    let failCount = 0
    
    for (const rule of rules) {
      try {
        // 验证必填字段
        if (!rule.name || !rule.rule_type || !rule.condition_field || 
            !rule.condition_operator || !rule.condition_value || !rule.alert_level) {
          failCount++
          continue
        }
        
        await CreateAlertRule(rule)
        successCount++
      } catch (error) {
        console.error('导入规则失败:', error)
        failCount++
      }
    }
    
    if (successCount > 0) {
      ElMessage.success(`成功导入 ${successCount} 条规则` + (failCount > 0 ? `，失败 ${failCount} 条` : ''))
      loadData()
    } else {
      ElMessage.error('导入失败')
    }
  } catch (error) {
    console.error('导入规则失败:', error)
    ElMessage.error('导入规则失败：' + (error instanceof Error ? error.message : '未知错误'))
  } finally {
    // 清空input值，允许重复选择同一文件
    target.value = ''
  }
}

function handleFilterChange() {
  currentPage.value = 1
  loadData()
}

function handlePageChange() {
  loadData()
}

function handleSizeChange() {
  currentPage.value = 1
  loadData()
}

function startAutoRefresh() {
  stopAutoRefresh()
  autoRefreshTimer = window.setInterval(() => {
    loadData()
  }, AUTO_REFRESH_INTERVAL)
}

function stopAutoRefresh() {
  if (autoRefreshTimer !== null) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
}

function getRuleTypeText(type: string) {
  const texts: any = {
    dst_ip: '目标IP',
    dns: 'DNS',
    http: 'HTTP',
    icmp: 'ICMP',
    process: '进程'
  }
  return texts[type] || type
}

function getRuleTypeColor(type: string) {
  const colors: any = {
    dst_ip: 'primary',
    dns: 'success',
    http: 'warning',
    icmp: 'danger',
    process: 'info'
  }
  return colors[type] || ''
}

function getConditionFieldText(field: string) {
  const texts: any = {
    dst_ip: '目标IP',
    src_ip: '源IP',
    domain: '域名',
    url: 'URL',
    process_name: '进程名称',
    process_exe: '进程路径',
    process_pid: '进程PID'
  }
  return texts[field] || field
}

function getOperatorText(operator: string) {
  const texts: any = {
    equals: '等于',
    contains: '包含',
    regex: '正则匹配'
  }
  return texts[operator] || operator
}

function getAlertType(level: string) {
  const types: any = {
    critical: 'danger',
    error: 'danger',
    warning: 'warning',
    info: 'info'
  }
  return types[level] || ''
}

function getAlertLevelText(level: string) {
  const texts: any = {
    critical: '严重',
    error: '错误',
    warning: '警告',
    info: '信息'
  }
  return texts[level] || level
}

function formatTime(timestamp: string) {
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

function formatShortTime(timestamp: string) {
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
  }
  
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}
</script>

<style scoped>
.alert-rules-container {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.table-header {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

.rule-count {
  margin-left: auto;
  font-size: 14px;
  color: #909399;
}

.rule-detail {
  padding: 20px;
  background: var(--el-fill-color-light);
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

:deep(.el-table__expanded-cell) {
  padding: 0;
}

:deep(.el-descriptions__label) {
  font-weight: 600;
}
</style>
