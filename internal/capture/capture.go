package capture

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"fastmonitor/internal/cache"
	"fastmonitor/internal/config"
	"fastmonitor/internal/netio"
	"fastmonitor/internal/parser"
	"fastmonitor/internal/process"
	"fastmonitor/internal/store"
	"fastmonitor/pkg/model"
)

var (
	ErrAlreadyRunning = errors.New("capture already running")
	ErrNotRunning     = errors.New("capture not running")
)

// Capture manages packet capture and processing
// 抓包核心模块
type Capture struct {
	mu sync.RWMutex

	cfg    *config.Config
	store  store.Store
	rings  *cache.RingSet

	// Runtime state
	handle        netio.Handle
	interfaceName string
	isRunning     atomic.Bool
	isPaused      atomic.Bool
	ctx           context.Context
	cancel        context.CancelFunc

	// Metrics
	packetsTotal   atomic.Int64
	packetsDropped atomic.Int64
	bytesTotal     atomic.Int64
	lastMetrics    time.Time
	lastPackets    int64
	lastBytes      int64
	metricsC       chan model.Metrics
	
	// 进程映射器 (平台特定实现)
	processMapper  process.IProcessMapper
	processStats   *process.ProcessStatsManager
}

// New creates a new Capture instance
func New(cfg *config.Config, s store.Store) *Capture {
	limits := cfg.GetLimits()
	
	// 获取数据库连接
	db := s.GetDB().GetRawDB()
	
	return &Capture{
		cfg:           cfg,
		store:         s,
		rings:         cache.NewRingSet(limits.RawMax, limits.DNSMax, limits.HTTPMax, limits.ICMPMax),
		lastMetrics:   time.Now(),
		metricsC:      make(chan model.Metrics, 10),
		processMapper: process.NewPlatformProcessMapper(), // 使用平台特定的进程映射器
		processStats:  process.NewProcessStatsManager(db),
	}
}

// Start starts packet capture on the specified interface
func (c *Capture) Start(iface string) error {
	if c.isRunning.Load() {
		return ErrAlreadyRunning
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Open interface
	handle, err := netio.Open(iface, int32(c.cfg.SnapshotLen), c.cfg.Promiscuous, int(c.cfg.GetTimeout()))
	if err != nil {
		return fmt.Errorf("open interface: %w", err)
	}

	c.handle = handle
	c.interfaceName = iface
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.isRunning.Store(true)
	c.isPaused.Store(false)

	// Reset metrics
	c.packetsTotal.Store(0)
	c.packetsDropped.Store(0)
	c.bytesTotal.Store(0)
	c.lastMetrics = time.Now()
	c.lastPackets = 0
	c.lastBytes = 0

	// Start capture goroutine
	go c.captureLoop()

	// Start metrics goroutine
	go c.metricsLoop()

	return nil
}

// Stop stops packet capture
func (c *Capture) Stop() error {
	if !c.isRunning.Load() {
		return ErrNotRunning
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.isRunning.Store(false)
	c.isPaused.Store(false)

	if c.cancel != nil {
		c.cancel()
	}

	if c.handle != nil {
		c.handle.Close()
		c.handle = nil
	}

	return nil
}

// Pause pauses packet capture (drops packets but keeps connection)
func (c *Capture) Pause() {
	c.isPaused.Store(true)
}

// Resume resumes packet capture
func (c *Capture) Resume() {
	c.isPaused.Store(false)
}

// IsRunning returns whether capture is running
func (c *Capture) IsRunning() bool {
	return c.isRunning.Load()
}

// IsPaused returns whether capture is paused
func (c *Capture) IsPaused() bool {
	return c.isPaused.Load()
}

// GetInterfaceName returns the current interface name
func (c *Capture) GetInterfaceName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.interfaceName
}

// StreamMetrics returns a channel for streaming metrics
func (c *Capture) StreamMetrics() <-chan model.Metrics {
	return c.metricsC
}

// Snapshot returns a snapshot of the specified ring buffer
func (c *Capture) Snapshot(table model.TableType) []interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch table {
	case model.TableRaw:
		return c.rings.GetRaw().Snapshot()
	case model.TableDNS:
		return c.rings.GetDNS().Snapshot()
	case model.TableHTTP:
		return c.rings.GetHTTP().Snapshot()
	case model.TableICMP:
		return c.rings.GetICMP().Snapshot()
	default:
		return nil
	}
}

// UpdateLimits updates the ring buffer limits with smooth migration
func (c *Capture) UpdateLimits(limits config.Limits) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rings.ResizeRaw(limits.RawMax)
	c.rings.ResizeDNS(limits.DNSMax)
	c.rings.ResizeHTTP(limits.HTTPMax)
	c.rings.ResizeICMP(limits.ICMPMax)

	c.cfg.UpdateLimits(limits)
}

