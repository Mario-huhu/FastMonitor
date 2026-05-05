<template>
  <div class="earth-3d-container">
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-overlay">
      <div class="loading-content">
        <div class="spinner"></div>
        <p class="loading-text">正在初始化3D地球...</p>
      </div>
    </div>

    <!-- 控制面板 -->
    <div class="control-panel">
      <!-- 自动旋转控制 -->
      <button
        @click="toggleAutoRotate"
        :class="['control-btn', autoRotate ? 'btn-active' : '']"
        :title="autoRotate ? '停止旋转' : '开始旋转'"
      >
        <component :is="autoRotate ? PauseIcon : PlayIcon" class="icon" />
      </button>

      <!-- 重置视角 -->
      <button
        @click="resetView"
        class="control-btn"
        title="重置视角"
      >
        <component :is="RotateCwIcon" class="icon" />
      </button>

      <!-- 可视模式切换 -->
      <div class="mode-switcher">
        <button
          v-for="mode in visualModes"
          :key="mode.value"
          @click="setVisualMode(mode.value)"
          :class="['mode-btn', { active: visualMode === mode.value }]"
          :title="mode.label"
        >
          {{ mode.label }}
        </button>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="stats-info">
      <div class="stats-content">
        <div class="stat-row">
          <div class="pulse-dot pulse-cyan"></div>
          <span class="stat-label">数据点:</span>
          <span class="stat-value">{{ data.length }}</span>
        </div>
        <div v-if="showConnections" class="stat-row">
          <div class="pulse-dot pulse-green"></div>
          <span class="stat-label">连接线:</span>
          <span class="stat-value">{{ data.length }}</span>
        </div>
      </div>
    </div>

    <!-- ECharts GL 3D容器 -->
    <div ref="chartContainer" class="chart-container"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from 'vue'
import * as echarts from 'echarts'
import 'echarts-gl'

// 图标组件 (使用h函数创建SVG)
import { h } from 'vue'

const PlayIcon = {
  render() {
    return h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { d: 'M8 5v14l11-7z' })
    ])
  }
}

const PauseIcon = {
  render() {
    return h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'currentColor', viewBox: '0 0 24 24' }, [
      h('path', { d: 'M6 4h4v16H6V4zm8 0h4v16h-4V4z' })
    ])
  }
}

const RotateCwIcon = {
  render() {
    return h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', viewBox: '0 0 24 24' }, [
      h('path', { d: 'M23 4v6h-6M1 20v-6h6' }),
      h('path', { d: 'M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15' })
    ])
  }
}

// 定义Props
interface MapPointData {
  lat: number
  lng: number
  value: number
  name: string
  type: 'country' | 'city'
  ips: string[]
  sessions?: any[]
  connections: number
  coordinates?: [number, number]
}

interface Props {
  data?: MapPointData[]
  title?: string
  loading?: boolean
  height?: string
  showConnections?: boolean
  defaultLocation?: { name: string; lat: number; lng: number }
}

const props = withDefaults(defineProps<Props>(), {
  data: () => [],
  title: '3D 地球网络威胁图',
  loading: false,
  height: '600px',
  showConnections: true,
  defaultLocation: () => ({ name: '东莞市', lat: 23.0489, lng: 113.7447 })
})

// 定义Events
const emit = defineEmits<{
  pointClick: [data: MapPointData]
  pointHover: [data: MapPointData]
}>()

// Refs
const chartContainer = ref<HTMLDivElement>()
let chartInstance: echarts.ECharts | null = null
let resizeObserver: ResizeObserver | null = null

// 状态
const autoRotate = ref(false)
const isInitialized = ref(false)

// visualMode完全由内部管理 - 默认设置为流线模式
const visualMode = ref<'scatter' | 'flow' | 'both'>('flow')

// 可视模式选项
const visualModes = [
  { value: 'scatter', label: '散点' },
  { value: 'flow', label: '流线' },
  { value: 'both', label: '全部' }
]

// 固定的最佳视角参数
const viewParams = { alpha: 48, beta: -90 }

// 获取点的颜色
const getPointColor = (value: number, maxValue: number): string => {
  const ratio = value / maxValue
  if (ratio > 0.8) return '#ff4757' // 红色 - 高威胁
  if (ratio > 0.6) return '#ff6b35' // 橙色 - 中高威胁
  if (ratio > 0.4) return '#ffa726' // 黄色 - 中等威胁
  if (ratio > 0.2) return '#26de81' // 绿色 - 低威胁
  return '#00d2ff' // 蓝色 - 极低威胁
}

