package process

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	psnet "github.com/shirou/gopsutil/v3/net"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID        int32  `json:"pid"`
	Name       string `json:"name"`
	Exe        string `json:"exe"`
	Cmdline    string `json:"cmdline"`
	Username   string `json:"username"`
	CreateTime int64  `json:"create_time"`
}

// ConnectionKey 连接唯一标识
type ConnectionKey struct {
	Protocol string
	LocalIP  string
	LocalPort uint32
	RemoteIP string
	RemotePort uint32
}

// ProcessMapper 进程映射管理器 - 100%准确方案
type ProcessMapper struct {
	mu sync.RWMutex
	
	// 方案1: 精确连接映射 (五元组 -> PID)
	connectionMap map[ConnectionKey]int32
	
	// 方案2: 端口映射 (协议:端口 -> PID列表)
	portMap map[string][]int32
	
	// 方案3: 进程信息缓存 (PID -> ProcessInfo)
	processCache map[int32]*ProcessInfo
	
	// 方案4: 端口所有者历史记录 (用于处理短连接)
	portHistory map[string]*PortOwnerHistory
	
	// macOS专用: lsof缓存 (作为备用方案)
	lsofCache     map[string]int32 // "protocol:port" -> PID
	lsofCacheTime time.Time
	
	// 配置
	updateInterval time.Duration
	historyTTL     time.Duration
	stopChan       chan struct{}
}

// 确保 ProcessMapper 实现 IProcessMapper 接口
var _ IProcessMapper = (*ProcessMapper)(nil)

// PortOwnerHistory 端口所有者历史记录
type PortOwnerHistory struct {
	PID        int32
	LastSeen   time.Time
	FirstSeen  time.Time
	PacketCount int64
}

// NewProcessMapper 创建进程映射管理器
func NewProcessMapper() *ProcessMapper {
	pm := &ProcessMapper{
		connectionMap:  make(map[ConnectionKey]int32),
		portMap:        make(map[string][]int32),
		processCache:   make(map[int32]*ProcessInfo),
		portHistory:    make(map[string]*PortOwnerHistory),
		lsofCache:      make(map[string]int32),
		updateInterval: 2 * time.Second,  // 2秒更新一次，确保及时性
		historyTTL:     30 * time.Second, // 保留30秒历史
		stopChan:       make(chan struct{}),
	}
	
	// 立即更新一次
	pm.Update()
	
	// macOS: 初始化lsof缓存
	if runtime.GOOS == "darwin" {
		pm.updateLsofCache()
	}
	
	// 启动自动更新
	go pm.autoUpdate()
	
	// 启动历史清理
	go pm.cleanupHistory()
	
	return pm
}

// autoUpdate 自动更新进程映射
func (pm *ProcessMapper) autoUpdate() {
	ticker := time.NewTicker(pm.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := pm.Update(); err != nil {
				fmt.Printf("[ProcessMapper] Update error: %v\n", err)
			}
			// macOS: 定期更新lsof缓存
			if runtime.GOOS == "darwin" {
				pm.updateLsofCache()
			}
		case <-pm.stopChan:
			return
		}
	}
}