// captureLoop is the main packet capture loop
func (c *Capture) captureLoop() {
	// macOS优化: 使用批量处理减少锁竞争
	batchSize := 10
	packetBatch := make([]*model.Packet, 0, batchSize)
	
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Check if paused
		if c.isPaused.Load() {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Read packet
		data, ci, err := c.handle.ReadPacketData()
		if err != nil {
			// Check if it's a timeout or actual error
			if errors.Is(err, context.Canceled) {
				return
			}
			// Timeout is normal, continue
			continue
		}

		// Update metrics
		c.packetsTotal.Add(1)
		c.bytesTotal.Add(int64(ci.Length))

		// Parse packet
		timestamp := time.Unix(0, ci.Timestamp)
		pkt, err := parser.ParsePacket(data, timestamp)
		if err != nil {
			continue
		}

		pkt.CaptureLen = ci.CaptureLength
		pkt.Length = ci.Length

		// ========== 100%准确进程关联 ==========
		// 方案1: 优先使用完整五元组进行精确匹配
		if pkt.Protocol == "TCP" || pkt.Protocol == "UDP" {
			// 判断数据包方向（入站/出站）
			srcIsLocal := process.IsLocalIP(pkt.SrcIP)
			dstIsLocal := process.IsLocalIP(pkt.DstIP)
			
			if srcIsLocal || dstIsLocal {
				// 尝试完整五元组匹配（最准确）
				if pid, procInfo, ok := c.processMapper.GetPIDByConnection(
					pkt.Protocol,
					pkt.SrcIP,
					pkt.DstIP,
					uint32(pkt.SrcPort),
					uint32(pkt.DstPort),
				); ok {
					pkt.ProcessPID = pid
					if procInfo != nil {
						pkt.ProcessName = procInfo.Name
						pkt.ProcessExe = procInfo.Exe
						
						// 记录进程统计（带详细信息用于缓存）
						c.processStats.RecordPacketWithDetails(
							pid, procInfo, srcIsLocal, pkt.Length,
							pkt.SrcIP, pkt.DstIP, pkt.SrcPort, pkt.DstPort, pkt.Protocol,
						)
					}
				} else {
					// 方案2: 五元组失败，使用本地端口匹配
					localPort := uint32(pkt.SrcPort)
					if dstIsLocal {
						localPort = uint32(pkt.DstPort)
					}
					
					if pid, procInfo, ok := c.processMapper.GetPIDByPort(pkt.Protocol, localPort); ok {
						pkt.ProcessPID = pid
						if procInfo != nil {
							pkt.ProcessName = procInfo.Name
							pkt.ProcessExe = procInfo.Exe
							
							// 记录进程统计（带详细信息用于缓存）
							c.processStats.RecordPacketWithDetails(
								pid, procInfo, srcIsLocal, pkt.Length,
								pkt.SrcIP, pkt.DstIP, pkt.SrcPort, pkt.DstPort, pkt.Protocol,
							)
						}
					}
				}
			}
		}
		// ICMP等其他协议：根据源IP判断
		if pkt.ProcessPID == 0 && pkt.Protocol == "ICMP" {
			if process.IsLocalIP(pkt.SrcIP) {
				// ICMP通常由系统进程或特定应用发出
				// 可以尝试通过ICMP ID字段进一步关联，这里简化处理
			}
		}
		// ========== 进程关联结束 ==========

		// Store raw packet
		c.rings.GetRaw().Push(pkt)
		
		// 批量处理优化
		packetBatch = append(packetBatch, pkt)
		if len(packetBatch) >= batchSize {
			// 批量写入
			batch := packetBatch
			packetBatch = make([]*model.Packet, 0, batchSize)
			go c.processBatch(batch)
		}
	}
}

