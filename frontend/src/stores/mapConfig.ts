// 地图配置存储
import { ref, watch } from 'vue'

export interface MapOriginPoint {
  name: string
  lat: number
  lng: number
}

// 内置常见城市坐标
export const PRESET_CITIES: MapOriginPoint[] = [
  // 中国主要城市
  { name: '北京', lat: 39.9042, lng: 116.4074 },
  { name: '上海', lat: 31.2304, lng: 121.4737 },
  { name: '广州', lat: 23.1291, lng: 113.2644 },
  { name: '深圳', lat: 22.5431, lng: 114.0579 },
  { name: '成都', lat: 30.5728, lng: 104.0668 },
  { name: '杭州', lat: 30.2741, lng: 120.1551 },
  { name: '重庆', lat: 29.5630, lng: 106.5516 },
  { name: '武汉', lat: 30.5928, lng: 114.3055 },
  { name: '西安', lat: 34.3416, lng: 108.9398 },
  { name: '南京', lat: 32.0603, lng: 118.7969 },
  { name: '天津', lat: 39.3434, lng: 117.3616 },
  { name: '苏州', lat: 31.2989, lng: 120.5853 },
  { name: '郑州', lat: 34.7466, lng: 113.6253 },
  { name: '长沙', lat: 28.2282, lng: 112.9388 },
  { name: '沈阳', lat: 41.8057, lng: 123.4328 },
  { name: '青岛', lat: 36.0671, lng: 120.3826 },
  { name: '东莞', lat: 23.0489, lng: 113.7447 },
  { name: '大连', lat: 38.9140, lng: 121.6147 },
  { name: '厦门', lat: 24.4798, lng: 118.0894 },
  { name: '宁波', lat: 29.8683, lng: 121.5440 },
  
  // 国际主要城市
  { name: '纽约', lat: 40.7128, lng: -74.0060 },
  { name: '洛杉矶', lat: 34.0522, lng: -118.2437 },
  { name: '伦敦', lat: 51.5074, lng: -0.1278 },
  { name: '巴黎', lat: 48.8566, lng: 2.3522 },
  { name: '东京', lat: 35.6762, lng: 139.6503 },
  { name: '首尔', lat: 37.5665, lng: 126.9780 },
  { name: '新加坡', lat: 1.3521, lng: 103.8198 },
  { name: '悉尼', lat: -33.8688, lng: 151.2093 },
  { name: '多伦多', lat: 43.6532, lng: -79.3832 },
  { name: '柏林', lat: 52.5200, lng: 13.4050 },
  { name: '莫斯科', lat: 55.7558, lng: 37.6173 },
  { name: '迪拜', lat: 25.2048, lng: 55.2708 }
]

// 默认起点：东莞市
const DEFAULT_ORIGIN: MapOriginPoint = {
  name: '东莞',
  lat: 23.0489,
  lng: 113.7447
}

// 从localStorage加载配置
const loadOriginFromStorage = (): MapOriginPoint => {
  try {
    const stored = localStorage.getItem('map_origin_point')
    if (stored) {
      return JSON.parse(stored)
    }
  } catch (e) {
    console.warn('[MapConfig] 加载起点配置失败:', e)
  }
  return DEFAULT_ORIGIN
}

// 保存配置到localStorage
const saveOriginToStorage = (origin: MapOriginPoint) => {
  try {
    localStorage.setItem('map_origin_point', JSON.stringify(origin))
    console.log('[MapConfig] 起点配置已保存:', origin)
  } catch (e) {
    console.error('[MapConfig] 保存起点配置失败:', e)
  }
}

// 响应式起点配置
export const mapOriginPoint = ref<MapOriginPoint>(loadOriginFromStorage())

// 监听变化并自动保存
watch(mapOriginPoint, (newOrigin) => {
  saveOriginToStorage(newOrigin)
}, { deep: true })

// 重置为默认值
export const resetMapOrigin = () => {
  mapOriginPoint.value = { ...DEFAULT_ORIGIN }
}

// 更新起点
export const updateMapOrigin = (origin: Partial<MapOriginPoint>) => {
  mapOriginPoint.value = {
    ...mapOriginPoint.value,
    ...origin
  }
}

// 导出默认值供参考
export { DEFAULT_ORIGIN }

// 根据城市名称查找预设城市
export const findPresetCity = (name: string): MapOriginPoint | undefined => {
  return PRESET_CITIES.find(city => city.name === name || city.name.includes(name))
}

// 获取所有预设城市名称列表
export const getPresetCityNames = (): string[] => {
  return PRESET_CITIES.map(city => city.name)
}

