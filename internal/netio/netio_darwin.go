// +build darwin

package netio

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"fastmonitor/pkg/model"
)

// DarwinInterfaceInfo macOS网络接口详细信息
type DarwinInterfaceInfo struct {
	Name        string
	DisplayName string
	Type        string // ethernet, wifi, thunderbolt, bridge, loopback, virtual
	MediaType   string // 媒体类型
	Status      string // active, inactive
	MTU         int
	Addresses   []string
	MacAddress  string
	Speed       string // 链路速度
	IsPhysical  bool
	IsWifi      bool
	IsUp        bool
}

// GetDarwinInterfaces 获取macOS网络接口详细信息
func GetDarwinInterfaces() ([]DarwinInterfaceInfo, error) {
	interfaces := make([]DarwinInterfaceInfo, 0)

	// 获取所有网络接口
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("get interfaces: %w", err)
	}

	for _, iface := range ifaces {
		info := DarwinInterfaceInfo{
			Name:       iface.Name,
			MTU:        iface.MTU,
			MacAddress: iface.HardwareAddr.String(),
			IsUp:       iface.Flags&net.FlagUp != 0,
		}

		// 获取IP地址
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			info.Addresses = append(info.Addresses, addr.String())
		}

		// 识别接口类型
		info.Type, info.DisplayName, info.IsPhysical, info.IsWifi = identifyDarwinInterface(iface.Name)

		// 获取媒体类型和状态
		info.MediaType, info.Status, info.Speed = getDarwinInterfaceMedia(iface.Name)

		interfaces = append(interfaces, info)
	}

	return interfaces, nil
}

// identifyDarwinInterface 识别macOS网络接口类型
func identifyDarwinInterface(name string) (ifaceType, displayName string, isPhysical, isWifi bool) {
	nameLower := strings.ToLower(name)

	// 回环接口
	if name == "lo0" {
		return "loopback", "Loopback", false, false
	}

	// Wi-Fi 接口 (通常是 en0 或 en1)
	if strings.HasPrefix(nameLower, "en") {
		// 使用 networksetup 检查是否是 Wi-Fi
		if isWifiInterface(name) {
			return "wifi", "Wi-Fi (" + name + ")", true, true
		}
		// 以太网
		return "ethernet", "Ethernet (" + name + ")", true, false
	}

	// Thunderbolt 桥接
	if strings.HasPrefix(nameLower, "bridge") {
		return "bridge", "Thunderbolt Bridge (" + name + ")", false, false
	}

	// Thunderbolt 网络
	if strings.HasPrefix(nameLower, "thunderbolt") {
		return "thunderbolt", "Thunderbolt (" + name + ")", true, false
	}

	// Apple Wireless Direct Link (AirDrop等)
	if strings.HasPrefix(nameLower, "awdl") {
		return "awdl", "Apple Wireless Direct Link", false, false
	}

	// Low Latency WLAN (Apple Watch等)
	if strings.HasPrefix(nameLower, "llw") {
		return "llw", "Low Latency WLAN", false, false
	}

	// VPN 隧道
	if strings.HasPrefix(nameLower, "utun") {
		return "vpn", "VPN Tunnel (" + name + ")", false, false
	}

	// PPP 连接
	if strings.HasPrefix(nameLower, "ppp") {
		return "ppp", "PPP Connection (" + name + ")", false, false
	}

	// gif 隧道
	if strings.HasPrefix(nameLower, "gif") {
		return "tunnel", "GIF Tunnel (" + name + ")", false, false
	}

	// stf 6to4 隧道
	if strings.HasPrefix(nameLower, "stf") {
		return "tunnel", "6to4 Tunnel (" + name + ")", false, false
	}

	// XHC (USB)
	if strings.HasPrefix(nameLower, "xhc") {
		return "usb", "USB (" + name + ")", false, false
	}

	// ap (Access Point)
	if strings.HasPrefix(nameLower, "ap") {
		return "ap", "Access Point (" + name + ")", false, false
	}

	// 默认
	return "unknown", name, false, false
}

// isWifiInterface 检查是否是Wi-Fi接口
func isWifiInterface(name string) bool {
	// 使用 networksetup 获取 Wi-Fi 接口名称
	cmd := exec.Command("networksetup", "-listallhardwareports")
	output, err := cmd.Output()
	if err != nil {
		// 回退到简单判断
		return name == "en0"
	}

	lines := strings.Split(string(output), "\n")
	isWifiSection := false
	for _, line := range lines {
		if strings.Contains(line, "Wi-Fi") || strings.Contains(line, "AirPort") {
			isWifiSection = true
			continue
		}
		if isWifiSection && strings.HasPrefix(line, "Device:") {
			device := strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			if device == name {
				return true
			}
			isWifiSection = false
		}
		if strings.HasPrefix(line, "Hardware Port:") {
			isWifiSection = false
		}
	}

	return false
}