// 获取点的大小
const getPointSize = (value: number, maxValue: number): number => {
  const baseSize = 8
  const normalized = Math.sqrt(value / maxValue)
  return Math.max(baseSize, normalized * 25)
}

// 获取流线颜色
const getFlowColor = (value: number, maxValue: number): string => {
  const ratio = value / maxValue
  if (ratio > 0.8) return '#ff4757'
  if (ratio > 0.6) return '#ff6b35'
  if (ratio > 0.4) return '#ffa726'
  if (ratio > 0.2) return '#26de81'
  return '#00d2ff'
}

// 获取流线宽度
const getFlowWidth = (value: number, maxValue: number): number => {
  const ratio = value / maxValue
  return Math.max(1, ratio * 4)
}

// 处理数据
const processedData = computed(() => {
  if (!props.data.length) return { scatterData: [], connectionsData: [] }

  console.log('[Enhanced3DEarth] processedData - showConnections:', props.showConnections)
  console.log('[Enhanced3DEarth] processedData - defaultLocation:', props.defaultLocation)
  console.log('[Enhanced3DEarth] processedData - data.length:', props.data.length)

  const values = props.data.map(d => d.connections || d.value || 1)
  const maxValue = Math.max(...values, 1)

  const scatterData = props.data.map((item, index) => {
    const value = values[index]
    return {
      name: item.name,
      value: [item.lng, item.lat, Math.log(value + 1) * 10],
      itemStyle: {
        color: getPointColor(value, maxValue),
        opacity: 0.9
      },
      symbolSize: getPointSize(value, maxValue),
      dataInfo: item
    }
  })

  const connectionsData =
    props.showConnections && props.defaultLocation
      ? props.data
          .filter(item => {
            const value = item.connections || item.value || 0
            return value > 0 && (item.lng !== 0 || item.lat !== 0)
          })
          .map(item => {
            const value = item.connections || item.value || 1
            return {
              coords: [
                [props.defaultLocation!.lng, props.defaultLocation!.lat],
                [item.lng, item.lat]
              ],
              lineStyle: {
                color: getFlowColor(value, maxValue),
                width: getFlowWidth(value, maxValue),
                opacity: 0.6
              },
              dataInfo: item
            }
          })
      : []

  console.log('[Enhanced3DEarth] processedData - connectionsData.length:', connectionsData.length)

  return { scatterData, connectionsData }
})

// 初始化图表
let initRetryCount = 0
const MAX_INIT_RETRIES = 20 // 最多重试20次（2秒）

