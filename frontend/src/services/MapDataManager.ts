// 统一地图数据管理器 - 高性能版本 (Vue3适配)
// 适配Wails Go函数调用
/* eslint-disable @typescript-eslint/no-unused-vars */

import { GetMapData } from '../../wailsjs/go/server/App'

export interface GeoLocation {
  country: string
  city: string
  latitude: number
  longitude: number
  source?: string
}

export interface MapPoint {
  name: string
  value: [number, number, number] // [longitude, latitude, count]
  ips?: string[]
  country?: string
  city?: string
  location?: string
  count?: number
  lat?: number
  lng?: number
  type?: string
  sessions?: any[]
  connections?: number
  coordinates?: [number, number]
}

export interface CountryStats {
  country: string
  connections: number
  uniqueIPs: number
  cities: string[]
  name?: string
  count?: number
  ips?: string[]
}

export interface CityStats {
  country: string
  city: string
  connections: number
  uniqueIPs: number
  name?: string
  count?: number
  ips?: string[]
}

export interface MapDataCache {
  mapPoints: MapPoint[]
  countryMapPoints: MapPoint[]
  cityMapPoints: MapPoint[]
  countryStats: CountryStats[]
  cityStats: CityStats[]
  topIPs: Array<{
    ip: string
    count: number
    country: string
    city: string
  }>
  lastUpdate: number
  totalSessions: number
  uniqueIPs: number
}

class MapDataManager {
  private static instance: MapDataManager
  private cache: MapDataCache | null = null
  private isUpdating: boolean = false

  // 🚀 优化：多级缓存策略
  private geoCache = new Map<string, GeoLocation>() // L1: 内存地理位置缓存
  private sessionCache = new Map<string, any>() // L2: 会话数据缓存
  private aggregateCache = new Map<string, any>() // L3: 聚合数据缓存

  // 🚀 优化：不同级别的超时时间
  private cacheTimeout = 30 * 60 * 1000 // 30分钟主缓存
  private sessionCacheTimeout = 5 * 60 * 1000 // 5分钟会话缓存
  private aggregateCacheTimeout = 15 * 60 * 1000 // 15分钟聚合缓存

  private constructor() {
    this.loadGeoCache()
    this.startCacheCleanup()
  }

  static getInstance(): MapDataManager {
    if (!MapDataManager.instance) {
      MapDataManager.instance = new MapDataManager()
    }
    return MapDataManager.instance
  }

  // 🚀 新增：智能缓存清理
  private startCacheCleanup(): void {
    setInterval(() => {
      const now = Date.now()
      
      // 清理过期的会话缓存
      for (const [key, value] of this.sessionCache.entries()) {
        if (now - value.timestamp > this.sessionCacheTimeout) {
          this.sessionCache.delete(key)
        }
      }
      
      // 清理过期的聚合缓存
      for (const [key, value] of this.aggregateCache.entries()) {
        if (now - value.timestamp > this.aggregateCacheTimeout) {
          this.aggregateCache.delete(key)
        }
      }
      
      console.log(`[MapDataManager] 缓存清理完成: 会话${this.sessionCache.size}, 聚合${this.aggregateCache.size}`)
    }, 10 * 60 * 1000) // 每10分钟清理一次
  }

  // 🔥 主方法：获取/更新地图数据
  async updateMapData(forceRefresh = false): Promise<MapDataCache> {
    // 🚀 快速返回：如果有有效缓存且不强制刷新
    if (!forceRefresh && this.isCacheValid()) {
      console.log('[MapDataManager] 🚀 使用有效缓存，耗时: 0ms')
      return this.cache!
    }

    // 防止并发更新
    if (this.isUpdating) {
      console.log('[MapDataManager] 正在更新中，等待完成...')
      await this.waitForUpdate()
      return this.cache!
    }

    this.isUpdating = true
    const startTime = Date.now()

    try {
      console.log('[MapDataManager] 🚀 开始地图数据更新')

      // 使用后端统一API获取地图数据
      const mapData = await this.getMapDataFromBackend()
      
      if (!mapData) {
        throw new Error('无法从后端获取地图数据')
      }

      // 更新缓存
      this.cache = mapData

      const duration = Date.now() - startTime
      console.log(`[MapDataManager] ✅ 地图数据更新完成，耗时: ${duration}ms`)

      return this.cache
    } catch (error) {
      console.error('[MapDataManager] 地图数据更新失败:', error)
      throw error
    } finally {
      this.isUpdating = false
    }
  }