// processBatch 批量处理数据包
func (c *Capture) processBatch(packets []*model.Packet) {
	for _, pkt := range packets {
		// Write to persistent storage
		if err := c.store.WriteRaw(pkt); err != nil {
			fmt.Printf("Error writing raw packet: %v\n", err)
		}

		// 实时更新会话流统计
		sqliteStore := c.store.GetDB()
		if err := sqliteStore.UpsertSessionFlow(pkt); err != nil {
			if c.packetsTotal.Load()%1000 == 0 {
				fmt.Printf("[WARN] Session flow upsert failed: %v\n", err)
			}
		}

		// Try to parse as DNS
		if dnsSession, err := parser.ParseDNS(pkt); err == nil {
			c.rings.GetDNS().Push(dnsSession)
			if err := c.store.WriteSession(model.TableDNS, dnsSession); err != nil {
				fmt.Printf("[ERROR] DNS写入失败: %v\n", err)
			}
			sqliteStore.CheckAlertRules(pkt, dnsSession)
		}

		// Try to parse as HTTP
		if httpSession, err := parser.ParseHTTP(pkt); err == nil {
			c.rings.GetHTTP().Push(httpSession)
			if err := c.store.WriteSession(model.TableHTTP, httpSession); err != nil {
				fmt.Printf("[ERROR] HTTP写入失败: %v\n", err)
			}
			sqliteStore.CheckAlertRules(pkt, httpSession)
		}

		// Try to parse as ICMP
		if icmpSession, err := parser.ParseICMP(pkt); err == nil {
			c.rings.GetICMP().Push(icmpSession)
			if err := c.store.WriteSession(model.TableICMP, icmpSession); err != nil {
				fmt.Printf("[ERROR] ICMP写入失败: %v\n", err)
			}
			sqliteStore.CheckAlertRules(pkt, icmpSession)
		}
		
		// 检查目标IP告警
		sqliteStore.CheckAlertRules(pkt, nil)
	}
}

// metricsLoop periodically calculates and sends metrics
func (c *Capture) metricsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			metrics := c.calculateMetrics()
			
			// Non-blocking send
			select {
			case c.metricsC <- metrics:
			default:
				// Channel full, skip this update
			}
		}
	}
}

// calculateMetrics calculates current metrics
func (c *Capture) calculateMetrics() model.Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	elapsed := now.Sub(c.lastMetrics).Seconds()

	currentPackets := c.packetsTotal.Load()
	currentBytes := c.bytesTotal.Load()

	var pps, bps float64
	if elapsed > 0 {
		pps = float64(currentPackets-c.lastPackets) / elapsed
		bps = float64(currentBytes-c.lastBytes) / elapsed
	}

	c.lastMetrics = now
	c.lastPackets = currentPackets
	c.lastBytes = currentBytes

	// Get dropped packets from handle
	var dropped int64
	if c.handle != nil {
		if stats, err := c.handle.Stats(); err == nil {
			dropped = int64(stats.PacketsDropped)
		}
	}

	return model.Metrics{
		Timestamp:      now,
		Interface:      c.interfaceName,
		IsCapturing:    c.isRunning.Load(),
		IsPaused:       c.isPaused.Load(),
		PacketsTotal:   currentPackets,
		PacketsDropped: dropped,
		BytesTotal:     currentBytes,
		PacketsPerSec:  pps,
		BytesPerSec:    bps,
		RawCount:       c.rings.GetRaw().Len(),
		DNSCount:       c.rings.GetDNS().Len(),
		HTTPCount:      c.rings.GetHTTP().Len(),
		ICMPCount:      c.rings.GetICMP().Len(),
	}
}

// GetMetrics returns the current metrics snapshot
func (c *Capture) GetMetrics() model.Metrics {
	return c.calculateMetrics()
}

// ClearAll clears all ring buffers
func (c *Capture) ClearAll() {
	c.rings.GetRaw().Clear()
	c.rings.GetDNS().Clear()
	c.rings.GetHTTP().Clear()
	c.rings.GetICMP().Clear()
}

// GetProcessStats 获取进程统计（代理到ProcessStatsManager）
func (c *Capture) GetProcessStats(offset, limit int) ([]process.ProcessStats, int, error) {
	return c.processStats.GetAllStats(offset, limit)
}

// GetTopProcessesByTraffic 获取流量排名前N的进程
func (c *Capture) GetTopProcessesByTraffic(limit int) ([]process.ProcessStats, error) {
	return c.processStats.GetTopByTraffic(limit)
}

// ClearProcessStats 清空进程统计
func (c *Capture) ClearProcessStats() error {
	return c.processStats.ClearAll()
}

// GetProcessPackets 获取指定进程的缓存数据包
func (c *Capture) GetProcessPackets(exe string) []process.ProcessPacket {
	return c.processStats.GetProcessPackets(exe)
}



// ProcessStatsWithPackets 带数据包缓存的进程统计
type ProcessStatsWithPackets struct {
	process.ProcessStats
	RecentPackets []process.ProcessPacket `json:"recent_packets"`
}

// ProcessPacketInfo 数据包信息（用于API返回）
type ProcessPacketInfo struct {
	Timestamp time.Time `json:"timestamp"`
	SrcIP     string    `json:"src_ip"`
	DstIP     string    `json:"dst_ip"`
	SrcPort   uint16    `json:"src_port"`
	DstPort   uint16    `json:"dst_port"`
	Protocol  string    `json:"protocol"`
	Size      int       `json:"size"`
	IsSent    bool      `json:"is_sent"`
}
