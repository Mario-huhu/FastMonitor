// +build darwin

package process

/*
#cgo LDFLAGS: -lproc
#include <libproc.h>
#include <sys/proc_info.h>
#include <stdlib.h>
#include <string.h>

// 获取进程名称
int get_proc_name(int pid, char *name, int name_size) {
    return proc_name(pid, name, name_size);
}

// 获取进程路径
int get_proc_path(int pid, char *path, int path_size) {
    return proc_pidpath(pid, path, path_size);
}

// 获取所有进程ID
int get_all_pids(int *pids, int max_pids) {
    return proc_listallpids(pids, max_pids * sizeof(int));
}
*/
import "C"

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// DarwinProcessMapper macOS专用进程映射器
// 使用 libproc 获取进程信息，使用 netstat 获取网络连接
type DarwinProcessMapper struct {
	mu sync.RWMutex

	// 连接映射 (五元组 -> PID)
	connectionMap map[ConnectionKey]int32

	// 端口映射 (协议:端口 -> PID)
	portMap map[string]int32

	// 进程信息缓存
	processCache map[int32]*ProcessInfo

	// 配置
	updateInterval time.Duration
	stopChan       chan struct{}
	lastUpdate     time.Time
}

// NewDarwinProcessMapper 创建macOS专用进程映射器
func NewDarwinProcessMapper() *DarwinProcessMapper {
	pm := &DarwinProcessMapper{
		connectionMap:  make(map[ConnectionKey]int32),
		portMap:        make(map[string]int32),
		processCache:   make(map[int32]*ProcessInfo),
		updateInterval: 2 * time.Second,
		stopChan:       make(chan struct{}),
	}

	// 立即更新一次
	pm.Update()

	// 启动自动更新
	go pm.autoUpdate()

	return pm
}

// autoUpdate 自动更新
func (pm *DarwinProcessMapper) autoUpdate() {
	ticker := time.NewTicker(pm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.Update()
		case <-pm.stopChan:
			return
		}
	}
}

// Update 更新进程映射表
// 使用 lsof 获取网络连接信息（比 netstat 更准确）
func (pm *DarwinProcessMapper) Update() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 清空旧映射
	pm.connectionMap = make(map[ConnectionKey]int32)
	pm.portMap = make(map[string]int32)

	// 使用 lsof 获取网络连接
	// -i: 网络连接, -n: 不解析主机名, -P: 不解析端口名, -F: 格式化输出
	cmd := exec.Command("lsof", "-i", "-n", "-P", "-F", "pcnPt")
	output, err := cmd.Output()
	if err != nil {
		// lsof 可能需要 root 权限，尝试使用 netstat 作为备用
		return pm.updateWithNetstat()
	}

	pm.parseLsofOutput(string(output))
	pm.lastUpdate = time.Now()
	return nil
}

// parseLsofOutput 解析 lsof -F 格式输出
func (pm *DarwinProcessMapper) parseLsofOutput(output string) {
	var currentPID int32
	var currentProtocol string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}

		prefix := line[0]
		value := line[1:]

		switch prefix {
		case 'p': // PID
			pid, err := strconv.ParseInt(value, 10, 32)
			if err == nil {
				currentPID = int32(pid)
				// 缓存进程信息
				if _, exists := pm.processCache[currentPID]; !exists {
					if info := pm.getProcessInfoNative(currentPID); info != nil {
						pm.processCache[currentPID] = info
					}
				}
			}
		case 'P': // Protocol (TCP/UDP)
			currentProtocol = strings.ToUpper(value)
		case 'n': // Name (address:port->remote:port)
			if currentPID > 0 && (currentProtocol == "TCP" || currentProtocol == "UDP") {
				pm.parseConnectionName(value, currentPID, currentProtocol)
			}
		}
	}
}

// parseConnectionName 解析连接名称
func (pm *DarwinProcessMapper) parseConnectionName(name string, pid int32, protocol string) {
	// 格式: local:port 或 local:port->remote:port
	if strings.Contains(name, "->") {
		// 已建立的连接
		parts := strings.Split(name, "->")
		if len(parts) == 2 {
			localIP, localPort := parseAddress(parts[0])
			remoteIP, remotePort := parseAddress(parts[1])

			if localPort > 0 {
				// 添加到连接映射
				key := ConnectionKey{
					Protocol:   protocol,
					LocalIP:    localIP,
					LocalPort:  uint32(localPort),
					RemoteIP:   remoteIP,
					RemotePort: uint32(remotePort),
				}
				pm.connectionMap[key] = pid

				// 添加到端口映射
				portKey := fmt.Sprintf("%s:%d", protocol, localPort)
				pm.portMap[portKey] = pid
			}
		}
	} else {
		// 监听端口
		localIP, localPort := parseAddress(name)
		if localPort > 0 {
			portKey := fmt.Sprintf("%s:%d", protocol, localPort)
			pm.portMap[portKey] = pid

			// 也添加到连接映射（用于监听端口）
			key := ConnectionKey{
				Protocol:  protocol,
				LocalIP:   localIP,
				LocalPort: uint32(localPort),
			}
			pm.connectionMap[key] = pid
		}
	}
}

// parseAddress 解析地址字符串
func parseAddress(addr string) (ip string, port int) {
	// 处理 IPv6 地址 [::1]:port
	if strings.HasPrefix(addr, "[") {
		idx := strings.LastIndex(addr, "]:")
		if idx > 0 {
			ip = addr[1:idx]
			port, _ = strconv.Atoi(addr[idx+2:])
			return
		}
	}

	// 处理 IPv4 地址 或 *:port
	idx := strings.LastIndex(addr, ":")
	if idx > 0 {
		ip = addr[:idx]
		if ip == "*" {
			ip = "0.0.0.0"
		}
		port, _ = strconv.Atoi(addr[idx+1:])
	}
	return
}

