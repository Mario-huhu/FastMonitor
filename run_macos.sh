#!/bin/bash

# FastMonitor macOS 启动脚本
# 自动请求 sudo 权限并运行

set -e

APP_PATH="build/bin/fastmonitor.app/Contents/MacOS/fastmonitor"

echo "=========================================="
echo "  FastMonitor - macOS 启动"
echo "=========================================="
echo ""

# 检查应用是否存在
if [ ! -f "$APP_PATH" ]; then
    echo "❌ 错误: 找不到应用程序"
    echo "   路径: $APP_PATH"
    echo ""
    echo "请先编译应用:"
    echo "  ./build.sh"
    exit 1
fi

echo "📍 应用位置: $APP_PATH"
echo ""

# 检查是否已有 root 权限
if [ "$EUID" -eq 0 ]; then
    echo "✓ 已有 root 权限"
    echo ""
    echo "🚀 启动 FastMonitor..."
    exec "$APP_PATH"
else
    echo "⚠️  需要管理员权限进行网络抓包"
    echo ""
    echo "请输入密码以继续..."
    echo ""
    
    # 使用 sudo 运行
    sudo "$APP_PATH"
fi