const initChart = () => {
  if (!chartContainer.value) return

  // 检查DOM尺寸
  const rect = chartContainer.value.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) {
    if (initRetryCount < MAX_INIT_RETRIES) {
      initRetryCount++
      setTimeout(initChart, 100)
      return
    } else {
      console.error('[Enhanced3DEarth] ❌ 初始化失败: 容器尺寸为0')
      return
    }
  }

  // 重置重试计数
  initRetryCount = 0
  console.log('[Enhanced3DEarth] ✅ 开始初始化，容器尺寸:', rect.width, 'x', rect.height)

  // 销毁旧实例
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }

  // 创建新实例
  chartInstance = echarts.init(chartContainer.value)

  // 绑定事件
  chartInstance.on('click', (params: any) => {
    if (params.componentType === 'series3D' && params.data?.dataInfo) {
      emit('pointClick', params.data.dataInfo)
    }
  })

  chartInstance.on('mouseover', (params: any) => {
    if (params.componentType === 'series3D' && params.data?.dataInfo) {
      emit('pointHover', params.data.dataInfo)
    }
  })

  // 初始化配置
  updateChart()
  isInitialized.value = true

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
  if (!chartInstance) return

  const { scatterData, connectionsData } = processedData.value

  console.log('[Enhanced3DEarth] updateChart - visualMode:', visualMode.value)
  console.log('[Enhanced3DEarth] updateChart - scatterData:', scatterData.length)
  console.log('[Enhanced3DEarth] updateChart - connectionsData:', connectionsData.length)
  console.log('[Enhanced3DEarth] updateChart - showConnections:', props.showConnections)

  // 确定要显示的系列
  const series: any[] = []

  // 散点系列 - 增强科技感
  if (visualMode.value === 'scatter' || visualMode.value === 'both') {
    series.push({
      type: 'scatter3D',
      coordinateSystem: 'globe',
      data: scatterData,
      blendMode: 'lighter',
      symbol: 'circle',
      symbolSize: (params: any) => {
        return (params.symbolSize || 10) * 1.2
      },
      itemStyle: {
        opacity: 0.95,
        shadowBlur: 12,
        shadowColor: 'rgba(34, 211, 238, 0.8)',
        borderWidth: 1,
        borderColor: 'rgba(34, 211, 238, 0.6)'
      },
      emphasis: {
        label: {
          show: true,
          formatter: (params: any) => {
            return params.data?.dataInfo?.name || ''
          },
          textStyle: {
            color: '#22d3ee',
            fontSize: 12,
            fontWeight: 'bold',
            backgroundColor: 'rgba(0, 0, 0, 0.8)',
            padding: [4, 8],
            borderRadius: 4
          }
        },
        itemStyle: {
          opacity: 1,
          shadowBlur: 20,
          shadowColor: 'rgba(34, 211, 238, 1)',
          borderWidth: 2
        }
      },
      zlevel: 10,
      silent: false
    })
  }

  // 流线系列 - 增强科技感和可见度
  if (visualMode.value === 'flow' || visualMode.value === 'both') {
    series.push({
      type: 'lines3D',
      coordinateSystem: 'globe',
      data: connectionsData,
      blendMode: 'lighter',
      lineStyle: {
        width: 3,          // 增加线宽
        opacity: 0.9,      // 提高不透明度
        color: '#00d2ff'   // 更亮的青色
      },
      effect: {
        show: true,
        period: 4,         // 加快速度
        trailWidth: 4,     // 增加轨迹宽度
        trailLength: 0.3,  // 增加轨迹长度
        trailOpacity: 1,   // 完全不透明
        trailColor: '#22d3ee',
        constantSpeed: 60, // 加快移动速度
        symbol: 'circle',
        symbolSize: 6      // 增大符号大小
      },
      zlevel: 5,
      silent: true
    })
  }

  const option: any = {
    backgroundColor: '#0f172a',
    tooltip: {
      show: true,
      formatter: (params: any) => {
        const data = params.data?.dataInfo
        if (!data) return ''
        return `
          <div style="padding: 8px;">
            <div style="font-weight: bold; color: #22d3ee; margin-bottom: 6px;">${data.name}</div>
            <div style="font-size: 12px; color: #cbd5e1;">
              <div>连接数: <span style="color: white; font-weight: 600;">${data.connections?.toLocaleString()}</span></div>
              ${data.ips ? `<div>IP数量: <span style="color: white;">${data.ips.length}</span></div>` : ''}
            </div>
          </div>
        `
      },
      backgroundColor: 'rgba(15, 23, 42, 0.95)',
      borderColor: '#06b6d4',
      borderWidth: 1,
      textStyle: {
        color: '#f8fafc'
      }
    },
    globe: {
      baseTexture: '/textures/earth.jpg',
      heightTexture: '/textures/bathymetry.jpg',
      displacementScale: 0.04,
      shading: 'realistic',
      environment: '/textures/starfield.jpg',
      realisticMaterial: {
        roughness: 0.8,
        metalness: 0.2,
        detailTexture: '#000'
      },
      postEffect: {
        enable: true,
        bloom: {
          enable: true,
          intensity: 0.15
        },
        SSAO: {
          enable: false
        }
      },
      light: {
        ambient: {
          intensity: 0.5
        },
        main: {
          intensity: 1.8,
          shadow: false
        }
      },
      viewControl: {
        autoRotate: autoRotate.value,
        autoRotateSpeed: 8,
        alpha: viewParams.alpha,
        beta: viewParams.beta,
        distance: 180,
        minDistance: 100,
        maxDistance: 400,
        damping: 0.85,
        rotateSensitivity: 1,
        zoomSensitivity: 1
      },
      layers: [],
      atmosphere: {
        show: true,
        offset: 5,
        color: '#06b6d4',
        glowPower: 2.5,
        innerGlowPower: 1.5
      }
    },
    series
  }

  chartInstance.setOption(option, true)
}