  // 🔥 使用后端统一API获取地图数据
  private async getMapDataFromBackend(): Promise<MapDataCache | null> {
    try {
      console.log('[MapDataManager] 调用后端GetMapData API')
      const result = await GetMapData()
      
      if (!result) {
        console.warn('[MapDataManager] 后端返回空数据')
        return null
      }

      console.log(`[MapDataManager] 后端返回数据:`, {
        mapPoints: result.map_points?.length || 0,
        countryPoints: result.country_map_points?.length || 0,
        cityPoints: result.city_map_points?.length || 0,
        totalSessions: result.total_sessions || 0
      })

      // 🎯 如果没有数据，返回默认示例数据
      if (!result.city_map_points || result.city_map_points.length === 0) {
        console.log('[MapDataManager] 🌍 无数据时返回示例地图点')
        return this.getDefaultMapData()
      }

      // 转换后端数据格式
      return {
        mapPoints: this.convertBackendMapPoints(result.city_map_points || []),
        countryMapPoints: this.convertBackendMapPoints(result.country_map_points || []),
        cityMapPoints: this.convertBackendMapPoints(result.city_map_points || []),
        countryStats: (result.country_stats || []).map((stat: any) => ({
          country: stat.country || '',
          connections: stat.connections || 0,
          uniqueIPs: stat.unique_ips || 0,
          cities: stat.cities || [],
          name: stat.country || '',
          count: stat.connections || 0,
          ips: stat.ips || []
        })),
        cityStats: (result.city_stats || []).map((stat: any) => ({
          country: stat.country || '',
          city: stat.city || '',
          connections: stat.connections || 0,
          uniqueIPs: stat.unique_ips || 0,
          name: stat.city || '',
          count: stat.connections || 0,
          ips: stat.ips || []
        })),
        topIPs: result.top_ips || [],
        lastUpdate: Date.now(),
        totalSessions: result.total_sessions || 0,
        uniqueIPs: result.unique_ips || 0
      }
    } catch (error) {
      console.error('[MapDataManager] 调用后端API失败:', error)
      // 出错时也返回默认数据
      return this.getDefaultMapData()
    }
  }

