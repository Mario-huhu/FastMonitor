<template>
  <div id="app" class="app-container" :class="{ 'macos-fullsize': isMacOS }">
    <!-- macOS 标题栏拖拽区域 -->
    <div v-if="isMacOS" class="macos-titlebar-drag-region"></div>
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

const isMacOS = ref(false)

onMounted(() => {
  // 检测是否为 macOS
  isMacOS.value = navigator.platform.toLowerCase().includes('mac')
  
  // 检测系统主题偏好
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)')
  
  // 设置初始主题
  if (prefersDark.matches) {
    document.documentElement.classList.add('dark')
    document.documentElement.classList.remove('light')
  } else {
    document.documentElement.classList.add('light')
    document.documentElement.classList.remove('dark')
  }
  
  // 监听系统主题变化
  prefersDark.addEventListener('change', (e) => {
    if (e.matches) {
      document.documentElement.classList.add('dark')
      document.documentElement.classList.remove('light')
    } else {
      document.documentElement.classList.add('light')
      document.documentElement.classList.remove('dark')
    }
  })
})
</script>

<style lang="scss">
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  font-size: 13px;
}

.app-container {
  width: 100%;
  height: 100vh;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  overflow: hidden;
}

/* macOS 全尺寸内容支持 */
.macos-fullsize {
  padding-top: 28px; /* 为标题栏留出空间 */
}

/* macOS 标题栏拖拽区域 */
.macos-titlebar-drag-region {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 28px;
  -webkit-app-region: drag; /* 允许拖拽窗口 */
  z-index: 9999;
  background: transparent;
}

/* 确保按钮等交互元素不被拖拽区域影响 */
.macos-titlebar-drag-region button,
.macos-titlebar-drag-region a,
.macos-titlebar-drag-region input,
.macos-titlebar-drag-region select {
  -webkit-app-region: no-drag;
}

