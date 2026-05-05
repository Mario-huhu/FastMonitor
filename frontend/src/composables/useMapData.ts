// Vue3 地图数据 Composable
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { mapDataManager, type MapPoint, type CountryStats, type CityStats } from '../services/MapDataManager'
import { mapOriginPoint } from '../stores/mapConfig'

export interface MapSettings {
  updateInterval: number // 秒
  displayMode: 'country' | 'city'
  mapType: 'world' | 'china'
}

export function useMapData() {
  const mapData = ref<MapPoint[]>([])
  const countryMapPoints = ref<MapPoint[]>([])
  const cityMapPoints = ref<MapPoint[]>([])
  const countryStats = ref<CountryStats[]>([])
  const cityStats = ref<CityStats[]>([])
  const topIPs = ref<Array<{ ip: string; count: number; country: string; city: string }>>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  
  // 统计数据
  const stats = computed(() => ({
    totalSessions: mapDataManager.getStats().totalSessions,
    uniqueIPs: mapDataManager.getStats().uniqueIPs,
    uniqueCountries: mapDataManager.getStats().uniqueCountries,
    uniqueCities: mapDataManager.getStats().uniqueCities
  }))

  // 默认设置
  const settings = ref<MapSettings>({
    updateInterval: 300, // 5分钟
    displayMode: 'city',
    mapType: 'world'
  })

  // 默认位置使用全局配置
  const defaultLocation = computed(() => mapOriginPoint.value)

  let updateTimer: number | null = null

  // 刷新数据
  async function refreshData(forceRefresh = false) {
    loading.value = true
    error.value = null

    try {
      console.log('[useMapData] 开始刷新地图数据...')
      const cache = await mapDataManager.updateMapData(forceRefresh)
      
      mapData.value = cache.mapPoints
      countryMapPoints.value = cache.countryMapPoints
      cityMapPoints.value = cache.cityMapPoints
      countryStats.value = cache.countryStats
      cityStats.value = cache.cityStats
      topIPs.value = cache.topIPs
      
      console.log('[useMapData] 地图数据刷新完成:', {
        mapPoints: mapData.value.length,
        countryPoints: countryMapPoints.value.length,
        cityPoints: cityMapPoints.value.length,
        countries: countryStats.value.length,
        cities: cityStats.value.length
      })
    } catch (err) {
      error.value = err instanceof Error ? err.message : '数据加载失败'
      console.error('[useMapData] 数据刷新失败:', err)
    } finally {
      loading.value = false
    }
  }

  // 清除缓存
  function clearCache() {
    mapDataManager.clearCache()
    mapData.value = []
    countryMapPoints.value = []
    cityMapPoints.value = []
    countryStats.value = []
    cityStats.value = []
    topIPs.value = []
  }

  // 启动自动刷新
  function startAutoRefresh() {
    if (updateTimer) {
      clearInterval(updateTimer)
    }
    
    updateTimer = window.setInterval(() => {
      refreshData(false)
    }, settings.value.updateInterval * 1000)
    
    console.log(`[useMapData] 自动刷新已启动，间隔: ${settings.value.updateInterval}秒`)
  }

  // 停止自动刷新
  function stopAutoRefresh() {
    if (updateTimer) {
      clearInterval(updateTimer)
      updateTimer = null
      console.log('[useMapData] 自动刷新已停止')
    }
  }

  // 组件挂载时初始化
  onMounted(() => {
    refreshData(true)
    startAutoRefresh()
  })

  // 组件卸载时清理
  onUnmounted(() => {
    stopAutoRefresh()
  })

  return {
    // 数据
    mapData,
    countryMapPoints,
    cityMapPoints,
    countryStats,
    cityStats,
    topIPs,
    stats,
    
    // 状态
    loading,
    error,
    
    // 设置
    settings,
    defaultLocation,
    
    // 方法
    refreshData,
    clearCache,
    startAutoRefresh,
    stopAutoRefresh
  }
}

