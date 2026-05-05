package server

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fastmonitor/internal/capture"
	"fastmonitor/internal/config"
	"fastmonitor/internal/geoip"
	"fastmonitor/internal/netio"
	"fastmonitor/internal/scheduler"
	"fastmonitor/internal/store"
	"fastmonitor/pkg/model"
)

// App is the Wails application structure
type App struct {
	ctx          context.Context
	cfg          *config.Config
	capture      *capture.Capture
	scheduler    *scheduler.Scheduler
	store        store.Store
	dashboard    *DashboardManager
	geoipService *geoip.GeoIPService // GeoIP服务（单例）
}

// NewApp creates a new App application
func NewApp(cfg *config.Config, cap *capture.Capture, sched *scheduler.Scheduler, s store.Store, dashboard *DashboardManager) *App {
	return &App{
		cfg:       cfg,
		capture:   cap,
		scheduler: sched,
		store:     s,
		dashboard: dashboard,
	}
}

// startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("Sniffer application started")
	
	// 初始化 GeoIP 服务（单例）
	a.initGeoIPService()
}

// initGeoIPService 初始化GeoIP服务
func (a *App) initGeoIPService() {
	// 尝试多个可能的路径
	geoipDirs := []string{
		"data/geoip",
		"./data/geoip",
	}
	
	// 添加可执行文件相对路径
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		geoipDirs = append(geoipDirs,
			filepath.Join(exeDir, "data", "geoip"),
			filepath.Join(exeDir, "..", "data", "geoip"),
		)
	}
	
	var err error
	for _, geoipDir := range geoipDirs {
		a.geoipService, err = geoip.NewGeoIPService(geoipDir)
		if err == nil {
			fmt.Printf("[App] ✓ GeoIP服务初始化成功\n")
			return
		}
	}
	
	fmt.Printf("[App] ⚠️  GeoIP服务初始化失败: %v\n", err)
	fmt.Println("[App] 地图功能将不可用")
}

// shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	fmt.Println("Shutting down...")
	
	// Stop capture if running
	if a.capture.IsRunning() {
		a.capture.Stop()
	}
	
	// Close GeoIP service
	if a.geoipService != nil {
		a.geoipService.Close()
		fmt.Println("GeoIP service closed")
	}

	// Close store
	if a.store != nil {
		a.store.Close()
	}
}

// GetInterfaces returns all available network interfaces
func (a *App) GetInterfaces() ([]model.NetworkInterface, error) {
	return netio.List()
}

// CheckPermission checks if the app has permission to capture packets
func (a *App) CheckPermission() error {
	return netio.CheckPermission()
}

// GetLibraryVersion returns the pcap library version
func (a *App) GetLibraryVersion() string {
	return netio.GetVersion()
}

// GetNpcapDownloadURL returns the Npcap download URL (Windows only)
func (a *App) GetNpcapDownloadURL() string {
	return netio.GetNpcapDownloadURL()
}

// StartCapture starts packet capture on the specified interface
func (a *App) StartCapture(iface string) error {
	return a.capture.Start(iface)
}

// StopCapture stops packet capture
func (a *App) StopCapture() error {
	return a.capture.Stop()
}

// PauseCapture pauses packet capture
func (a *App) PauseCapture() {
	a.capture.Pause()
}

// ResumeCapture resumes packet capture
func (a *App) ResumeCapture() {
	a.capture.Resume()
}

// ClearAllData clears all captured data (memory + database + process stats)
func (a *App) ClearAllData() error {
	// Stop capture if running (ignore error if not running)
	_ = a.capture.Stop()
	
	// Clear memory rings
	a.capture.ClearAll()
	
	// Clear database
	if err := a.store.ClearAll(); err != nil {
		return fmt.Errorf("clear database: %w", err)
	}
	
	// Clear process stats
	if err := a.capture.ClearProcessStats(); err != nil {
		return fmt.Errorf("clear process stats: %w", err)
	}
	
	return nil
}

// GetMetrics returns current capture metrics
func (a *App) GetMetrics() model.Metrics {
	return a.capture.GetMetrics()
}