/* 专业级深色主题 - Cyberpunk Tech风格 */
.dark {
  // 背景色 - 深邃科技感
  --el-bg-color: #0a0e27;
  --el-bg-color-page: #050811;
  --el-bg-color-overlay: #10162f;
  
  // 填充色 - 多层次深度
  --el-fill-color-blank: #0a0e27;
  --el-fill-color: #10162f;
  --el-fill-color-light: #151d3b;
  --el-fill-color-lighter: #1a2445;
  --el-fill-color-extra-light: #202b52;
  
  // 边框色 - 荧光边框
  --el-border-color: #1e3a8a;
  --el-border-color-light: #2563eb;
  --el-border-color-lighter: #3b82f6;
  --el-border-color-extra-light: #60a5fa;
  
  // 文字色 - 高亮对比
  --el-text-color-primary: #e0e7ff;
  --el-text-color-regular: #c7d2fe;
  --el-text-color-secondary: #a5b4fc;
  --el-text-color-placeholder: #6366f1;
  --el-text-color-disabled: #4c51bf;
  
  // 主题色 - 电光蓝
  --el-color-primary: #3b82f6;
  --el-color-primary-light-3: #60a5fa;
  --el-color-primary-light-5: #93c5fd;
  --el-color-primary-light-7: #bfdbfe;
  --el-color-primary-light-8: #dbeafe;
  --el-color-primary-light-9: #eff6ff;
  --el-color-primary-dark-2: #2563eb;
  
  // 成功色 - 荧光绿
  --el-color-success: #10b981;
  --el-color-success-light-3: #34d399;
  --el-color-success-light-5: #6ee7b7;
  
  // 警告色 - 琥珀金
  --el-color-warning: #f59e0b;
  --el-color-warning-light-3: #fbbf24;
  --el-color-warning-light-5: #fcd34d;
  
  // 危险色 - 赛博红
  --el-color-danger: #ef4444;
  --el-color-danger-light-3: #f87171;
  --el-color-danger-light-5: #fca5a5;
  
  // 信息色 - 紫罗兰
  --el-color-info: #8b5cf6;
  --el-color-info-light-3: #a78bfa;
  --el-color-info-light-5: #c4b5fd;
  
  // 赛博朋克渐变背景
  background: 
    radial-gradient(ellipse at top, rgba(59, 130, 246, 0.15) 0%, transparent 60%),
    radial-gradient(ellipse at bottom, rgba(139, 92, 246, 0.1) 0%, transparent 60%),
    linear-gradient(135deg, #050811 0%, #0a0e27 50%, #10162f 100%);
  
  // 霓虹阴影
  --el-box-shadow: 0 8px 32px rgba(59, 130, 246, 0.3), 0 0 64px rgba(139, 92, 246, 0.1);
  --el-box-shadow-light: 0 4px 16px rgba(59, 130, 246, 0.2);
  --el-box-shadow-lighter: 0 2px 8px rgba(59, 130, 246, 0.15);
  --el-box-shadow-dark: 0 16px 64px rgba(59, 130, 246, 0.4), 0 0 96px rgba(139, 92, 246, 0.2);
}

/* 现代简约浅色主题 - Material Design风格 */
.light {
  // 背景色 - 纯净明亮
  --el-bg-color: #ffffff;
  --el-bg-color-page: #fafafa;
  --el-bg-color-overlay: #ffffff;
  
  // 填充色 - 轻盈舒适
  --el-fill-color-blank: #ffffff;
  --el-fill-color: #f5f5f5;
  --el-fill-color-light: #eeeeee;
  --el-fill-color-lighter: #e0e0e0;
  --el-fill-color-extra-light: #bdbdbd;
  
  // 边框色 - 精致界限
  --el-border-color: #e0e0e0;
  --el-border-color-light: #eeeeee;
  --el-border-color-lighter: #f5f5f5;
  --el-border-color-extra-light: #fafafa;
  
  // 文字色 - 优雅对比
  --el-text-color-primary: #212121;
  --el-text-color-regular: #424242;
  --el-text-color-secondary: #757575;
  --el-text-color-placeholder: #9e9e9e;
  --el-text-color-disabled: #bdbdbd;
  
  // 主题色 - 活力蓝
  --el-color-primary: #1976d2;
  --el-color-primary-light-3: #42a5f5;
  --el-color-primary-light-5: #90caf9;
  --el-color-primary-light-7: #bbdefb;
  --el-color-primary-light-8: #e3f2fd;
  --el-color-primary-light-9: #f3f9ff;
  --el-color-primary-dark-2: #1565c0;
  
  // 成功色 - 自然绿
  --el-color-success: #388e3c;
  --el-color-success-light-3: #66bb6a;
  --el-color-success-light-5: #a5d6a7;
  
  // 警告色 - 温暖橙
  --el-color-warning: #f57c00;
  --el-color-warning-light-3: #ffb74d;
  --el-color-warning-light-5: #ffcc80;
  
  // 危险色 - 明确红
  --el-color-danger: #d32f2f;
  --el-color-danger-light-3: #e57373;
  --el-color-danger-light-5: #ef9a9a;
  
  // 信息色 - 紫罗兰
  --el-color-info: #7e57c2;
  --el-color-info-light-3: #9575cd;
  --el-color-info-light-5: #b39ddb;
  
  // Material阴影
  --el-box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04);
  --el-box-shadow-light: 0 1px 3px rgba(0, 0, 0, 0.1);
  --el-box-shadow-lighter: 0 1px 2px rgba(0, 0, 0, 0.06);
  --el-box-shadow-dark: 0 8px 16px rgba(0, 0, 0, 0.15), 0 0 12px rgba(0, 0, 0, 0.06);
  
  // 清爽渐变背景
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 50%, #f5f5f5 100%);
}

/* 高级滚动条 - Mac 风格 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: rgba(128, 128, 128, 0.4);
  border-radius: 4px;
  transition: all 0.3s ease;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(128, 128, 128, 0.6);
}

/* 隐藏滚动条但保持滚动功能 - Mac 风格 */
.el-table__body-wrapper::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.el-table__body-wrapper::-webkit-scrollbar-thumb {
  background: rgba(128, 128, 128, 0.3);
  border-radius: 3px;
}

.el-table__body-wrapper::-webkit-scrollbar-thumb:hover {
  background: rgba(128, 128, 128, 0.5);
}
</style>
