package process

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ProcessStats 进程流量统计
type ProcessStats struct {
	PID          int32     `json:"pid"`
	Name         string    `json:"name"`
	Exe          string    `json:"exe"`
	Username     string    `json:"username"`
	PacketsSent  int64     `json:"packets_sent"`
	PacketsRecv  int64     `json:"packets_recv"`
	BytesSent    int64     `json:"bytes_sent"`
	BytesRecv    int64     `json:"bytes_recv"`
	Connections  int       `json:"connections"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
}

// ProcessPacket 进程数据包记录（用于缓存最近的数据包）
type ProcessPacket struct {
	Timestamp time.Time `json:"timestamp"`
	SrcIP     string    `json:"src_ip"`
	DstIP     string    `json:"dst_ip"`
	SrcPort   uint16    `json:"src_port"`
	DstPort   uint16    `json:"dst_port"`
	Protocol  string    `json:"protocol"`
	Size      int       `json:"size"`
	IsSent    bool      `json:"is_sent"` // true=发送, false=接收
}

// PacketRingBuffer 数据包环形缓冲区
type PacketRingBuffer struct {
	packets []ProcessPacket
	size    int
	head    int
	count   int
}

// NewPacketRingBuffer 创建环形缓冲区
func NewPacketRingBuffer(size int) *PacketRingBuffer {
	return &PacketRingBuffer{
		packets: make([]ProcessPacket, size),
		size:    size,
	}
}

// Push 添加数据包
func (rb *PacketRingBuffer) Push(pkt ProcessPacket) {
	rb.packets[rb.head] = pkt
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll 获取所有缓存的数据包（按时间倒序）
func (rb *PacketRingBuffer) GetAll() []ProcessPacket {
	if rb.count == 0 {
		return nil
	}
	
	result := make([]ProcessPacket, rb.count)
	for i := 0; i < rb.count; i++ {
		// 从最新的开始取
		idx := (rb.head - 1 - i + rb.size) % rb.size
		result[i] = rb.packets[idx]
	}
	return result
}

// ProcessStatsManager 进程统计管理器（性能优化版，按可执行文件路径汇总）
type ProcessStatsManager struct {
	mu    sync.RWMutex
	db    *sql.DB
	stats map[string]*ProcessStats // 按可执行文件路径汇总，key=exe
	
	// 数据包缓存（每个进程缓存最近10个数据包）
	packetCache map[string]*PacketRingBuffer // key=exe
	cacheSize   int                          // 每个进程缓存的数据包数量
	
	// 性能优化配置
	flushInterval time.Duration // 批量写入间隔
	batchSize     int           // 批量写入大小
	stopChan      chan struct{}
	
	// macOS优化: 使用更大的内存缓冲
	maxMemoryStats int // 内存中最大统计条目数
}

// NewProcessStatsManager 创建进程统计管理器
func NewProcessStatsManager(db *sql.DB) *ProcessStatsManager {
	// macOS优化: 使用更长的刷新间隔和更大的缓冲
	flushInterval := 10 * time.Second
	maxMemoryStats := 500
	
	if runtime.GOOS == "darwin" {
		// macOS上使用更长的间隔减少磁盘IO
		flushInterval = 15 * time.Second
		maxMemoryStats = 1000
	}
	
	psm := &ProcessStatsManager{
		db:             db,
		stats:          make(map[string]*ProcessStats),
		packetCache:    make(map[string]*PacketRingBuffer),
		cacheSize:      10, // 每个进程缓存10个数据包
		flushInterval:  flushInterval,
		batchSize:      100,
		maxMemoryStats: maxMemoryStats,
		stopChan:       make(chan struct{}),
	}
	
	// 初始化数据库表
	if err := psm.initDB(); err != nil {
		fmt.Printf("[ProcessStats] DB init failed: %v\n", err)
	}
	
	// 启动自动刷新
	go psm.autoFlush()
	
	return psm
}

// initDB 初始化数据库表（迁移：删除旧表，重建新表）
func (psm *ProcessStatsManager) initDB() error {
	// 先删除旧表（因为主键从pid改为exe）
	_, err := psm.db.Exec(`DROP TABLE IF EXISTS process_stats`)
	if err != nil {
		return fmt.Errorf("drop old table: %w", err)
	}
	
	// 创建新表（exe作为主键）
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS process_stats (
		exe TEXT PRIMARY KEY,
		pid INTEGER,
		name TEXT NOT NULL,
		username TEXT,
		packets_sent INTEGER DEFAULT 0,
		packets_recv INTEGER DEFAULT 0,
		bytes_sent INTEGER DEFAULT 0,
		bytes_recv INTEGER DEFAULT 0,
		connections INTEGER DEFAULT 0,
		first_seen TIMESTAMP NOT NULL,
		last_seen TIMESTAMP NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_process_bytes_sent ON process_stats(bytes_sent DESC);
	CREATE INDEX IF NOT EXISTS idx_process_bytes_recv ON process_stats(bytes_recv DESC);
	CREATE INDEX IF NOT EXISTS idx_process_last_seen ON process_stats(last_seen DESC);
	`
	
	_, err = psm.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	
	fmt.Println("[ProcessStats] Table recreated with exe as primary key")
	return nil
}