// GetSnapshot returns a snapshot of the specified data table
func (a *App) GetSnapshot(table string, limit int) ([]interface{}, error) {
	tableType := model.TableType(table)
	
	// First get from memory ring buffer
	snapshot := a.capture.Snapshot(tableType)
	
	if len(snapshot) >= limit {
		// Return from memory
		if len(snapshot) > limit {
			return snapshot[:limit], nil
		}
		return snapshot, nil
	}

	// If memory doesn't have enough, load from database (for session tables)
	if tableType != model.TableRaw {
		sessions, err := a.store.LoadSnapshot(tableType, limit)
		if err != nil {
			return nil, fmt.Errorf("load from database: %w", err)
		}

		// Convert to []interface{}
		result := make([]interface{}, len(sessions))
		for i, s := range sessions {
			result[i] = s
		}
		return result, nil
	}

	return snapshot, nil
}

// GetRawPackets returns raw packets from memory
func (a *App) GetRawPackets(limit int) ([]*model.Packet, error) {
	snapshot := a.capture.Snapshot(model.TableRaw)
	
	packets := make([]*model.Packet, 0, len(snapshot))
	for _, item := range snapshot {
		if pkt, ok := item.(*model.Packet); ok {
			packets = append(packets, pkt)
			if len(packets) >= limit {
				break
			}
		}
	}

	return packets, nil
}

// GetSessions returns sessions from memory or database
func (a *App) GetSessions(table string, limit int) ([]*model.Session, error) {
	tableType := model.TableType(table)
	if tableType == model.TableRaw {
		return nil, fmt.Errorf("use GetRawPackets for raw data")
	}

	snapshot := a.capture.Snapshot(tableType)
	
	sessions := make([]*model.Session, 0, len(snapshot))
	for _, item := range snapshot {
		if sess, ok := item.(*model.Session); ok {
			sessions = append(sessions, sess)
			if len(sessions) >= limit {
				break
			}
		}
	}

	// If not enough in memory, load from database
	if len(sessions) < limit {
		dbSessions, err := a.store.LoadSnapshot(tableType, limit)
		if err == nil {
			sessions = append(sessions, dbSessions...)
		}
	}

	return sessions, nil
}

// UpdateLimits updates the ring buffer limits
func (a *App) UpdateLimits(limits config.Limits) error {
	a.capture.UpdateLimits(limits)
	return nil
}

// GetLimits returns the current ring buffer limits
func (a *App) GetLimits() config.Limits {
	return a.cfg.GetLimits()
}

// GetConfig returns the current configuration
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// UpdateConfig updates and saves the configuration
func (a *App) UpdateConfig(newCfg *config.Config) error {
	// Update limits if changed
	if newCfg.RawMax != a.cfg.RawMax || 
	   newCfg.DNSMax != a.cfg.DNSMax ||
	   newCfg.HTTPMax != a.cfg.HTTPMax ||
	   newCfg.ICMPMax != a.cfg.ICMPMax {
		a.capture.UpdateLimits(config.Limits{
			RawMax:  newCfg.RawMax,
			DNSMax:  newCfg.DNSMax,
			HTTPMax: newCfg.HTTPMax,
			ICMPMax: newCfg.ICMPMax,
		})
	}

	// Update config
	*a.cfg = *newCfg

	// Save to file
	return a.cfg.Save("config.yaml")
}

// ExportPCAP exports packets in the time range to PCAP format
func (a *App) ExportPCAP(startTime, endTime int64) ([]byte, error) {
	start := time.Unix(startTime, 0)
	end := time.Unix(endTime, 0)

	var buf bytes.Buffer
	if err := a.store.ExportPCAP(start, end, &buf); err != nil {
		return nil, fmt.Errorf("export pcap: %w", err)
	}

	return buf.Bytes(), nil
}

// GetStorageStats returns storage statistics
func (a *App) GetStorageStats() (store.StoreStats, error) {
	return a.store.Stats()
}

// VacuumStorage manually triggers storage cleanup
func (a *App) VacuumStorage() error {
	before := time.Now().AddDate(0, 0, -a.cfg.DBVacuumDay)
	return a.store.Vacuum(before)
}

// IsCapturing returns whether capture is currently running
func (a *App) IsCapturing() bool {
	return a.capture.IsRunning()
}

// IsPaused returns whether capture is paused
func (a *App) IsPaused() bool {
	return a.capture.IsPaused()
}

// GetCurrentInterface returns the current capture interface name
func (a *App) GetCurrentInterface() string {
	return a.capture.GetInterfaceName()
}