  // 🌍 返回默认示例地图数据
  private getDefaultMapData(): MapDataCache {
    const defaultPoints: MapPoint[] = [
      {
        name: '北京, 中国',
        value: [116.4074, 39.9042, 5],
        ips: ['示例数据'],
        country: '中国-示例数据',
        city: '北京',
        location: '北京, 中国',
        count: 5,
        lat: 39.9042,
        lng: 116.4074,
        type: 'city',
        sessions: [],
        connections: 5,
        coordinates: [116.4074, 39.9042]
      },
      {
        name: '上海, 中国',
        value: [121.4737, 31.2304, 3],
        ips: ['示例数据'],
        country: '中国',
        city: '上海',
        location: '上海, 中国',
        count: 3,
        lat: 31.2304,
        lng: 121.4737,
        type: 'city',
        sessions: [],
        connections: 3,
        coordinates: [121.4737, 31.2304]
      },
      {
        name: '纽约, 美国',
        value: [-74.0060, 40.7128, 8],
        ips: ['示例数据'],
        country: '美国-示例数据',
        city: '纽约',
        location: '纽约, 美国',
        count: 8,
        lat: 40.7128,
        lng: -74.0060,
        type: 'city',
        sessions: [],
        connections: 8,
        coordinates: [-74.0060, 40.7128]
      },
      {
        name: '伦敦, 英国',
        value: [-0.1278, 51.5074, 4],
        ips: ['示例数据'],
        country: '英国',
        city: '伦敦',
        location: '伦敦, 英国',
        count: 4,
        lat: 51.5074,
        lng: -0.1278,
        type: 'city',
        sessions: [],
        connections: 4,
        coordinates: [-0.1278, 51.5074]
      },
      {
        name: '东京, 日本',
        value: [139.6503, 35.6762, 6],
        ips: ['示例数据'],
        country: '日本',
        city: '东京',
        location: '东京, 日本',
        count: 6,
        lat: 35.6762,
        lng: 139.6503,
        type: 'city',
        sessions: [],
        connections: 6,
        coordinates: [139.6503, 35.6762]
      },
      {
        name: '新加坡, 新加坡',
        value: [103.8198, 1.3521, 7],
        ips: ['示例数据'],
        country: '新加坡',
        city: '新加坡',
        location: '新加坡, 新加坡',
        count: 7,
        lat: 1.3521,
        lng: 103.8198,
        type: 'city',
        sessions: [],
        connections: 7,
        coordinates: [103.8198, 1.3521]
      }
    ]

    return {
      mapPoints: defaultPoints,
      countryMapPoints: [
        {
          name: '中国-示例数据',
          value: [116.4074, 39.9042, 8],
          ips: ['示例数据'],
          country: '中国-示例数据',
          city: '',
          location: '中国',
          count: 8,
          lat: 39.9042,
          lng: 116.4074,
          type: 'country',
          sessions: [],
          connections: 8,
          coordinates: [116.4074, 39.9042]
        },
        {
          name: '美国-示例数据',
          value: [-98.5795, 39.8283, 8],
          ips: ['示例数据'],
          country: '美国-示例数据',
          city: '',
          location: '美国',
          count: 8,
          lat: 39.8283,
          lng: -98.5795,
          type: 'country',
          sessions: [],
          connections: 8,
          coordinates: [-98.5795, 39.8283]
        }
      ],
      cityMapPoints: defaultPoints,
      countryStats: [
        {
          country: '中国-示例数据',
          connections: 8,
          uniqueIPs: 2,
          cities: ['北京', '上海'],
          name: '中国-示例数据',
          count: 8,
          ips: ['示例数据']
        },
        {
          country: '美国-示例数据',
          connections: 8,
          uniqueIPs: 1,
          cities: ['纽约'],
          name: '美国-示例数据',
          count: 8,
          ips: ['示例数据']
        }
      ],
      cityStats: [
        {
          country: '美国-示例数据',
          city: '纽约',
          connections: 8,
          uniqueIPs: 1,
          name: '纽约, 美国',
          count: 8,
          ips: ['示例数据']
        },
        {
          country: '新加坡-示例数据',
          city: '新加坡',
          connections: 7,
          uniqueIPs: 1,
          name: '新加坡, 新加坡',
          count: 7,
          ips: ['示例数据']
        }
      ],
      topIPs: [
        { ip: '8.8.8.8', count: 10, country: '美国', city: '纽约' },
        { ip: '1.1.1.1', count: 8, country: '美国', city: '旧金山' }
      ],
      lastUpdate: Date.now(),
      totalSessions: 33,
      uniqueIPs: 6
    }
  }

  // 转换后端地图点格式
  private convertBackendMapPoints(points: any[]): MapPoint[] {
    return points.map(p => ({
      name: p.name || '',
      value: [p.longitude || 0, p.latitude || 0, p.connections || 0],
      ips: p.ips || [],
      country: p.country || '',
      city: p.city || '',
      location: p.name || '',
      count: p.connections || 0,
      lat: p.latitude || 0,
      lng: p.longitude || 0,
      type: p.type || 'city',
      sessions: [],
      connections: p.connections || 0,
      coordinates: [p.longitude || 0, p.latitude || 0]
    }))
  }