// RecordPacket 记录数据包（高性能，仅更新内存）
func (psm *ProcessStatsManager) RecordPacket(pid int32, procInfo *ProcessInfo, isSent bool, packetSize int) {
	psm.RecordPacketWithDetails(pid, procInfo, isSent, packetSize, "", "", 0, 0, "")
}

// RecordPacketWithDetails 记录数据包（带详细信息，用于缓存）
func (psm *ProcessStatsManager) RecordPacketWithDetails(pid int32, procInfo *ProcessInfo, isSent bool, packetSize int, srcIP, dstIP string, srcPort, dstPort uint16, protocol string) {
	if pid == 0 || procInfo == nil {
		return
	}
	
	// 使用exe作为key进行汇总
	if procInfo.Exe == "" {
		return
	}
	
	psm.mu.Lock()
	defer psm.mu.Unlock()
	
	// 检查内存限制
	if len(psm.stats) >= psm.maxMemoryStats {
		// 触发异步刷新
		go psm.FlushToDB()
	}
	
	exeKey := procInfo.Exe
	stat, exists := psm.stats[exeKey]
	if !exists {
		stat = &ProcessStats{
			PID:       pid,  // 保存遇到的第一个PID
			Name:      procInfo.Name,
			Exe:       procInfo.Exe,
			Username:  procInfo.Username,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
		psm.stats[exeKey] = stat
	} else {
		// 更新PID为最近一次遇到的PID
		stat.PID = pid
	}
	
	// 更新统计
	if isSent {
		stat.PacketsSent++
		stat.BytesSent += int64(packetSize)
	} else {
		stat.PacketsRecv++
		stat.BytesRecv += int64(packetSize)
	}
	stat.LastSeen = time.Now()
	
	// 缓存数据包详情（如果提供了详细信息）
	if srcIP != "" || dstIP != "" {
		cache, exists := psm.packetCache[exeKey]
		if !exists {
			cache = NewPacketRingBuffer(psm.cacheSize)
			psm.packetCache[exeKey] = cache
		}
		
		cache.Push(ProcessPacket{
			Timestamp: time.Now(),
			SrcIP:     srcIP,
			DstIP:     dstIP,
			SrcPort:   srcPort,
			DstPort:   dstPort,
			Protocol:  protocol,
			Size:      packetSize,
			IsSent:    isSent,
		})
	}
}

// RecordConnection 记录连接数
func (psm *ProcessStatsManager) RecordConnection(pid int32, procInfo *ProcessInfo) {
	if pid == 0 || procInfo == nil || procInfo.Exe == "" {
		return
	}
	
	psm.mu.Lock()
	defer psm.mu.Unlock()
	
	exeKey := procInfo.Exe
	stat, exists := psm.stats[exeKey]
	if !exists {
		stat = &ProcessStats{
			PID:       pid,
			Name:      procInfo.Name,
			Exe:       procInfo.Exe,
			Username:  procInfo.Username,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
		psm.stats[exeKey] = stat
	} else {
		stat.PID = pid
	}
	
	stat.Connections++
	stat.LastSeen = time.Now()
}

// autoFlush 自动批量写入数据库
func (psm *ProcessStatsManager) autoFlush() {
	ticker := time.NewTicker(psm.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := psm.FlushToDB(); err != nil {
				fmt.Printf("[ProcessStats] Flush error: %v\n", err)
			}
		case <-psm.stopChan:
			// 最后一次刷新
			psm.FlushToDB()
			return
		}
	}
}

// FlushToDB 批量写入数据库（性能优化：使用事务）
func (psm *ProcessStatsManager) FlushToDB() error {
	psm.mu.RLock()
	if len(psm.stats) == 0 {
		psm.mu.RUnlock()
		return nil
	}
	
	// 复制数据，快速释放锁
	statsCopy := make(map[string]*ProcessStats, len(psm.stats))
	for exeKey, stat := range psm.stats {
		statsCopy[exeKey] = stat
	}
	psm.mu.RUnlock()
	
	// 批量写入（使用事务）
	tx, err := psm.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO process_stats 
		(exe, pid, name, username, packets_sent, packets_recv, bytes_sent, bytes_recv, connections, first_seen, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(exe) DO UPDATE SET
			pid = excluded.pid,
			packets_sent = packets_sent + excluded.packets_sent,
			packets_recv = packets_recv + excluded.packets_recv,
			bytes_sent = bytes_sent + excluded.bytes_sent,
			bytes_recv = bytes_recv + excluded.bytes_recv,
			connections = excluded.connections,
			last_seen = excluded.last_seen
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()
	
	count := 0
	for _, stat := range statsCopy {
		_, err := stmt.Exec(
			stat.Exe,
			stat.PID,
			stat.Name,
			stat.Username,
			stat.PacketsSent,
			stat.PacketsRecv,
			stat.BytesSent,
			stat.BytesRecv,
			stat.Connections,
			stat.FirstSeen.Unix(),
			stat.LastSeen.Unix(),
		)
		if err != nil {
			fmt.Printf("[ProcessStats] Insert error for Exe %s: %v\n", stat.Exe, err)
			continue
		}
		count++
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	
	fmt.Printf("[ProcessStats] Flushed %d process stats to DB\n", count)
	
	// 清空内存缓存（已写入DB）
	psm.mu.Lock()
	psm.stats = make(map[string]*ProcessStats)
	psm.mu.Unlock()
	
	return nil
}

// GetTopByTraffic 获取流量排名前N的进程（按exe汇总）
func (psm *ProcessStatsManager) GetTopByTraffic(limit int) ([]ProcessStats, error) {
	query := `
		SELECT exe, pid, name, username, 
		       packets_sent, packets_recv, 
		       bytes_sent, bytes_recv, 
		       connections, first_seen, last_seen
		FROM process_stats
		ORDER BY (bytes_sent + bytes_recv) DESC
		LIMIT ?
	`
	
	rows, err := psm.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []ProcessStats
	for rows.Next() {
		var stat ProcessStats
		var firstSeen, lastSeen int64
		
		err := rows.Scan(
			&stat.Exe,
			&stat.PID,
			&stat.Name,
			&stat.Username,
			&stat.PacketsSent,
			&stat.PacketsRecv,
			&stat.BytesSent,
			&stat.BytesRecv,
			&stat.Connections,
			&firstSeen,
			&lastSeen,
		)
		if err != nil {
			continue
		}
		
		stat.FirstSeen = time.Unix(firstSeen, 0)
		stat.LastSeen = time.Unix(lastSeen, 0)
		results = append(results, stat)
	}
	
	return results, nil
}

// GetAllStats 获取所有进程统计
func (psm *ProcessStatsManager) GetAllStats(offset, limit int) ([]ProcessStats, int, error) {
	// 获取总数
	var total int
	err := psm.db.QueryRow("SELECT COUNT(*) FROM process_stats").Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	// 获取分页数据
	query := `
		SELECT exe, pid, name, username, 
		       packets_sent, packets_recv, 
		       bytes_sent, bytes_recv, 
		       connections, first_seen, last_seen
		FROM process_stats
		ORDER BY last_seen DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := psm.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var results []ProcessStats
	for rows.Next() {
		var stat ProcessStats
		var firstSeen, lastSeen int64
		
		err := rows.Scan(
			&stat.Exe,
			&stat.PID,
			&stat.Name,
			&stat.Username,
			&stat.PacketsSent,
			&stat.PacketsRecv,
			&stat.BytesSent,
			&stat.BytesRecv,
			&stat.Connections,
			&firstSeen,
			&lastSeen,
		)
		if err != nil {
			continue
		}
		
		stat.FirstSeen = time.Unix(firstSeen, 0)
		stat.LastSeen = time.Unix(lastSeen, 0)
		results = append(results, stat)
	}
	
	return results, total, nil
}

// ClearAll 清空所有统计数据
func (psm *ProcessStatsManager) ClearAll() error {
	psm.mu.Lock()
	psm.stats = make(map[string]*ProcessStats)
	psm.packetCache = make(map[string]*PacketRingBuffer)
	psm.mu.Unlock()
	
	_, err := psm.db.Exec("DELETE FROM process_stats")
	return err
}

// GetProcessPackets 获取指定进程的缓存数据包
func (psm *ProcessStatsManager) GetProcessPackets(exe string) []ProcessPacket {
	psm.mu.RLock()
	defer psm.mu.RUnlock()
	
	if cache, exists := psm.packetCache[exe]; exists {
		return cache.GetAll()
	}
	return nil
}

// Stop 停止管理器
func (psm *ProcessStatsManager) Stop() {
	close(psm.stopChan)
}