// GetDashboardStats returns dashboard statistics
func (a *App) GetDashboardStats() (*model.DashboardStats, error) {
	if a.dashboard == nil {
		return nil, fmt.Errorf("dashboard not initialized")
	}
	
	// 更新实时流量数据点
	metrics := a.capture.GetMetrics()
	a.dashboard.UpdateTrafficPoint(
		metrics.PacketsTotal,
		metrics.BytesTotal,
		metrics.PacketsPerSec,
		metrics.BytesPerSec,
	)
	
	return a.dashboard.GetDashboardStats()
}

// GetProtocolDistribution returns protocol distribution statistics
func (a *App) GetProtocolDistribution() (map[string]int64, error) {
	if a.dashboard == nil {
		return nil, fmt.Errorf("dashboard not initialized")
	}
	
	stats, err := a.dashboard.GetProtocolDistribution()
	if err != nil {
		return nil, err
	}
	
	return map[string]int64{
		"TCP":   stats.TCP,
		"UDP":   stats.UDP,
		"ICMP":  stats.ICMP,
		"Other": stats.Other,
	}, nil
}

// QuerySessions 查询会话（支持分页、排序、搜索）
func (a *App) QuerySessions(opts model.QueryOptions) (*model.QueryResult, error) {
	composite, ok := a.store.(*store.CompositeStore)
	if !ok {
		return nil, fmt.Errorf("store is not composite")
	}
	
	sqliteStore := composite.GetDB()
	return sqliteStore.QuerySessions(opts)
}

// QuerySessionFlows 查询会话流统计
func (a *App) QuerySessionFlows(opts model.SessionFlowQuery) (*model.SessionFlowResult, error) {
	composite, ok := a.store.(*store.CompositeStore)
	if !ok {
		return nil, fmt.Errorf("store is not composite")
	}
	
	sqliteStore := composite.GetDB()
	return sqliteStore.QuerySessionFlows(opts)
}


// ========== 进程统计相关 API ==========

// ProcessStatsResult 进程统计结果
type ProcessStatsResult struct {
	Data  []capture.ProcessStatsWithPackets `json:"data"`
	Total int                               `json:"total"`
}

// GetProcessStats 获取进程统计（分页）
func (a *App) GetProcessStats(page, pageSize int) (*ProcessStatsResult, error) {
	offset := (page - 1) * pageSize
	stats, total, err := a.capture.GetProcessStats(offset, pageSize)
	if err != nil {
		return nil, err
	}
	
	// 转换为带数据包缓存的结果
	data := make([]capture.ProcessStatsWithPackets, len(stats))
	for i, stat := range stats {
		data[i] = capture.ProcessStatsWithPackets{
			ProcessStats: stat,
			RecentPackets: a.capture.GetProcessPackets(stat.Exe),
		}
	}
	
	return &ProcessStatsResult{
		Data:  data,
		Total: total,
	}, nil
}

// GetTopProcessesByTraffic 获取流量排名前N的进程
func (a *App) GetTopProcessesByTraffic(limit int) ([]capture.ProcessStatsWithPackets, error) {
	stats, err := a.capture.GetTopProcessesByTraffic(limit)
	if err != nil {
		return nil, err
	}
	
	// 转换为带数据包缓存的结果
	result := make([]capture.ProcessStatsWithPackets, len(stats))
	for i, stat := range stats {
		result[i] = capture.ProcessStatsWithPackets{
			ProcessStats: stat,
			RecentPackets: a.capture.GetProcessPackets(stat.Exe),
		}
	}
	
	return result, nil
}

// ClearProcessStats 清空进程统计
func (a *App) ClearProcessStats() error {
	return a.capture.ClearProcessStats()
}

// GetProcessPackets 获取指定进程的缓存数据包
func (a *App) GetProcessPackets(exe string) []capture.ProcessPacketInfo {
	packets := a.capture.GetProcessPackets(exe)
	if packets == nil {
		return nil
	}
	
	result := make([]capture.ProcessPacketInfo, len(packets))
	for i, pkt := range packets {
		result[i] = capture.ProcessPacketInfo{
			Timestamp: pkt.Timestamp,
			SrcIP:     pkt.SrcIP,
			DstIP:     pkt.DstIP,
			SrcPort:   pkt.SrcPort,
			DstPort:   pkt.DstPort,
			Protocol:  pkt.Protocol,
			Size:      pkt.Size,
			IsSent:    pkt.IsSent,
		}
	}
	return result
}
