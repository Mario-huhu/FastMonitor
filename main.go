package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"fastmonitor/internal/capture"
	"fastmonitor/internal/config"
	"fastmonitor/internal/scheduler"
	"fastmonitor/internal/server"
	"fastmonitor/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// GeoIP数据从assets embed.FS中读取，避免重复嵌入

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config: %v, using defaults", err)
		cfg = config.Default()
	}

	// Create store
	st, err := store.NewComposite(cfg)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Get underlying SQLite store for dashboard
	sqliteStore := st.GetDB()

	// Create dashboard manager
	dashboard := server.NewDashboardManager(sqliteStore.GetRawDB())

	// Create capture
	cap := capture.New(cfg, st)

	// Create scheduler
	sched := scheduler.New(st, cfg)

	// Create app
	app := server.NewApp(cfg, cap, sched, st, dashboard)

	// Start scheduler in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sched.Run(ctx)

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
		if cap.IsRunning() {
			cap.Stop()
		}
		st.Close()
		os.Exit(0)
	}()

	// Create Wails application
	err = wails.Run(&options.App{
		Title:             "FastMonitor - 网络流量监控与威胁检测工具 v1.2.0",
		Width:             1400,
		Height:            900,
		MinWidth:          1200,
		MinHeight:         700,
		MaxWidth:          2560,
		MaxHeight:         1440,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
			Handler: nil, // 使用默认handler来提供embed的文件
		},
		Menu:             nil,
		Logger:           nil,
		LogLevel:         logger.DEBUG,
		OnStartup:        app.Startup,
		OnDomReady:       nil,
		OnBeforeClose:    nil,
		OnShutdown:       app.Shutdown,
		WindowStartState: options.Normal,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			WebviewBrowserPath:                "",
			Theme:                             windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            true,  // 启用全尺寸内容，沉浸式体验
				UseToolbar:                 true,  // 启用工具栏
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.DefaultAppearance, // 跟随系统外观设置
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "FastMonitor - 网络流量监控与威胁检测工具 v1.1.0",
				Message: "高性能跨平台网络流量监控与威胁检测系统\n\n基于 Wails 框架开发，集成数据包捕获、协议解析、进程关联、威胁情报、可视化分析等核心功能。",
				Icon:    icon,
			},
		},
		Linux: &linux.Options{
			Icon: icon,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// extractGeoIPDatabases 函数已废弃 - GeoIP数据库现在直接从embed读取，无需提取到磁盘
