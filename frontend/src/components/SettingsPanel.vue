<template>
  <div class="settings-panel">
    <el-row :gutter="16">
      <el-col :span="12">
        <el-card header="地图起点设置" shadow="never" style="margin-bottom: 16px;">
          <el-form :model="mapOrigin" label-width="100px" size="small">
            <el-form-item label="选择城市">
              <el-select
                v-model="selectedCity"
                placeholder="选择预设城市"
                filterable
                @change="handleCityChange"
                style="width: 100%;"
              >
                <el-option
                  v-for="city in presetCities"
                  :key="city.name"
                  :label="city.name"
                  :value="city.name"
                >
                  <span>{{ city.name }}</span>
                  <span style="float: right; color: var(--el-text-color-secondary); font-size: 11px;">
                    {{ city.lat.toFixed(4) }}, {{ city.lng.toFixed(4) }}
                  </span>
                </el-option>
              </el-select>
            </el-form-item>
            
            <el-form-item label="城市名称">
              <el-input v-model="mapOrigin.name" placeholder="自定义城市名称" />
            </el-form-item>
            
            <el-form-item label="纬度">
              <el-input-number
                v-model="mapOrigin.lat"
                :min="-90"
                :max="90"
                :precision="4"
                :step="0.1"
                style="width: 100%;"
              />
            </el-form-item>
            
            <el-form-item label="经度">
              <el-input-number
                v-model="mapOrigin.lng"
                :min="-180"
                :max="180"
                :precision="4"
                :step="0.1"
                style="width: 100%;"
              />
            </el-form-item>
            
            <el-form-item>
              <el-button type="primary" @click="saveMapOrigin" size="small">
                保存地图起点
              </el-button>
              <el-button @click="resetMapOrigin" size="small">重置默认</el-button>
            </el-form-item>
            
            <el-alert
              type="info"
              :closable="false"
              show-icon
              style="margin-top: 8px;"
            >
              <template #title>
                <span style="font-size: 11px;">
                  地图起点用于威胁地图和3D地球的连线起点，设置为您的位置可更直观地查看网络流量方向
                </span>
              </template>
            </el-alert>
          </el-form>
        </el-card>
        
        <el-card header="环形缓冲区上限" shadow="never">
          <el-form :model="limits" label-width="140px" size="small">
            <el-form-item label="原始数据包最大数量">
              <el-input-number v-model="limits.raw_max" :min="1000" :max="100000" :step="1000" />
            </el-form-item>
            <el-form-item label="DNS会话最大数量">
              <el-input-number v-model="limits.dns_max" :min="1000" :max="50000" :step="1000" />
            </el-form-item>
            <el-form-item label="HTTP会话最大数量">
              <el-input-number v-model="limits.http_max" :min="1000" :max="50000" :step="1000" />
            </el-form-item>
            <el-form-item label="ICMP会话最大数量">
              <el-input-number v-model="limits.icmp_max" :min="1000" :max="50000" :step="1000" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveLimits" :loading="saving">
                应用更改
              </el-button>
              <el-button @click="resetLimits">重置默认</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
      
      <el-col :span="12">
        <el-card header="存储统计" shadow="never">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="总数据包">
              {{ stats.raw_count?.toLocaleString() || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="DNS会话">
              {{ stats.dns_count?.toLocaleString() || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="HTTP会话">
              {{ stats.http_count?.toLocaleString() || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="ICMP会话">
              {{ stats.icmp_count?.toLocaleString() || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="存储大小">
              {{ formatBytes(stats.total_size || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="PCAP文件数">
              {{ stats.pcap_file_count || 0 }}
            </el-descriptions-item>
          </el-descriptions>

          <el-button
            type="warning"
            size="small"
            style="margin-top: 16px;"
            @click="runVacuum"
            :loading="vacuuming"
          >
            运行存储清理
          </el-button>
        </el-card>
        
        <el-card header="关于" shadow="never" style="margin-top: 16px;">
          <p><strong>网络抓包分析器</strong></p>
          <p>基于 Wails + Go + Vue3 构建的网络数据包捕获分析工具。</p>
          <p class="info-text">
            <el-icon><InfoFilled /></el-icon>
            需要管理员/root权限才能捕获数据包
          </p>
          <p class="info-text">
            <el-icon><Warning /></el-icon>
            仅捕获明文 HTTP，不包括 HTTPS
          </p>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled, Warning } from '@element-plus/icons-vue'
import { GetLimits, UpdateLimits, GetStorageStats, VacuumStorage } from '../../wailsjs/go/server/App'
import { mapOriginPoint, updateMapOrigin, resetMapOrigin as resetOrigin, PRESET_CITIES, DEFAULT_ORIGIN } from '../stores/mapConfig'
import type { MapOriginPoint } from '../stores/mapConfig'

const emit = defineEmits(['config-updated'])

// 地图起点配置
const mapOrigin = ref<MapOriginPoint>({ ...mapOriginPoint.value })
const selectedCity = ref(mapOriginPoint.value.name)
const presetCities = ref(PRESET_CITIES)

// 处理城市选择变化
const handleCityChange = (cityName: string) => {
  const city = presetCities.value.find(c => c.name === cityName)
  if (city) {
    mapOrigin.value = { ...city }
  }
}

// 保存地图起点
const saveMapOrigin = () => {
  updateMapOrigin(mapOrigin.value)
  ElMessage.success('地图起点已保存')
  emit('config-updated')
}

// 重置地图起点
const resetMapOrigin = () => {
  resetOrigin()
  mapOrigin.value = { ...DEFAULT_ORIGIN }
  selectedCity.value = DEFAULT_ORIGIN.name
  ElMessage.success('已重置为默认起点')
}

const limits = ref({
  raw_max: 20000,
  dns_max: 5000,
  http_max: 5000,
  icmp_max: 5000,
  session_flow_max: 5000
})

const stats = ref<any>({})
const saving = ref(false)
const vacuuming = ref(false)

onMounted(async () => {
  await loadLimits()
  await loadStats()
})

async function loadLimits() {
  try {
    limits.value = await GetLimits()
  } catch (error) {
    ElMessage.error('加载配置失败: ' + error)
  }
}

async function loadStats() {
  try {
    stats.value = await GetStorageStats()
  } catch (error) {
    console.error('加载统计失败:', error)
  }
}

async function saveLimits() {
  try {
    saving.value = true
    await UpdateLimits(limits.value)
    ElMessage.success('设置已保存')
    emit('config-updated')
  } catch (error) {
    ElMessage.error('保存设置失败: ' + error)
  } finally {
    saving.value = false
  }
}

function resetLimits() {
  limits.value = {
    raw_max: 20000,
    dns_max: 5000,
    http_max: 5000,
    icmp_max: 5000
  }
}

async function runVacuum() {
  try {
    vacuuming.value = true
    await VacuumStorage()
    await loadStats()
    ElMessage.success('存储清理完成')
  } catch (error) {
    ElMessage.error('清理失败: ' + error)
  } finally {
    vacuuming.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}
</script>

<style scoped lang="scss">
.settings-panel {
  padding: 16px;
  height: 100%;
  overflow-y: auto;
  
  :deep(.el-card) {
    border-radius: 8px;
    height: fit-content;
    
    .el-card__header {
      font-size: 14px;
      font-weight: 600;
      padding: 10px 14px;
    }
    
    .el-card__body {
      padding: 14px;
    }
  }
  
  :deep(.el-form-item) {
    margin-bottom: 14px;
    
    .el-form-item__label {
      font-size: 12px;
    }
  }
  
  :deep(.el-descriptions__label) {
    font-size: 12px;
    width: 80px;
  }
  
  :deep(.el-descriptions__content) {
    font-size: 12px;
  }
  
  p {
    font-size: 12px;
    line-height: 1.6;
    margin-bottom: 6px;
  }
  
  .info-text {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    display: flex;
    align-items: center;
    gap: 4px;
  }
}
</style>
