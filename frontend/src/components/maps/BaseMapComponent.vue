<template>
  <div class="w-full h-full relative">
    <!-- 加载状态 -->
    <div
      v-if="loading"
      class="absolute inset-0 flex items-center justify-center bg-slate-900/80 backdrop-blur-sm z-20"
    >
      <div class="flex flex-col items-center space-y-4">
        <div class="animate-spin rounded-full h-12 w-12 border-4 border-cyan-500 border-t-transparent"></div>
        <p class="text-slate-300 text-sm">正在加载地图数据...</p>
      </div>
    </div>

    <!-- ECharts容器 -->
    <div ref="chartContainer" class="w-full h-full"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'

// 定义Props
interface MapPoint {
  name: string
  value: [number, number, number] // [lng, lat, count]
  country?: string
  city?: string
  ips?: string[]
  type?: string
}

interface Props {
  data?: MapPoint[]
  loading?: boolean
  height?: string
  mapRegion?: 'world' | 'china'
  showLines?: boolean
  defaultLocation?: { name: string; lat: number; lng: number }
  visualMode?: 'scatter' | 'heatmap' | 'both'
}

const props = withDefaults(defineProps<Props>(), {
  data: () => [],
  loading: false,
  height: '600px',
  mapRegion: 'world',
  showLines: true,
  defaultLocation: () => ({ name: '中国', lat: 35, lng: 105 }),
  visualMode: 'both'
})

// 定义Events
const emit = defineEmits<{
  pointClick: [data: MapPoint]
  pointHover: [data: MapPoint]
}>()

// Refs
const chartContainer = ref<HTMLDivElement>()
let chartInstance: echarts.ECharts | null = null
let resizeObserver: ResizeObserver | null = null

// 获取点的颜色
const getPointColor = (value: number, maxValue: number): string => {
  const ratio = value / maxValue
  if (ratio > 0.8) return '#ef4444' // 红色 - 高流量
  if (ratio > 0.6) return '#f59e0b' // 橙色 - 中高流量
  if (ratio > 0.4) return '#eab308' // 黄色 - 中等流量
  if (ratio > 0.2) return '#22c55e' // 绿色 - 低流量
  return '#3b82f6' // 蓝色 - 极低流量
}

// 获取点的大小
const getPointSize = (value: number, maxValue: number): number => {
  const baseSize = 8
  const normalized = Math.sqrt(value / maxValue)
  return Math.max(baseSize, normalized * 30)
}

// 初始化图表
let initRetryCount = 0
const MAX_INIT_RETRIES = 20 // 最多重试20次（2秒）

const initChart = async () => {
  if (!chartContainer.value) {
    console.warn('[BaseMapComponent] chartContainer not available')
    return
  }

  // 检查DOM尺寸
  const rect = chartContainer.value.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) {
    if (initRetryCount < MAX_INIT_RETRIES) {
      initRetryCount++
      setTimeout(initChart, 100)
      return
    } else {
      console.error('[BaseMapComponent] ❌ 初始化失败: 容器尺寸为0')
      return
    }
  }

  // 重置重试计数
  initRetryCount = 0
  console.log('[BaseMapComponent] ✅ 开始初始化，容器尺寸:', rect.width, 'x', rect.height)

  // 销毁旧实例
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }

  // 创建新实例
  chartInstance = echarts.init(chartContainer.value)

  // 加载世界地图
  try {
    console.log('[BaseMapComponent] 开始加载地图JSON文件...')
    const response = await fetch('/maps/world.json')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
    const geoJson = await response.json()
    echarts.registerMap('world', geoJson)
    console.log('[BaseMapComponent] ✅ 世界地图加载成功')
    
    // 加载中国地图
    const chinaResponse = await fetch('/maps/china.json')
    if (!chinaResponse.ok) {
      throw new Error(`HTTP ${chinaResponse.status}: ${chinaResponse.statusText}`)
    }
    const chinaGeoJson = await chinaResponse.json()
    echarts.registerMap('china', chinaGeoJson)
    console.log('[BaseMapComponent] ✅ 中国地图加载成功')
  } catch (error) {
    console.error('[BaseMapComponent] ❌ 加载地图文件失败:', error)
    console.error('[BaseMapComponent] 请检查 /maps/world.json 和 /maps/china.json 是否可访问')
  }

  // 绑定事件
  chartInstance.on('click', (params: any) => {
    if (params.componentType === 'series' && params.data?.dataInfo) {
      emit('pointClick', params.data.dataInfo)
    }
  })

  chartInstance.on('mouseover', (params: any) => {
    if (params.componentType === 'series' && params.data?.dataInfo) {
      emit('pointHover', params.data.dataInfo)
    }
  })

  // 初始化配置
  updateChart()

  // 监听窗口大小变化
  window.addEventListener('resize', handleResize)
  
  // 使用ResizeObserver监听容器大小变化
  if (chartContainer.value) {
    resizeObserver = new ResizeObserver(() => {
      chartInstance?.resize()
    })
    resizeObserver.observe(chartContainer.value)
  }
}