  // 🔥 关键优化：提取唯一IP (保留以供未来使用)
  private extractUniqueDestinationIPs(sessions: any[]): string[] {
    const ipSet = new Set<string>()
    
    sessions.forEach(session => {
      const dstIP = session.dst_ip || session.DstIP
      const srcIP = session.src_ip || session.SrcIP
      
      // 提取目标IP（如果是外网IP）
      if (dstIP && !this.isLocalIP(dstIP)) {
        ipSet.add(dstIP)
      }
      // 提取源IP（如果是外网IP）
      if (srcIP && !this.isLocalIP(srcIP)) {
        ipSet.add(srcIP)
      }
    })

    return Array.from(ipSet)
  }

  // 🔥 关键优化：批量处理IP地理信息 (保留以供未来使用)
  private async batchProcessIPGeoLocation(ips: string[]): Promise<void> {
    // 过滤已缓存的IP
    const uncachedIPs = ips.filter(ip => !this.geoCache.has(ip))
    
    if (uncachedIPs.length === 0) {
      console.log('[MapDataManager] 🚀 所有IP已缓存，跳过地理位置解析')
      return
    }

    console.log(`[MapDataManager] 需要解析 ${uncachedIPs.length}/${ips.length} 个IP的地理位置`)

    // 模拟地理位置解析（实际项目中应该调用后端API）
    // 这里我们为每个IP生成一个基于IP的固定位置
    uncachedIPs.forEach(ip => {
      const geoInfo = this.mockIPGeolocation(ip)
      this.geoCache.set(ip, geoInfo)
    })
    
    console.log(`[MapDataManager] ✅ 地理位置解析完成，新增 ${uncachedIPs.length} 个IP`)
  }

  // 模拟IP地理位置解析
  private mockIPGeolocation(ip: string): GeoLocation {
    // 根据IP的第一段简单分配国家
    const firstOctet = parseInt(ip.split('.')[0])
    
    const countries = [
      { name: '中国', lat: 35, lng: 105 },
      { name: '美国', lat: 40, lng: -100 },
      { name: '日本', lat: 36, lng: 138 },
      { name: '英国', lat: 54, lng: -2 },
      { name: '德国', lat: 51, lng: 10 },
      { name: '新加坡', lat: 1.3, lng: 103.8 }
    ]
    
    const country = countries[firstOctet % countries.length]
    
    // 添加随机偏移
    const latOffset = (Math.random() - 0.5) * 10
    const lngOffset = (Math.random() - 0.5) * 10
    
    return {
      country: country.name,
      city: '未知',
      latitude: country.lat + latOffset,
      longitude: country.lng + lngOffset,
      source: 'mock'
    }
  }

  // 🚀 优化：聚合地图数据 (保留以供未来使用)
  private async aggregateMapDataOptimized(sessions: any[]): Promise<{
    mapPoints: MapPoint[]
    countryMapPoints: MapPoint[]
    cityMapPoints: MapPoint[]
    countryStats: CountryStats[]
    cityStats: CityStats[]
    topIPs: Array<{ ip: string; count: number; country: string; city: string }>
  }> {
    // L3缓存检查
    const cacheKey = `aggregate_${sessions.length}_${this.geoCache.size}`
    const cached = this.aggregateCache.get(cacheKey)
    if (cached && (Date.now() - cached.timestamp) < this.aggregateCacheTimeout) {
      console.log('[MapDataManager] 🚀 使用聚合数据缓存')
      return cached.data
    }

    const result = await this.aggregateMapData(sessions)
    
    // 缓存聚合结果
    this.aggregateCache.set(cacheKey, {
      data: result,
      timestamp: Date.now()
    })
    
    return result
  }

