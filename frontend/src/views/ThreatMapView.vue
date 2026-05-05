<template>
  <div class="threat-map-page" ref="mapPageRef">
    <!-- 顶部工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <div class="title-section">
          <div class="pulse-dot"></div>
          <h1 class="page-title">网络流量威胁地图</h1>
        </div>
        
        <!-- 统计卡片 -->
        <div class="stats-cards">
          <div class="stat-card stat-cyan">
            <span class="stat-label">总连接</span>
            <span class="stat-value">{{ stats.totalSessions?.toLocaleString() || 0 }}</span>
          </div>
          <div class="stat-card stat-purple">
            <span class="stat-label">唯一IP</span>
            <span class="stat-value">{{ stats.uniqueIPs?.toLocaleString() || 0 }}</span>
          </div>
          <div class="stat-card stat-green">
            <span class="stat-label">国家/地区</span>
            <span class="stat-value">{{ countryStats.length }}</span>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="toolbar-right">
        <button @click="toggleMapType" class="action-btn btn-info">
          {{ mapType === 'world' ? '中国地图' : '世界地图' }}
        </button>

        <button @click="toggleDataSource" :class="['action-btn', dataSource === 'city' ? 'btn-warning active' : 'btn-secondary']">
          {{ dataSource === 'city' ? '城市' : '国家/地区' }}
        </button>
        
        <button @click="toggleShowLines" :class="['action-btn', showLines ? 'btn-primary active' : 'btn-secondary']">
          {{ showLines ? '隐藏连线' : '显示连线' }}
        </button>
        
        <button @click="refreshData" :disabled="loading" class="action-btn btn-success" :class="{ disabled: loading }">
          <span v-if="loading">刷新中...</span>
          <span v-else>刷新数据</span>
        </button>

        <button @click="toggleStatsPanel" class="action-btn btn-purple">
          {{ showStatsPanel ? '隐藏统计' : '显示统计' }}
        </button>

        <button @click="toggleFullscreen" class="action-btn btn-secondary">
          {{ isFullscreen ? '退出全屏' : '全屏' }}
        </button>
      </div>
    </div>

    <!-- 地图容器 -->
    <div class="map-container">
      <BaseMapComponent
        ref="mapRef"
        :data="mapPoints"
        :loading="loading"
        :show-lines="showLines"
        :map-region="mapType"
        :default-location="defaultLocation"
        @point-click="handlePointClick"
        @point-hover="handlePointHover"
      />
    </div>

    <!-- 右侧统计面板 -->
    <div v-if="showStatsPanel" class="stats-panel" :class="{ collapsed: isPanelCollapsed }">
      <!-- 面板头部 -->
      <div class="panel-header" @mousedown="handlePanelDragStart">
        <div class="panel-title">
          <svg width="16" height="16" fill="none" stroke="#a855f7" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span>统计面板</span>
        </div>
      </div>

      <!-- 面板内容 -->
      <div v-if="!isPanelCollapsed" class="panel-content">
        <!-- 标签导航 -->
        <div class="panel-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            @click="activeTab = tab.value"
            :class="['tab-btn', { active: activeTab === tab.value }, tab.value]"
          >
            <component :is="tab.icon" style="width: 12px; height: 12px; margin-bottom: 4px;" />
            <span>{{ tab.label }}</span>
          </button>
        </div>

        <!-- 标签内容 -->
        <div class="tab-content-area">
          <!-- 国家统计 -->
          <div v-if="activeTab === 'countries'" class="stats-list">
            <div
              v-for="(country, index) in countryStats.slice(0, 10)"
              :key="country.country"
              class="stat-item"
              @click="handleCountryClick(country)"
            >
              <div class="stat-item-header">
                <span class="stat-name">{{ country.country }}</span>
                <span class="stat-rank rank-countries">#{{ index + 1 }}</span>
              </div>
              <div class="stat-item-body">
                <div class="stat-detail">
                  <span class="label">连接:</span>
                  <span class="value">{{ country.connections?.toLocaleString() }}</span>
                </div>
                <div class="stat-detail">
                  <span class="label">IP:</span>
                  <span class="value">{{ country.uniqueIPs || country.ips?.length || 0 }}</span>
                </div>
              </div>
              <div v-if="country.cities && country.cities.length" class="stat-extra">
                城市: {{ country.cities.slice(0, 3).join(', ') }}
              </div>
            </div>
          </div>

          <!-- 城市统计 -->
          <div v-if="activeTab === 'cities'" class="stats-list">
            <div
              v-for="(city, index) in cityStats.slice(0, 10)"
              :key="`${city.country}-${city.city}`"
              class="stat-item"
              @click="handleCityClick(city)"
            >
              <div class="stat-item-header">
                <span class="stat-name">{{ city.city }}</span>
                <span class="stat-rank rank-cities">#{{ index + 1 }}</span>
              </div>
              <div class="stat-extra">{{ city.country }}</div>
              <div class="stat-item-body">
                <div class="stat-detail">
                  <span class="label">连接:</span>
                  <span class="value">{{ city.connections?.toLocaleString() }}</span>
                </div>
                <div class="stat-detail">
                  <span class="label">IP:</span>
                  <span class="value">{{ city.uniqueIPs || city.ips?.length || 0 }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 热门IP统计 -->
          <div v-if="activeTab === 'ips'" class="stats-list">
            <div
              v-for="(ip, index) in topIPs.slice(0, 10)"
              :key="ip.ip"
              class="stat-item"
              @click="handleIPClick(ip)"
            >
              <div class="stat-item-header">
                <span class="stat-name stat-ip">{{ ip.ip }}</span>
                <span class="stat-rank rank-ips">#{{ index + 1 }}</span>
              </div>
              <div class="stat-item-body">
                <div class="stat-detail">
                  <span class="label">连接数:</span>
                  <span class="value">{{ ip.count?.toLocaleString() }}</span>
                </div>
              </div>
              <div v-if="ip.country || ip.city" class="stat-extra">
                {{ ip.city ? `${ip.city}, ` : '' }}{{ ip.country }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import BaseMapComponent from '../components/maps/BaseMapComponent.vue'
import { mapDataManager, type MapDataCache } from '../services/MapDataManager'
import { mapOriginPoint } from '../stores/mapConfig'

// 图标组件 (使用h函数创建SVG)
import { h } from 'vue'

const UsersIcon = {
  render() {
    return h('svg', { class: 'w-full h-full', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z' })
    ])
  }
}

const NetworkIcon = {
  render() {
    return h('svg', { class: 'w-full h-full', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9' })
    ])
  }
}

// Refs
const mapRef = ref()
const loading = ref(false)
const showLines = ref(true)
const showStatsPanel = ref(true)
const isPanelCollapsed = ref(false)
const activeTab = ref<'countries' | 'cities' | 'ips'>('countries')
const mapType = ref<'world' | 'china'>('world')
const dataSource = ref<'country' | 'city'>('city')
const mapPageRef = ref<HTMLElement>()
const isFullscreen = ref(false)

// 地图数据
const mapData = ref<MapDataCache | null>(null)

// 统计数据
const stats = computed(() => ({
  totalSessions: mapData.value?.totalSessions || 0,
  uniqueIPs: mapData.value?.uniqueIPs || 0
}))

// 根据数据源选择地图点
const mapPoints = computed(() => {
  if (dataSource.value === 'city') {
    return mapData.value?.cityMapPoints || []
  } else {
    return mapData.value?.countryMapPoints || []
  }
})
const countryStats = computed(() => {
  const stats = mapData.value?.countryStats || []
  return stats.sort((a, b) => (b.connections || 0) - (a.connections || 0))
})
const cityStats = computed(() => {
  const stats = mapData.value?.cityStats || []
  return stats.sort((a, b) => (b.connections || 0) - (a.connections || 0))
})
const topIPs = computed(() => {
  const ips = mapData.value?.topIPs || []
  return ips.sort((a, b) => (b.count || 0) - (a.count || 0))
})

// 默认位置 - 使用持久化配置（东莞市）
const defaultLocation = computed(() => mapOriginPoint.value)

// 标签配置
const tabs = [
  {
    value: 'countries',
    label: '国家/地区',
    icon: UsersIcon,
    activeClass: 'bg-blue-600',
    borderClass: 'border-blue-400'
  },
  {
    value: 'cities',
    label: '城市',
    icon: UsersIcon,
    activeClass: 'bg-green-600',
    borderClass: 'border-green-400'
  },
  {
    value: 'ips',
    label: 'IP',
    icon: NetworkIcon,
    activeClass: 'bg-orange-600',
    borderClass: 'border-orange-400'
  }
]

// 面板拖拽 - 移除，改用固定右侧位置
const handlePanelDragStart = (e: MouseEvent) => {
  // 保留用于头部拖拽区域的视觉反馈
  e.preventDefault()
}

// 操作方法
const toggleMapType = () => {
  mapType.value = mapType.value === 'world' ? 'china' : 'world'
}

const toggleDataSource = () => {
  dataSource.value = dataSource.value === 'city' ? 'country' : 'city'
}

const toggleShowLines = () => {
  showLines.value = !showLines.value
}

const toggleStatsPanel = () => {
  showStatsPanel.value = !showStatsPanel.value
}

const refreshData = async () => {
  loading.value = true
  try {
    console.log('[ThreatMapView] 手动刷新数据...')
    mapData.value = await mapDataManager.updateMapData(true)
    console.log('[ThreatMapView] ✅ 地图数据刷新成功')
  } catch (error) {
    console.error('[ThreatMapView] ⚠️ 刷新数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 全屏功能
const toggleFullscreen = async () => {
  if (!mapPageRef.value) return
  
  if (!isFullscreen.value) {
    try {
      await mapPageRef.value.requestFullscreen()
      isFullscreen.value = true
    } catch (error) {
      console.error('[ThreatMapView] 全屏失败:', error)
    }
  } else {
    try {
      await document.exitFullscreen()
      isFullscreen.value = false
    } catch (error) {
      console.error('[ThreatMapView] 退出全屏失败:', error)
    }
  }
}

// 监听全屏变化
const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
}

// 事件处理
const handlePointClick = (data: any) => {
  console.log('[ThreatMapView] 点击地图点:', data)
}

const handlePointHover = (data: any) => {
  console.log('[ThreatMapView] 悬停地图点:', data)
}

const handleCountryClick = (country: any) => {
  console.log('[ThreatMapView] 点击国家/地区:', country)
}

const handleCityClick = (city: any) => {
  console.log('[ThreatMapView] 点击城市:', city)
}

const handleIPClick = (ip: any) => {
  console.log('[ThreatMapView] 点击IP:', ip)
}

// 自动更新定时器
let autoUpdateTimer: number | null = null

// 启动自动更新
const startAutoUpdate = () => {
  // 先清除旧定时器
  if (autoUpdateTimer) {
    clearInterval(autoUpdateTimer)
  }
  
  // 每60秒自动刷新一次（性能考虑：不要太频繁）
  autoUpdateTimer = window.setInterval(async () => {
    try {
      console.log('[ThreatMapView] 🔄 自动更新地图数据...')
      const newData = await mapDataManager.getMapData()
      mapData.value = newData
      console.log('[ThreatMapView] ✅ 自动更新完成')
    } catch (error) {
      console.error('[ThreatMapView] ⚠️ 自动更新失败:', error)
    }
  }, 30000) // 30秒
  
  console.log('[ThreatMapView] 🚀 自动更新已启动（间隔30秒）')
}

// 初始化
const init = async () => {
  loading.value = true
  try {
    console.log('[ThreatMapView] 🚀 开始初始化...')
    // 强制更新数据
    mapData.value = await mapDataManager.updateMapData(true)
    console.log('[ThreatMapView] ✅ 初始化地图数据成功:', mapData.value)
    
    // 启动自动更新
    startAutoUpdate()
  } catch (error) {
    console.error('[ThreatMapView] ❌ 初始化失败:', error)
  } finally {
    loading.value = false
  }
}

// 生命周期
onMounted(async () => {
  document.addEventListener('fullscreenchange', handleFullscreenChange)
  // 延迟初始化，确保DOM已渲染
  await nextTick()
  setTimeout(() => {
    init()
  }, 100)
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
  
  // 清理定时器
  if (autoUpdateTimer) {
    clearInterval(autoUpdateTimer)
    autoUpdateTimer = null
    console.log('[ThreatMapView] 🛑 自动更新已停止')
  }
})

// 导出方法供父组件使用
defineExpose({
  refreshData
})
</script>

<style scoped>
/* 页面容器 */
.threat-map-page {
  width: 100%;
  height: 100%;
  background: linear-gradient(to bottom right, #0f172a, #1e293b, #0f172a);
  padding: 16px;
  overflow: hidden;
}

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 24px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-section {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pulse-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #22d3ee;
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.page-title {
  font-size: 24px;
  font-weight: bold;
  color: white;
  margin: 0;
}

/* 统计卡片 */
.stats-cards {
  display: flex;
  gap: 12px;
}

.stat-card {
  padding: 8px 16px;
  border-radius: 8px;
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-cyan {
  background: rgba(6, 182, 212, 0.1);
  border: 1px solid rgba(6, 182, 212, 0.2);
}

.stat-purple {
  background: rgba(168, 85, 247, 0.1);
  border: 1px solid rgba(168, 85, 247, 0.2);
}

.stat-green {
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.2);
}

.stat-label {
  font-size: 12px;
  color: #94a3b8;
}

.stat-value {
  font-size: 14px;
  font-weight: bold;
}

.stat-cyan .stat-value { color: #22d3ee; }
.stat-purple .stat-value { color: #a855f7; }
.stat-green .stat-value { color: #22c55e; }

/* 操作按钮 */
.action-btn {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: white;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-info {
  background: #0ea5e9;
}

.btn-info:hover {
  background: #0284c7;
}

.btn-primary {
  background: #2563eb;
}

.btn-primary:hover, .btn-primary.active {
  background: #1d4ed8;
}

.btn-secondary {
  background: #475569;
  color: #cbd5e1;
}

.btn-secondary:hover {
  background: #334155;
}

.btn-success {
  background: #16a34a;
}

.btn-success:hover {
  background: #15803d;
}

.btn-success.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-purple {
  background: #9333ea;
}

.btn-purple:hover {
  background: #7e22ce;
}

.btn-warning {
  background: #f59e0b;
}

.btn-warning:hover, .btn-warning.active {
  background: #d97706;
}

/* 地图容器固定高度 */
.map-container {
  position: relative;
  background: #0f172a;
  border-radius: 8px;
  border: 1px solid #334155;
  overflow: hidden;
  height: calc(100vh - 180px);
  min-height: 500px;
}

/* 右侧统计面板 - 优化尺寸和位置到右下角 */
.stats-panel {
  position: fixed;
  right: 16px;
  bottom: 16px;
  width: 280px;
  max-height: 360px;
  background: rgba(30, 41, 59, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 10px;
  border: 1px solid #334155;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3);
  z-index: 1000;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.stats-panel.collapsed {
  width: 48px;
  height: 48px;
  max-height: 48px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  background: #475569;
  border-radius: 10px 10px 0 0;
  border-bottom: 1px solid #334155;
}

.collapsed .panel-header {
  border-radius: 10px;
  border-bottom: none;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: white;
}


.collapsed .panel-content {
  display: none;
}

.panel-content {
  display: flex;
  flex-direction: column;
  height: calc(100% - 44px);
}

.panel-tabs {
  display: flex;
  border-bottom: 1px solid #334155;
  background: rgba(51, 65, 85, 0.3);
}

.tab-btn {
  flex: 1;
  padding: 10px 12px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #94a3b8;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.tab-btn:hover {
  background: rgba(71, 85, 105, 0.5);
  color: white;
}

.tab-btn.active.countries {
  background: rgba(37, 99, 235, 0.2);
  color: white;
  border-bottom-color: #3b82f6;
}

.tab-btn.active.cities {
  background: rgba(34, 197, 94, 0.2);
  color: white;
  border-bottom-color: #22c55e;
}

.tab-btn.active.ips {
  background: rgba(249, 115, 22, 0.2);
  color: white;
  border-bottom-color: #f97316;
}

.tab-content-area {
  height: 240px;
  overflow-y: auto;
  padding: 12px;
}

.stats-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-item {
  padding: 10px;
  background: rgba(51, 65, 85, 0.4);
  border: 1px solid #475569;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.stat-item:hover {
  background: rgba(71, 85, 105, 0.6);
  border-color: #06b6d4;
  transform: translateX(-2px);
}

.stat-item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.stat-name {
  font-size: 13px;
  font-weight: 600;
  color: white;
}

.stat-ip {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
}

.stat-rank {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
}

.rank-countries {
  background: rgba(34, 211, 238, 0.2);
  color: #22d3ee;
}

.rank-cities {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
}

.rank-ips {
  background: rgba(251, 146, 60, 0.2);
  color: #fb923c;
}

.stat-item-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.stat-detail {
  font-size: 11px;
}

.stat-detail .label {
  color: #94a3b8;
}

.stat-detail .value {
  color: white;
  font-weight: 600;
  margin-left: 4px;
}

.stat-extra {
  margin-top: 4px;
  font-size: 10px;
  color: #94a3b8;
}

/* 滚动条样式 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: #1e293b;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb {
  background: #475569;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #64748b;
}

/* 动画 */
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
</style>