// 更新图表
const updateChart = () => {
  if (!chartInstance || !props.data.length) return

  const maxValue = Math.max(...props.data.map(d => d.value[2] || 0), 1)

  // 准备散点数据
  const scatterData = props.data.map(item => ({
    name: item.name,
    value: item.value,
    itemStyle: {
      color: getPointColor(item.value[2] || 0, maxValue),
      opacity: 0.9,
      shadowBlur: 15,
      shadowColor: 'rgba(34, 211, 238, 0.7)',
      borderColor: 'rgba(34, 211, 238, 0.8)',
      borderWidth: 1
    },
    symbolSize: getPointSize(item.value[2] || 0, maxValue) * 1.2,
    dataInfo: item
  }))

  // 准备连线数据 - 优化样式
  const lineData = props.showLines && props.defaultLocation
    ? props.data
        .filter(item => item.value[2] > 0)
        .map(item => {
          const value = item.value[2] || 0
          const color = getPointColor(value, maxValue)
          return {
            coords: [
              [props.defaultLocation!.lng, props.defaultLocation!.lat],
              [item.value[0], item.value[1]]
            ],
            lineStyle: {
              color: color,
              width: Math.max(2, Math.sqrt(value / maxValue) * 5),
              opacity: 0.6,
              curveness: 0.35,
              type: 'solid'
            },
            effect: {
              show: true,
              period: 5 + Math.random() * 3, // 5-8秒随机周期,更自然
              trailLength: 0.2,
              symbol: 'circle',
              symbolSize: 5,
              color: '#22d3ee',
              constantSpeed: 50
            }
          }
        })
    : []

  const currentMap = props.mapRegion === 'china' ? 'china' : 'world'
  const zoomLevel = props.mapRegion === 'china' ? 1.5 : 1.2

  const option: echarts.EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(15, 23, 42, 0.95)',
      borderColor: '#06b6d4',
      borderWidth: 1,
      textStyle: {
        color: '#f8fafc',
        fontSize: 12
      },
      formatter: (params: any) => {
        const data = params.data?.dataInfo
        if (!data) return ''
        return `
          <div style="padding: 8px;">
            <div style="font-weight: bold; color: #22d3ee; margin-bottom: 8px;">${data.name}</div>
            <div style="font-size: 12px;">
              <div style="margin-bottom: 4px;">连接数: <span style="color: white; font-weight: 600;">${data.value[2]?.toLocaleString()}</span></div>
              ${data.country ? `<div style="margin-bottom: 4px;">国家/地区: <span style="color: white;">${data.country}</span></div>` : ''}
              ${data.city ? `<div style="margin-bottom: 4px;">城市: <span style="color: white;">${data.city}</span></div>` : ''}
              ${data.ips ? `<div>IP数量: <span style="color: white;">${data.ips.length}</span></div>` : ''}
            </div>
          </div>
        `
      }
    },
      geo: {
        map: currentMap,
        roam: true,
        scaleLimit: {
          min: 1,
          max: 20
        },
        zoom: zoomLevel,
        center: props.mapRegion === 'china' ? [104, 36] : [0, 20],
        itemStyle: {
          areaColor: 'rgba(15, 35, 65, 0.95)',
          borderColor: 'rgba(59, 130, 246, 0.3)',
          borderWidth: 1,
          shadowColor: 'rgba(0, 0, 0, 0.5)',
          shadowBlur: 10
        },
        emphasis: {
          itemStyle: {
            areaColor: 'rgba(30, 50, 90, 0.95)',
            borderColor: 'rgba(59, 130, 246, 0.6)'
          },
          label: {
            show: true,
            color: '#22d3ee',
            fontSize: 14,
            fontWeight: 'bold'
          }
        },
        label: {
          show: false
        },
        silent: false
      },
    series: [
      // 连线图层 (先绘制,在底层)
      ...(lineData.length > 0
        ? [
            {
              type: 'lines',
              coordinateSystem: 'geo',
              data: lineData,
              zlevel: 1,
              polyline: false,
              large: true,
              largeThreshold: 100,
              effect: {
                show: true,
                period: 4,
                trailLength: 0.15,
                symbolSize: 4,
                constantSpeed: 60
              },
              lineStyle: {
                opacity: 0.4,
                curveness: 0.3
              },
              emphasis: {
                lineStyle: {
                  width: 3,
                  opacity: 0.8
                }
              },
              animation: true,
              animationDuration: 1500,
              animationEasing: 'cubicInOut'
            }
          ]
        : []),
      // 散点图层 (后绘制,在上层)
      {
        type: 'scatter',
        coordinateSystem: 'geo',
        data: scatterData,
        zlevel: 2,
        silent: false,
        animation: true,
        animationDuration: 1500,
        animationEasing: 'elasticOut',
        progressive: 200,
        progressiveThreshold: 400,
        emphasis: {
          scale: true,
          scaleSize: 18,
          itemStyle: {
            opacity: 1,
            shadowBlur: 25,
            shadowColor: 'rgba(34, 211, 238, 1)',
            borderWidth: 2
          }
        }
      },
      // 脉冲效果图层 - 前10个点
      ...(scatterData.length > 0
        ? [
            {
              type: 'effectScatter',
              coordinateSystem: 'geo',
              data: scatterData.slice(0, Math.min(10, scatterData.length)),
              zlevel: 3,
              symbolSize: (params: any) => {
                return (params.symbolSize || 10) * 0.6
              },
              rippleEffect: {
                brushType: 'stroke',
                scale: 3.5,
                period: 4
              },
              itemStyle: {
                color: 'rgba(34, 211, 238, 0.8)',
                shadowBlur: 12,
                shadowColor: 'rgba(34, 211, 238, 0.6)'
              },
              silent: true
            }
          ]
        : [])
    ]
  }

  chartInstance.setOption(option, { replaceMerge: ['series'] })
}