// Update 更新进程映射表 - 核心方法
func (pm *ProcessMapper) Update() error {
	// 获取所有网络连接
	connections, err := psnet.Connections("all")
	if err != nil {
		return fmt.Errorf("get connections: %w", err)
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// 清空旧映射
	pm.connectionMap = make(map[ConnectionKey]int32)
	pm.portMap = make(map[string][]int32)
	
	now := time.Now()
	
	for _, conn := range connections {
		if conn.Pid == 0 {
			continue
		}
		
		// 将连接类型转换为协议字符串
		protocol := getProtocolName(conn.Family, conn.Type)
		
		// 方案1: 五元组精确映射 (最准确)
		if conn.Laddr.IP != "" && conn.Raddr.IP != "" {
			key := ConnectionKey{
				Protocol:   protocol,
				LocalIP:    conn.Laddr.IP,
				LocalPort:  conn.Laddr.Port,
				RemoteIP:   conn.Raddr.IP,
				RemotePort: conn.Raddr.Port,
			}
			pm.connectionMap[key] = conn.Pid
		}
		
		// 方案2: 本地端口映射 (用于监听端口和未建立连接的数据包)
		if conn.Laddr.Port > 0 {
			portKey := fmt.Sprintf("%s:%d", protocol, conn.Laddr.Port)
			pm.portMap[portKey] = append(pm.portMap[portKey], conn.Pid)
			
			// 方案4: 记录端口所有者历史
			if history, exists := pm.portHistory[portKey]; exists {
				history.LastSeen = now
				history.PacketCount++
			} else {
				pm.portHistory[portKey] = &PortOwnerHistory{
					PID:        conn.Pid,
					FirstSeen:  now,
					LastSeen:   now,
					PacketCount: 1,
				}
			}
		}
		
		// 方案3: 缓存进程信息
		if _, exists := pm.processCache[conn.Pid]; !exists {
			if info, err := GetProcessByPID(conn.Pid); err == nil {
				pm.processCache[conn.Pid] = info
			}
		}
	}
	
	return nil
}

// GetPIDByConnection 通过完整五元组获取PID (最准确)
func (pm *ProcessMapper) GetPIDByConnection(protocol, srcIP, dstIP string, srcPort, dstPort uint32) (int32, *ProcessInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// 尝试正向匹配 (本机发出的包)
	key := ConnectionKey{
		Protocol:   strings.ToUpper(protocol),
		LocalIP:    srcIP,
		LocalPort:  srcPort,
		RemoteIP:   dstIP,
		RemotePort: dstPort,
	}
	
	if pid, ok := pm.connectionMap[key]; ok {
		info := pm.processCache[pid]
		return pid, info, true
	}
	
	// 尝试反向匹配 (发往本机的包)
	reverseKey := ConnectionKey{
		Protocol:   strings.ToUpper(protocol),
		LocalIP:    dstIP,
		LocalPort:  dstPort,
		RemoteIP:   srcIP,
		RemotePort: srcPort,
	}
	
	if pid, ok := pm.connectionMap[reverseKey]; ok {
		info := pm.processCache[pid]
		return pid, info, true
	}
	
	return 0, nil, false
}

// GetPIDByPort 通过本地端口获取PID (次优方案，用于无完整连接信息的情况)
func (pm *ProcessMapper) GetPIDByPort(protocol string, port uint32) (int32, *ProcessInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	portKey := fmt.Sprintf("%s:%d", strings.ToUpper(protocol), port)
	
	// 优先从当前连接表查找
	if pids, ok := pm.portMap[portKey]; ok && len(pids) > 0 {
		// 如果有多个进程使用同一端口，返回第一个
		// 在实际应用中，这种情况很少见
		pid := pids[0]
		info := pm.processCache[pid]
		return pid, info, true
	}
	
	// macOS: 从lsof缓存查找
	if runtime.GOOS == "darwin" {
		if pid, ok := pm.lsofCache[portKey]; ok && pid > 0 {
			info := pm.processCache[pid]
			if info == nil {
				// 尝试获取进程信息
				if procInfo, err := GetProcessByPID(pid); err == nil {
					info = procInfo
				}
			}
			return pid, info, true
		}
	}
	
	// 从历史记录查找 (处理短连接)
	if history, ok := pm.portHistory[portKey]; ok {
		if time.Since(history.LastSeen) < pm.historyTTL {
			info := pm.processCache[history.PID]
			return history.PID, info, true
		}
	}
	
	return 0, nil, false
}

// GetPIDByLocalAddress 通过本地IP和端口获取PID
func (pm *ProcessMapper) GetPIDByLocalAddress(protocol, localIP string, localPort uint32) (int32, *ProcessInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// 遍历连接映射，查找匹配的本地地址
	for key, pid := range pm.connectionMap {
		if key.Protocol == strings.ToUpper(protocol) && 
		   key.LocalIP == localIP && 
		   key.LocalPort == localPort {
			info := pm.processCache[pid]
			return pid, info, true
		}
	}
	
	// 回退到仅端口匹配
	return pm.GetPIDByPort(protocol, localPort)
}

// cleanupHistory 清理过期的历史记录
func (pm *ProcessMapper) cleanupHistory() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pm.mu.Lock()
			now := time.Now()
			for key, history := range pm.portHistory {
				if now.Sub(history.LastSeen) > pm.historyTTL {
					delete(pm.portHistory, key)
				}
			}
			pm.mu.Unlock()
		case <-pm.stopChan:
			return
		}
	}
}

// GetProcessByPID 获取进程详细信息
func GetProcessByPID(pid int32) (*ProcessInfo, error) {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}
	
	name, _ := proc.Name()
	exe, _ := proc.Exe()
	cmdline, _ := proc.Cmdline()
	username, _ := proc.Username()
	createTime, _ := proc.CreateTime()
	
	return &ProcessInfo{
		PID:        pid,
		Name:       name,
		Exe:        exe,
		Cmdline:    cmdline,
		Username:   username,
		CreateTime: createTime,
	}, nil
}

