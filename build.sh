#!/bin/bash

# FastMonitor 快速编译脚本（当前平台）

set -e

VERSION="1.0.0"
APP_NAME="fastmonitor"

echo "=========================================="
echo "  FastMonitor 编译"
echo "  版本: $VERSION"
echo "=========================================="
echo ""

# 检查 wails
WAILS_CMD="wails"
if ! command -v wails &> /dev/null; then
    # 尝试使用 go/bin 路径
    if [ -f "$HOME/go/bin/wails" ]; then
        WAILS_CMD="$HOME/go/bin/wails"
        echo "✓ 使用 wails: $WAILS_CMD"
    else
        echo "❌ 错误: wails 未安装"
        echo "请运行: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    fi
fi

# 构建前端
echo "🎨 构建前端..."
cd frontend
npm run build
cd ..

# 编译
echo ""
echo "🔨 编译应用..."
$WAILS_CMD build -clean

# 复制资源文件
echo ""
echo "📋 复制资源文件..."

if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS - 复制到 .app 包外部，方便用户更新
    BUILD_DIR="build/bin"
    APP_BUNDLE="$BUILD_DIR/fastmonitor.app"
    ENTITLEMENTS="build/darwin/entitlements.plist"
    
    # 代码签名（macOS Sequoia+ 必需，否则即使 sudo 也无法访问 BPF 设备）
    echo ""
    echo "🔐 代码签名..."
    if [ -f "$ENTITLEMENTS" ]; then
        # 签名可执行文件（带 entitlements 和 hardened runtime）
        codesign --force --options runtime --entitlements "$ENTITLEMENTS" --sign - "$APP_BUNDLE/Contents/MacOS/fastmonitor"
        # 签名整个 .app 包
        codesign --force --options runtime --entitlements "$ENTITLEMENTS" --sign - "$APP_BUNDLE"
        echo "  ✓ 已使用 entitlements 签名（Hardened Runtime）"
        echo "  ✓ 授权: network.client, network.server, device.network"
    else
        codesign --force --options runtime --sign - "$APP_BUNDLE/Contents/MacOS/fastmonitor"
        codesign --force --options runtime --sign - "$APP_BUNDLE"
        echo "  ⚠️  未找到 entitlements.plist，仅做基础签名"
    fi
    
    # 创建外部 data/geoip 目录
    mkdir -p "$BUILD_DIR/data/geoip"
    
    # 复制 mmdb 文件到外部目录
    echo "  复制 GeoIP 数据库到外部目录..."
    if ls data/geoip/*.mmdb 1> /dev/null 2>&1; then
        if [ -w "$BUILD_DIR/data/geoip" ]; then
            cp data/geoip/*.mmdb "$BUILD_DIR/data/geoip/"
        else
            sudo cp data/geoip/*.mmdb "$BUILD_DIR/data/geoip/"
            sudo chmod -R 755 "$BUILD_DIR/data"
        fi
        echo "  ✓ 已复制 $(ls data/geoip/*.mmdb | wc -l | tr -d ' ') 个 .mmdb 文件"
        echo "  文件列表:"
        ls -lh "$BUILD_DIR/data/geoip/"*.mmdb | awk '{print "    - " $9 " (" $5 ")"}'
    else
        echo "  ⚠️  警告: 未找到 data/geoip/*.mmdb 文件"
    fi
    
    # 复制配置文件到外部目录
    if [ -f "config.yaml" ]; then
        cp config.yaml "$BUILD_DIR/" 2>/dev/null || sudo cp config.yaml "$BUILD_DIR/"
        echo "  ✓ 已复制 config.yaml"
    fi
    
    echo ""
    echo "✅ 编译成功！"
    echo "📍 应用位置: build/bin/fastmonitor.app"
    echo "📂 数据库位置: build/bin/data/geoip/ (外部目录，可直接更新)"
    echo ""
    echo "💡 优势:"
    echo "  - 数据库文件在 .app 包外部"
    echo "  - 可以直接替换 .mmdb 文件更新数据库"
    echo "  - 无需重新编译应用"
    echo ""
    echo "运行方式:"
    echo "  cd build/bin"
    echo "  sudo ./fastmonitor.app/Contents/MacOS/fastmonitor"
    
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    # Windows
    mkdir -p "build/bin/data/geoip"
    cp -r data/geoip/*.mmdb "build/bin/data/geoip/" 2>/dev/null || true
    cp config.yaml "build/bin/" 2>/dev/null || true
    
    echo ""
    echo "✅ 编译成功！"
    echo "📍 应用位置: build/bin/fastmonitor.exe"
    echo ""
    echo "运行方式:"
    echo "  以管理员身份运行 fastmonitor.exe"
    
else
    # Linux
    mkdir -p "build/bin/data/geoip"
    cp -r data/geoip/*.mmdb "build/bin/data/geoip/" 2>/dev/null || true
    cp config.yaml "build/bin/" 2>/dev/null || true
    
    echo ""
    echo "✅ 编译成功！"
    echo "📍 应用位置: build/bin/fastmonitor"
    echo ""
    echo "运行方式:"
    echo "  sudo ./build/bin/fastmonitor"
fi

echo ""