// getDarwinInterfaceMedia 获取接口媒体信息
func getDarwinInterfaceMedia(name string) (mediaType, status, speed string) {
	cmd := exec.Command("ifconfig", name)
	output, err := cmd.Output()
	if err != nil {
		return "", "unknown", ""
	}

	outputStr := string(output)

	// 解析状态
	if strings.Contains(outputStr, "status: active") {
		status = "active"
	} else if strings.Contains(outputStr, "status: inactive") {
		status = "inactive"
	} else {
		status = "unknown"
	}

	// 解析媒体类型
	mediaRegex := regexp.MustCompile(`media: (.+)`)
	if matches := mediaRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		mediaType = strings.TrimSpace(matches[1])

		// 提取速度
		speedRegex := regexp.MustCompile(`(\d+)base`)
		if speedMatches := speedRegex.FindStringSubmatch(mediaType); len(speedMatches) > 1 {
			speedVal, _ := strconv.Atoi(speedMatches[1])
			if speedVal >= 1000 {
				speed = fmt.Sprintf("%d Gbps", speedVal/1000)
			} else {
				speed = fmt.Sprintf("%d Mbps", speedVal)
			}
		}
	}

	return
}

// GetDarwinNetworkServices 获取macOS网络服务列表
func GetDarwinNetworkServices() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	services := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过标题行和禁用的服务
		if line == "" || strings.HasPrefix(line, "An asterisk") || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}

	return services, nil
}

// GetDarwinWifiInfo 获取Wi-Fi详细信息
func GetDarwinWifiInfo() (map[string]string, error) {
	info := make(map[string]string)

	// 使用 airport 命令获取 Wi-Fi 信息
	cmd := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I")
	output, err := cmd.Output()
	if err != nil {
		return info, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			info[key] = value
		}
	}

	return info, nil
}

// EnhancedDarwinList 增强的macOS网络接口列表
func EnhancedDarwinList() ([]model.NetworkInterface, error) {
	darwinInterfaces, err := GetDarwinInterfaces()
	if err != nil {
		return nil, err
	}

	interfaces := make([]model.NetworkInterface, 0)

	for _, di := range darwinInterfaces {
		// 过滤掉不需要的接口
		if di.Type == "loopback" || di.Type == "awdl" || di.Type == "llw" ||
			di.Type == "tunnel" || di.Type == "usb" || di.Type == "ap" {
			continue
		}

		// 只显示活跃的或物理接口
		if !di.IsUp && !di.IsPhysical {
			continue
		}

		iface := model.NetworkInterface{
			Name:        di.Name,
			Description: di.DisplayName,
			Addresses:   di.Addresses,
			IsLoopback:  di.Type == "loopback",
			IsPhysical:  di.IsPhysical,
			IsUp:        di.IsUp,
		}

		// 添加额外信息到描述
		if di.Speed != "" {
			iface.Description += " - " + di.Speed
		}
		if di.Status == "active" {
			iface.Description += " [Active]"
		}

		interfaces = append(interfaces, iface)
	}

	// 按优先级排序：活跃的物理接口优先
	sortDarwinInterfaces(interfaces)

	return interfaces, nil
}

// sortDarwinInterfaces 排序接口
func sortDarwinInterfaces(interfaces []model.NetworkInterface) {
	// 简单的冒泡排序，按优先级排列
	for i := 0; i < len(interfaces); i++ {
		for j := i + 1; j < len(interfaces); j++ {
			if interfacePriority(interfaces[j]) > interfacePriority(interfaces[i]) {
				interfaces[i], interfaces[j] = interfaces[j], interfaces[i]
			}
		}
	}
}

// interfacePriority 计算接口优先级
func interfacePriority(iface model.NetworkInterface) int {
	priority := 0

	if iface.IsUp {
		priority += 100
	}
	if iface.IsPhysical {
		priority += 50
	}
	if len(iface.Addresses) > 0 {
		priority += 25
	}
	// Wi-Fi 通常是 en0
	if iface.Name == "en0" {
		priority += 10
	}
	// 以太网通常是 en1 或更高
	if strings.HasPrefix(iface.Name, "en") {
		priority += 5
	}

	return priority
}

// CheckDarwinPermission 检查macOS权限
func CheckDarwinPermission() error {
	// 检查 BPF 设备权限
	// macOS 上需要 root 权限或者配置 BPF 设备权限

	// 尝试读取 /dev/bpf0
	cmd := exec.Command("test", "-r", "/dev/bpf0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法访问 BPF 设备。请使用 sudo 运行，或配置 BPF 权限:\n" +
			"  sudo chown $(whoami):admin /dev/bpf*\n" +
			"  sudo chmod g+rw /dev/bpf*")
	}

	return nil
}

// SetupDarwinBPFPermission 设置BPF权限（需要sudo）
func SetupDarwinBPFPermission() error {
	// 这个函数需要 root 权限
	cmds := [][]string{
		{"chgrp", "admin", "/dev/bpf*"},
		{"chmod", "g+rw", "/dev/bpf*"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("设置 BPF 权限失败: %v", err)
		}
	}

	return nil
}
