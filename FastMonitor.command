#!/bin/bash

# ============================================
# FastMonitor macOS 启动脚本
# 用于编译后的应用，双击即可运行
# ============================================

set -e

# 获取脚本所在的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# 定义应用路径（在当前目录下）
APP_PATH="$SCRIPT_DIR/FastMonitor.app/Contents/MacOS/FastMonitor"
APP_DIR="$SCRIPT_DIR/FastMonitor.app"

# 清屏并显示欢迎信息
clear
echo "=============================================="
echo "  🚀 FastMonitor 网络流量分析工具"
echo "=============================================="
echo ""

# ============================================
# 步骤 1: 移除隔离属性（解决"文件已损坏"问题）
# ============================================
echo "[1/4] 🔓 移除系统隔离属性..."

# 移除脚本自身的隔离属性
xattr -d com.apple.quarantine "$0" 2>/dev/null || true

# 移除整个应用包的隔离属性（递归）
if [ -d "$APP_DIR" ]; then
    echo "      正在移除应用包隔离属性..."
    xattr -dr com.apple.quarantine "$APP_DIR" 2>/dev/null || true
fi

# 移除可执行文件的隔离属性
if [ -f "$APP_PATH" ]; then
    xattr -d com.apple.quarantine "$APP_PATH" 2>/dev/null || true
    chmod +x "$APP_PATH" 2>/dev/null || true
fi

# 移除 data 目录的隔离属性（如果存在）
if [ -d "$SCRIPT_DIR/data" ]; then
    xattr -dr com.apple.quarantine "$SCRIPT_DIR/data" 2>/dev/null || true
fi

echo "      ✓ 完成"
echo ""

# ============================================
# 步骤 2: 检查应用是否存在
# ============================================
echo "[2/4] 📦 检查应用..."

if [ ! -f "$APP_PATH" ]; then
    echo ""
    echo "      ❌ 错误: 找不到 FastMonitor 应用"
    echo ""
    echo "      期望路径: $APP_PATH"
    echo "      当前目录: $SCRIPT_DIR"
    echo ""
    echo "      💡 提示: 请将此脚本复制到 build/bin/ 目录下运行"
    echo "      或者运行: cd ../../ && ./build.sh"
    echo ""
    read -p "按回车键退出..." dummy
    exit 1
fi

echo "      ✓ 应用已找到"
echo ""

# ============================================
# 步骤 3: 配置 BPF 设备权限
# ============================================
echo "[3/4] 🔧 配置网络抓包权限..."

BPF_ACCESSIBLE=false

# 检查当前用户是否能访问 BPF 设备
if [ -r /dev/bpf0 ] 2>/dev/null; then
    BPF_ACCESSIBLE=true
    echo "      ✓ BPF 设备已可访问"
else
    echo "      ⚠️  当前用户无法访问 BPF 设备（需要管理员权限配置）"
    echo "      正在请求权限配置 BPF 设备..."
    
    # 尝试配置 BPF 权限
    if sudo chown $(whoami):admin /dev/bpf* 2>/dev/null && sudo chmod g+rw /dev/bpf* 2>/dev/null; then
        BPF_ACCESSIBLE=true
        echo "      ✓ BPF 设备权限已配置"
    else
        echo "      ⚠️  BPF 设备权限配置失败，将使用 sudo 运行"
    fi
fi

echo ""

# ============================================
# 步骤 4: 启动应用
# ============================================
echo "[4/4] 🚀 启动应用..."
echo ""

# 确保可执行权限
chmod +x "$APP_PATH" 2>/dev/null || true

# 检查是否已有 root 权限
if [ "$EUID" -eq 0 ]; then
    echo "      ✓ 已有管理员权限，正在启动..."
    echo ""
    exec "$APP_PATH"
elif [ "$BPF_ACCESSIBLE" = true ]; then
    echo "      ✓ BPF 设备已可访问，直接启动..."
    echo ""
    exec "$APP_PATH"
else
    echo "      ⚠️  FastMonitor 需要管理员权限进行网络抓包"
    echo "      （这是 macOS 系统要求，类似 Wireshark）"
    echo ""
    
    # 尝试使用 AppleScript 图形化授权（更友好）
    if osascript -e "do shell script \"'$APP_PATH' > /dev/null 2>&1 &\" with administrator privileges" 2>/dev/null; then
        # 成功启动
        echo "      ✓ 应用已在后台启动"
        echo ""
        echo "=============================================="
        echo "  FastMonitor 正在运行"
        echo "=============================================="
        echo ""
        echo "💡 提示:"
        echo "  - 应用窗口应该已经打开"
        echo "  - 如果没有看到窗口，请检查 Dock"
        echo "  - 关闭此窗口不会影响应用运行"
        echo ""
        exit 0
    else
        # AppleScript 失败（可能用户取消），尝试命令行 sudo
        echo "      使用命令行授权..."
        echo "      请输入密码:"
        echo ""
        
        if sudo "$APP_PATH"; then
            echo ""
            echo "      ✓ 应用已启动"
            echo ""
            exit 0
        else
            echo ""
            echo "      ❌ 启动失败"
            echo ""
            echo "      可能的原因:"
            echo "        1. 用户取消了授权"
            echo "        2. 密码输入错误"
            echo "        3. 应用未正确签名（macOS Sequoia+ 要求）"
            echo ""
            echo "      解决方法:"
            echo "        1. 重新编译并签名: ./build.sh"
            echo "        2. 配置 BPF 权限: sudo chown \$(whoami):admin /dev/bpf* && sudo chmod g+rw /dev/bpf*"
            echo "        3. 在终端中运行: sudo '$APP_PATH'"
            echo ""
            read -p "按回车键退出..." dummy
            exit 1
        fi
    fi
fi