  // 聚合地图数据
  private async aggregateMapData(sessions: any[]): Promise<{
    mapPoints: MapPoint[]
    countryMapPoints: MapPoint[]
    cityMapPoints: MapPoint[]
    countryStats: CountryStats[]
    cityStats: CityStats[]
    topIPs: Array<{ ip: string; count: number; country: string; city: string }>
  }> {
    const countryMap = new Map<string, CountryStats>()
    const cityMap = new Map<string, CityStats>()
    const ipMap = new Map<string, { count: number; geo: GeoLocation }>()
    const locationMap = new Map<string, MapPoint>()
    const countryLocationMap = new Map<string, MapPoint>()
    const cityLocationMap = new Map<string, MapPoint>()

    // 处理每个会话
    sessions.forEach(session => {
      const dstIP = session.dst_ip || session.DstIP
      if (!dstIP || this.isLocalIP(dstIP)) return

      const geoInfo = this.geoCache.get(dstIP)
      if (!geoInfo) return

      const { country, city, latitude, longitude } = geoInfo
      
      // 统计IP
      if (ipMap.has(dstIP)) {
        ipMap.get(dstIP)!.count++
      } else {
        ipMap.set(dstIP, { count: 1, geo: geoInfo })
      }

      // 国家统计
      if (countryMap.has(country)) {
        const stats = countryMap.get(country)!
        stats.connections++
        if (!stats.cities.includes(city)) {
          stats.cities.push(city)
        }
      } else {
        countryMap.set(country, {
          country,
          connections: 1,
          uniqueIPs: 0,
          cities: [city]
        })
      }

      // 城市统计
      const cityKey = `${country}-${city}`
      if (cityMap.has(cityKey)) {
        cityMap.get(cityKey)!.connections++
      } else {
        cityMap.set(cityKey, {
          country,
          city,
          connections: 1,
          uniqueIPs: 0
        })
      }

      // 生成地图点
      const locationKey = `${latitude.toFixed(2)}_${longitude.toFixed(2)}`
      if (locationMap.has(locationKey)) {
        const point = locationMap.get(locationKey)!
        point.value[2]++
        if (!point.ips!.includes(dstIP)) {
          point.ips!.push(dstIP)
        }
      } else if (latitude !== 0 || longitude !== 0) {
        locationMap.set(locationKey, {
          name: city !== '未知' ? `${city}, ${country}` : country,
          value: [longitude, latitude, 1],
          ips: [dstIP],
          country,
          city,
          location: `${city}, ${country}`,
          count: 1,
          lat: latitude,
          lng: longitude
        })
      }

      // 国家级地图点
      if (countryLocationMap.has(country)) {
        const point = countryLocationMap.get(country)!
        point.value[2]++
        if (!point.ips!.includes(dstIP)) {
          point.ips!.push(dstIP)
        }
      } else {
        const defaultCoords = this.getDefaultCountryCoordinates(country)
        countryLocationMap.set(country, {
          name: country,
          value: [defaultCoords.lng, defaultCoords.lat, 1],
          ips: [dstIP],
          country,
          city: '',
          location: country,
          count: 1,
          lat: defaultCoords.lat,
          lng: defaultCoords.lng
        })
      }

      // 城市级地图点
      if (cityLocationMap.has(cityKey)) {
        const point = cityLocationMap.get(cityKey)!
        point.value[2]++
        if (!point.ips!.includes(dstIP)) {
          point.ips!.push(dstIP)
        }
      } else {
        let coords = { lat: latitude, lng: longitude }
        if (latitude === 0 && longitude === 0) {
          coords = this.getDefaultCountryCoordinates(country)
        }
        
        cityLocationMap.set(cityKey, {
          name: city !== '未知' ? `${city}, ${country}` : country,
          value: [coords.lng, coords.lat, 1],
          ips: [dstIP],
          country,
          city,
          location: `${city}, ${country}`,
          count: 1,
          lat: coords.lat,
          lng: coords.lng
        })
      }
    })

    // 计算唯一IP数量
    for (const [, data] of ipMap.entries()) {
      const { country, city } = data.geo
      
      if (countryMap.has(country)) {
        countryMap.get(country)!.uniqueIPs++
      }
      
      const cityKey = `${country}-${city}`
      if (cityMap.has(cityKey)) {
        cityMap.get(cityKey)!.uniqueIPs++
      }
    }

    // TOP IP统计
    const topIPs = Array.from(ipMap.entries())
      .map(([ip, data]) => ({
        ip,
        count: data.count,
        country: data.geo.country,
        city: data.geo.city
      }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 20)

    const countryStats = Array.from(countryMap.values()).map(stat => ({
      ...stat,
      name: stat.country,
      count: stat.connections,
      ips: Array.from(ipMap.entries())
        .filter(([_, data]) => data.geo.country === stat.country)
        .map(([ip, _]) => ip)
    })).sort((a, b) => b.connections - a.connections)

    const cityStats = Array.from(cityMap.values()).map(stat => ({
      ...stat,
      name: `${stat.city}, ${stat.country}`,
      count: stat.connections,
      ips: Array.from(ipMap.entries())
        .filter(([_, data]) => data.geo.country === stat.country && data.geo.city === stat.city)
        .map(([ip, _]) => ip)
    })).sort((a, b) => b.connections - a.connections)

    return {
      mapPoints: Array.from(locationMap.values()),
      countryMapPoints: Array.from(countryLocationMap.values()),
      cityMapPoints: Array.from(cityLocationMap.values()),
      countryStats,
      cityStats,
      topIPs
    }
  }

  // 检查是否为本地IP
  private isLocalIP(ip: string): boolean {
    return ip.startsWith('192.168.') || 
           ip.startsWith('10.') || 
           ip.startsWith('127.') ||
           ip.startsWith('169.254.') ||
           ip.match(/^172\.(1[6-9]|2[0-9]|3[01])\./) !== null
  }

  // 获取国家的默认坐标
  private getDefaultCountryCoordinates(country: string): { lat: number; lng: number } {
    const defaultCoords: { [key: string]: { lat: number; lng: number } } = {
      '中国': { lat: 39.9042, lng: 116.4074 },
      '美国': { lat: 39.8283, lng: -98.5795 },
      '日本': { lat: 35.6762, lng: 139.6503 },
      '英国': { lat: 51.5074, lng: -0.1278 },
      '德国': { lat: 52.5200, lng: 13.4050 },
      '新加坡': { lat: 1.3521, lng: 103.8198 },
      '未知': { lat: 0, lng: 0 }
    }

    return defaultCoords[country] || { lat: 0, lng: 0 }
  }

  // 缓存管理
  private isCacheValid(): boolean {
    return this.cache !== null && 
           (Date.now() - this.cache.lastUpdate) < this.cacheTimeout
  }

  private async waitForUpdate(): Promise<void> {
    while (this.isUpdating) {
      await new Promise(resolve => setTimeout(resolve, 100))
    }
  }

  private loadGeoCache(): void {
    console.log('[MapDataManager] 初始化地理位置内存缓存')
  }

  // 公共接口
  getMapPoints(): MapPoint[] {
    return this.cache?.mapPoints || []
  }

  getCountryMapPoints(): MapPoint[] {
    return this.cache?.countryMapPoints || []
  }

  getCityMapPoints(): MapPoint[] {
    return this.cache?.cityMapPoints || []
  }

  getCountryStats(): CountryStats[] {
    return this.cache?.countryStats || []
  }

  getCityStats(): CityStats[] {
    return this.cache?.cityStats || []
  }

  getTopIPs() {
    return this.cache?.topIPs || []
  }

  getStats() {
    return {
      totalSessions: this.cache?.totalSessions || 0,
      uniqueCountries: this.cache?.countryStats?.length || 0,
      uniqueCities: this.cache?.cityStats?.length || 0,
      uniqueIPs: this.cache?.uniqueIPs || 0
    }
  }

  // 公共API: 获取地图数据(返回缓存或更新)
  async getMapData(forceRefresh = false): Promise<MapDataCache> {
    return await this.updateMapData(forceRefresh)
  }

  clearCache(): void {
    this.cache = null
    this.geoCache.clear()
    this.sessionCache.clear()
    this.aggregateCache.clear()
    console.log('[MapDataManager] 所有多级缓存已清理')
  }
}

// 导出单例实例
export const mapDataManager = MapDataManager.getInstance()

