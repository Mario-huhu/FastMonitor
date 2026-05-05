package process

import (
	"fmt"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessMonitor 进程监控器 - 监控系统所有运行的进程
type ProcessMonitor struct {
	mu              sync.RWMutex
	runningProcesses map[int32]*ProcessInfo // PID -> ProcessInfo
	onProcessStart  func(*ProcessInfo)      // 进程启动回调
	onProcessStop   func(int32)             // 进程停止回调
	stopChan        chan struct{}
	scanInterval    time.Duration
}

// NewProcessMonitor 创建进程监控器
func NewProcessMonitor(scanInterval time.Duration) *ProcessMonitor {
	return &ProcessMonitor{
		runningProcesses: make(map[int32]*ProcessInfo),
		stopChan:         make(chan struct{}),
		scanInterval:     scanInterval,
	}
}

// SetCallbacks 设置回调函数
func (pm *ProcessMonitor) SetCallbacks(onStart func(*ProcessInfo), onStop func(int32)) {
	pm.onProcessStart = onStart
	pm.onProcessStop = onStop
}

// Start 启动进程监控
func (pm *ProcessMonitor) Start() {
	// 初始扫描
	pm.scan()
	
	// 定期扫描
	go pm.monitorLoop()
}

// Stop 停止进程监控
func (pm *ProcessMonitor) Stop() {
	close(pm.stopChan)
}

// monitorLoop 监控循环
func (pm *ProcessMonitor) monitorLoop() {
	ticker := time.NewTicker(pm.scanInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pm.scan()
		case <-pm.stopChan:
			return
		}
	}
}

// scan 扫描当前运行的所有进程
func (pm *ProcessMonitor) scan() {
	// 获取所有进程
	pids, err := process.Pids()
	if err != nil {
		fmt.Printf("[ProcessMonitor] Failed to get PIDs: %v\n", err)
		return
	}
	
	currentPids := make(map[int32]bool)
	
	// 检查新进程
	for _, pid := range pids {
		currentPids[pid] = true
		
		pm.mu.RLock()
		_, exists := pm.runningProcesses[pid]
		pm.mu.RUnlock()
		
		if !exists {
			// 发现新进程
			if info, err := GetProcessByPID(pid); err == nil {
				pm.mu.Lock()
				pm.runningProcesses[pid] = info
				pm.mu.Unlock()
				
				fmt.Printf("[ProcessMonitor] 新进程: PID=%d, Name=%s, Exe=%s\n", pid, info.Name, info.Exe)
				
				// 触发回调
				if pm.onProcessStart != nil {
					pm.onProcessStart(info)
				}
			}
		}
	}
	
	// 检查已停止的进程
	pm.mu.Lock()
	for pid := range pm.runningProcesses {
		if !currentPids[pid] {
			fmt.Printf("[ProcessMonitor] 进程停止: PID=%d\n", pid)
			
			// 触发回调
			if pm.onProcessStop != nil {
				pm.onProcessStop(pid)
			}
			
			delete(pm.runningProcesses, pid)
		}
	}
	pm.mu.Unlock()
}

// GetRunningProcesses 获取当前运行的所有进程
func (pm *ProcessMonitor) GetRunningProcesses() []*ProcessInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	result := make([]*ProcessInfo, 0, len(pm.runningProcesses))
	for _, info := range pm.runningProcesses {
		result = append(result, info)
	}
	
	return result
}

// GetProcessByPID 获取指定PID的进程信息
func (pm *ProcessMonitor) GetProcessByPID(pid int32) (*ProcessInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	info, exists := pm.runningProcesses[pid]
	return info, exists
}