// 处理窗口大小变化
const handleResize = () => {
  chartInstance?.resize()
}

// 监听数据变化
watch(
  () => [props.data, props.showLines, props.visualMode, props.defaultLocation, props.mapRegion],
  () => {
    nextTick(() => {
      if (chartInstance) {
        updateChart()
      } else {
        // 如果图表未初始化（容器之前为0），尝试重新初始化
        console.log('[BaseMapComponent] 数据变化，尝试重新初始化图表')
        initRetryCount = 0
        initChart()
      }
    })
  },
  { deep: true }
)

watch(
  () => props.loading,
  (newVal) => {
    if (!newVal) {
      nextTick(() => {
        if (chartInstance) {
          chartInstance.resize()
        } else {
          // 加载完成但图表未初始化，尝试初始化
          console.log('[BaseMapComponent] 加载完成，尝试初始化图表')
          initRetryCount = 0
          initChart()
        }
      })
    }
  }
)

// 可见性检测 - 使用IntersectionObserver
let intersectionObserver: IntersectionObserver | null = null

const setupVisibilityObserver = () => {
  if (!chartContainer.value) return

  intersectionObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting && !chartInstance) {
          // 容器变为可见且图表未初始化，尝试初始化
          console.log('[BaseMapComponent] 容器变为可见，尝试初始化图表')
          initRetryCount = 0
          setTimeout(() => {
            initChart()
          }, 200)
        }
      })
    },
    { threshold: 0.1 }
  )

  intersectionObserver.observe(chartContainer.value)
}

// 生命周期
onMounted(async () => {
  await nextTick()
  initChart()

  // 设置可见性观察器
  setupVisibilityObserver()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (resizeObserver) {
    resizeObserver.disconnect()
  }
  if (intersectionObserver) {
    intersectionObserver.disconnect()
  }
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})

// 暴露方法
defineExpose({
  resize: () => chartInstance?.resize(),
  getInstance: () => chartInstance
})
</script>

<style scoped>
/* 确保容器有明确的尺寸 */
.w-full {
  width: 100%;
}

.h-full {
  height: 100%;
}
</style>