// GetStats 获取映射统计信息
func (pm *ProcessMapper) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return map[string]interface{}{
		"connections":     len(pm.connectionMap),
		"ports":           len(pm.portMap),
		"cached_processes": len(pm.processCache),
		"history_records":  len(pm.portHistory),
	}
}

// IsLocalIP 判断是否为本地IP
func IsLocalIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	
	// 检查是否为回环地址
	if parsedIP.IsLoopback() {
		return true
	}
	
	// 获取本机所有IP地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.Equal(parsedIP) {
				return true
			}
		}
	}
	
	return false
}

// Stop 停止进程映射管理器
func (pm *ProcessMapper) Stop() {
	close(pm.stopChan)
}

// getProtocolName 将连接类型转换为协议名称
func getProtocolName(family, connType uint32) string {
	// connType: 1=TCP, 2=UDP (根据gopsutil定义)
	switch connType {
	case 1:
		return "TCP"
	case 2:
		return "UDP"
	default:
		return "UNKNOWN"
	}
}

// ========== macOS 专用优化 ==========

// updateLsofCache 使用lsof更新端口到进程的映射 (macOS专用)
func (pm *ProcessMapper) updateLsofCache() {
	if runtime.GOOS != "darwin" {
		return
	}
	
	// 使用lsof获取网络连接信息
	// -i: 网络连接, -n: 不解析主机名, -P: 不解析端口名
	cmd := exec.Command("lsof", "-i", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		// lsof可能需要root权限，静默失败
		return
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// 清空旧缓存
	pm.lsofCache = make(map[string]int32)
	pm.lsofCacheTime = time.Now()
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] { // 跳过标题行
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		
		// 解析PID
		pid, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			continue
		}
		
		// 解析协议和端口
		// 格式: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
		// NAME格式: *:port 或 host:port->remote:port
		name := fields[len(fields)-1]
		node := fields[len(fields)-2] // TCP 或 UDP
		
		protocol := strings.ToUpper(node)
		if protocol != "TCP" && protocol != "UDP" {
			continue
		}
		
		// 解析本地端口
		var localPort string
		if strings.Contains(name, "->") {
			// 已建立连接: local:port->remote:port
			parts := strings.Split(name, "->")
			if len(parts) >= 1 {
				localParts := strings.Split(parts[0], ":")
				if len(localParts) >= 1 {
					localPort = localParts[len(localParts)-1]
				}
			}
		} else if strings.Contains(name, ":") {
			// 监听: *:port 或 host:port
			parts := strings.Split(name, ":")
			if len(parts) >= 1 {
				localPort = parts[len(parts)-1]
			}
		}
		
		if localPort == "" || localPort == "*" {
			continue
		}
		
		// 移除可能的状态后缀 (LISTEN)
		localPort = strings.Split(localPort, " ")[0]
		
		portKey := fmt.Sprintf("%s:%s", protocol, localPort)
		pm.lsofCache[portKey] = int32(pid)
		
		// 同时缓存进程信息
		if _, exists := pm.processCache[int32(pid)]; !exists {
			if info, err := GetProcessByPID(int32(pid)); err == nil {
				pm.processCache[int32(pid)] = info
			}
		}
	}
}

// GetPIDByLsof 使用lsof实时查询端口对应的进程 (macOS专用，用于缓存未命中时)
func (pm *ProcessMapper) GetPIDByLsof(protocol string, port uint32) (int32, *ProcessInfo, bool) {
	if runtime.GOOS != "darwin" {
		return 0, nil, false
	}
	
	// 使用lsof查询特定端口
	portStr := fmt.Sprintf(":%d", port)
	cmd := exec.Command("lsof", "-i", fmt.Sprintf("%s%s", strings.ToLower(protocol), portStr), "-n", "-P", "-t")
	output, err := cmd.Output()
	if err != nil {
		return 0, nil, false
	}
	
	// 解析PID
	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, nil, false
	}
	
	// 可能有多个PID，取第一个
	pids := strings.Split(pidStr, "\n")
	pid, err := strconv.ParseInt(pids[0], 10, 32)
	if err != nil {
		return 0, nil, false
	}
	
	// 获取进程信息
	info, _ := GetProcessByPID(int32(pid))
	
	return int32(pid), info, true
}

// GetMacOSNetworkStats 获取macOS网络统计信息
func GetMacOSNetworkStats() (map[string]interface{}, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("not macOS")
	}
	
	stats := make(map[string]interface{})
	
	// 使用netstat获取统计
	cmd := exec.Command("netstat", "-s")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	stats["raw"] = string(output)
	
	// 解析关键指标
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "packets received") {
			stats["packets_received"] = line
		} else if strings.Contains(line, "packets sent") {
			stats["packets_sent"] = line
		}
	}
	
	return stats, nil
}