// updateWithNetstat 使用 netstat 作为备用方案
func (pm *DarwinProcessMapper) updateWithNetstat() error {
	// netstat -anv 显示所有连接和进程信息
	cmd := exec.Command("netstat", "-anv", "-p", "tcp")
	tcpOutput, _ := cmd.Output()

	cmd = exec.Command("netstat", "-anv", "-p", "udp")
	udpOutput, _ := cmd.Output()

	pm.parseNetstatOutput(string(tcpOutput), "TCP")
	pm.parseNetstatOutput(string(udpOutput), "UDP")

	pm.lastUpdate = time.Now()
	return nil
}

// parseNetstatOutput 解析 netstat 输出
func (pm *DarwinProcessMapper) parseNetstatOutput(output, protocol string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		// 跳过标题行
		if fields[0] == "Proto" || fields[0] == "Active" {
			continue
		}

		// 解析 PID（最后一列或倒数第二列）
		var pid int32
		for i := len(fields) - 1; i >= 0; i-- {
			if p, err := strconv.ParseInt(fields[i], 10, 32); err == nil && p > 0 {
				pid = int32(p)
				break
			}
		}

		if pid == 0 {
			continue
		}

		// 解析本地地址
		localAddr := fields[3]
		localIP, localPort := parseNetstatAddress(localAddr)

		if localPort > 0 {
			portKey := fmt.Sprintf("%s:%d", protocol, localPort)
			pm.portMap[portKey] = pid

			// 解析远程地址
			if len(fields) > 4 {
				remoteAddr := fields[4]
				remoteIP, remotePort := parseNetstatAddress(remoteAddr)

				if remotePort > 0 && remoteIP != "*" && remoteIP != "0.0.0.0" {
					key := ConnectionKey{
						Protocol:   protocol,
						LocalIP:    localIP,
						LocalPort:  uint32(localPort),
						RemoteIP:   remoteIP,
						RemotePort: uint32(remotePort),
					}
					pm.connectionMap[key] = pid
				}
			}

			// 缓存进程信息
			if _, exists := pm.processCache[pid]; !exists {
				if info := pm.getProcessInfoNative(pid); info != nil {
					pm.processCache[pid] = info
				}
			}
		}
	}
}

// parseNetstatAddress 解析 netstat 地址格式
func parseNetstatAddress(addr string) (ip string, port int) {
	// 格式: ip.port 或 *.port
	idx := strings.LastIndex(addr, ".")
	if idx > 0 {
		ip = addr[:idx]
		if ip == "*" {
			ip = "0.0.0.0"
		}
		port, _ = strconv.Atoi(addr[idx+1:])
	}
	return
}

// getProcessInfoNative 使用 libproc 获取进程信息
func (pm *DarwinProcessMapper) getProcessInfoNative(pid int32) *ProcessInfo {
	info := &ProcessInfo{PID: pid}

	// 获取进程名
	name := make([]C.char, 256)
	if C.get_proc_name(C.int(pid), &name[0], 256) > 0 {
		info.Name = C.GoString(&name[0])
	}

	// 获取进程路径
	path := make([]C.char, 1024)
	if C.get_proc_path(C.int(pid), &path[0], 1024) > 0 {
		info.Exe = C.GoString(&path[0])
	}

	return info
}

// GetPIDByConnection 通过五元组获取PID
func (pm *DarwinProcessMapper) GetPIDByConnection(protocol, srcIP, dstIP string, srcPort, dstPort uint32) (int32, *ProcessInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 正向匹配
	key := ConnectionKey{
		Protocol:   strings.ToUpper(protocol),
		LocalIP:    srcIP,
		LocalPort:  srcPort,
		RemoteIP:   dstIP,
		RemotePort: dstPort,
	}
	if pid, ok := pm.connectionMap[key]; ok {
		return pid, pm.processCache[pid], true
	}

	// 反向匹配
	reverseKey := ConnectionKey{
		Protocol:   strings.ToUpper(protocol),
		LocalIP:    dstIP,
		LocalPort:  dstPort,
		RemoteIP:   srcIP,
		RemotePort: srcPort,
	}
	if pid, ok := pm.connectionMap[reverseKey]; ok {
		return pid, pm.processCache[pid], true
	}

	return 0, nil, false
}

// GetPIDByPort 通过端口获取PID
func (pm *DarwinProcessMapper) GetPIDByPort(protocol string, port uint32) (int32, *ProcessInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	portKey := fmt.Sprintf("%s:%d", strings.ToUpper(protocol), port)
	if pid, ok := pm.portMap[portKey]; ok {
		return pid, pm.processCache[pid], true
	}

	return 0, nil, false
}

// GetStats 获取统计信息
func (pm *DarwinProcessMapper) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return map[string]interface{}{
		"connections":      len(pm.connectionMap),
		"ports":            len(pm.portMap),
		"cached_processes": len(pm.processCache),
		"last_update":      pm.lastUpdate,
		"method":           "libproc + lsof (macOS native)",
	}
}

// Stop 停止映射器
func (pm *DarwinProcessMapper) Stop() {
	close(pm.stopChan)
}

// getAllPIDs 获取所有进程ID
func (pm *DarwinProcessMapper) getAllPIDs() []int32 {
	// 分配足够大的缓冲区
	maxPids := 4096
	pids := make([]C.int, maxPids)

	numPids := C.get_all_pids((*C.int)(unsafe.Pointer(&pids[0])), C.int(maxPids))
	if numPids <= 0 {
		return nil
	}

	result := make([]int32, numPids)
	for i := 0; i < int(numPids); i++ {
		result[i] = int32(pids[i])
	}

	return result
}