// 切换自动旋转
const toggleAutoRotate = () => {
  autoRotate.value = !autoRotate.value
  updateChart()
}

// 重置视角
const resetView = () => {
  if (!chartInstance) return
  chartInstance.setOption({
    globe: {
      viewControl: {
        alpha: viewParams.alpha,
        beta: viewParams.beta,
        distance: 200
      }
    }
  })
}

// 设置可视模式
const setVisualMode = (mode: 'scatter' | 'flow' | 'both') => {
  visualMode.value = mode
  updateChart()
}

// 处理窗口大小变化
const handleResize = () => {
  chartInstance?.resize()
}

// 监听数据变化
watch(
  () => [props.data, props.showConnections, props.defaultLocation],
  () => {
    nextTick(() => {
      if (chartInstance) {
        if (isInitialized.value) {
          updateChart()
        }
      } else {
        // 如果图表未初始化（容器之前为0），尝试重新初始化
        console.log('[Enhanced3DEarth] 数据变化，尝试重新初始化图表')
        initRetryCount = 0
        initChart()
      }
    })
  },
  { deep: true }
)

// 监听visualMode变化，触发图表更新
watch(visualMode, () => {
  nextTick(() => {
    if (chartInstance && isInitialized.value) {
      console.log('[Enhanced3DEarth] visualMode变化，更新图表:', visualMode.value)
      updateChart()
    }
  })
})

watch(
  () => props.loading,
  (newVal) => {
    if (!newVal) {
      nextTick(() => {
        if (chartInstance) {
          chartInstance.resize()
        } else {
          // 加载完成但图表未初始化，尝试初始化
          console.log('[Enhanced3DEarth] 加载完成，尝试初始化图表')
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
          console.log('[Enhanced3DEarth] 容器变为可见，尝试初始化图表')
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
  
  // 设置可见性观察器
  setupVisibilityObserver()
  
  // 延迟初始化以确保DOM完全渲染
  setTimeout(() => {
    initChart()
  }, 100)
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
  getInstance: () => chartInstance,
  toggleAutoRotate,
  resetView,
  setVisualMode
})
</script>

<style scoped>
/* 主容器 */
.earth-3d-container {
  width: 100%;
  height: 100%;
  position: relative;
  background: #0f172a;
  overflow: hidden;
}

/* 图表容器 */
.chart-container {
  width: 100%;
  height: 100%;
}

/* 加载遮罩 */
.loading-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.8);
  backdrop-filter: blur(4px);
  z-index: 20;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid transparent;
  border-top-color: #06b6d4;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.loading-text {
  color: #cbd5e1;
  font-size: 14px;
  margin: 0;
}

/* 控制面板 (右上角) */
.control-panel {
  position: absolute;
  top: 16px;
  right: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 10;
}

.control-btn {
  padding: 8px;
  border-radius: 8px;
  background: rgba(51, 65, 85, 0.9);
  backdrop-filter: blur(8px);
  border: 1px solid #475569;
  color: #cbd5e1;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.control-btn:hover {
  background: rgba(71, 85, 105, 0.9);
  color: white;
  border-color: #64748b;
}

.control-btn.btn-active {
  background: rgba(22, 163, 74, 0.9);
  color: white;
  border-color: #16a34a;
}

.control-btn.btn-active:hover {
  background: rgba(21, 128, 61, 0.9);
}

.control-btn .icon {
  width: 20px;
  height: 20px;
}

/* 模式切换器 */
.mode-switcher {
  background: rgba(30, 41, 59, 0.9);
  backdrop-filter: blur(8px);
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mode-btn {
  width: 100%;
  padding: 6px 8px;
  border-radius: 6px;
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}

.mode-btn:hover {
  background: rgba(71, 85, 105, 0.5);
  color: white;
}

.mode-btn.active {
  background: #2563eb;
  color: white;
}

/* 统计信息 (左下角) */
.stats-info {
  position: absolute;
  bottom: 16px;
  left: 16px;
  background: rgba(30, 41, 59, 0.9);
  backdrop-filter: blur(8px);
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 12px;
  z-index: 10;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.stats-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 12px;
}

.stat-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.pulse-cyan {
  background: #22d3ee;
}

.pulse-green {
  background: #22c55e;
}

.stat-label {
  color: #94a3b8;
}

.stat-value {
  color: white;
  font-weight: 600;
}

/* 动画 */
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>

