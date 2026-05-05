package geoip

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPService GeoIP解析服务
type GeoIPService struct {
	cityDB    *geoip2.Reader
	countryDB *geoip2.Reader
	mu        sync.RWMutex
}

// Location 地理位置信息
type Location struct {
	IP        string  `json:"ip"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// NewGeoIPService 创建GeoIP服务
func NewGeoIPService(geoipDir string) (*GeoIPService, error) {
	service := &GeoIPService{}

	// 构建可能的路径列表（优先级从高到低）
	possibleDirs := []string{}
	
	// 1. 首先尝试可执行文件同级的 data/geoip 目录（最高优先级）
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		
		// macOS .app 包的特殊处理
		// 如果在 .app/Contents/MacOS/ 中，向上查找到 .app 同级目录
		if strings.Contains(exeDir, ".app/Contents/MacOS") {
			// 从 /path/to/app.app/Contents/MacOS 向上到 /path/to/
			appDir := filepath.Dir(filepath.Dir(filepath.Dir(exeDir)))
			possibleDirs = append(possibleDirs,
				filepath.Join(appDir, "data", "geoip"),     // /path/to/data/geoip
			)
		}
		
		// 通用路径（适用于所有平台）
		possibleDirs = append(possibleDirs,
			filepath.Join(exeDir, "data", "geoip"),         // 可执行文件同级
			filepath.Join(exeDir, "..", "data", "geoip"),   // 上一级
		)
	}
	
	// 2. 当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		possibleDirs = append(possibleDirs,
			filepath.Join(cwd, "data", "geoip"),
			filepath.Join(cwd, geoipDir),
		)
	}
	
	// 3. 相对路径
	possibleDirs = append(possibleDirs,
		"data/geoip",
		"./data/geoip",
		geoipDir,
	)
	
	// 4. macOS .app 包内部路径（最低优先级，作为后备）
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		if strings.Contains(exeDir, ".app/Contents/MacOS") {
			possibleDirs = append(possibleDirs,
				filepath.Join(exeDir, "data", "geoip"),             // Contents/MacOS/data/geoip
				filepath.Join(exeDir, "..", "Resources", "data", "geoip"), // Contents/Resources/data/geoip
			)
		}
	}

	// 尝试加载City数据库
	var cityDB *geoip2.Reader
	var loadedDir string
	cityNames := []string{"Geolite2-City.mmdb", "GeoLite2-City.mmdb"}
	
	for _, dir := range possibleDirs {
		for _, name := range cityNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				if db, err := geoip2.Open(path); err == nil {
					cityDB = db
					loadedDir = dir
					break
				}
			}
		}
		if cityDB != nil {
			break
		}
	}

	if cityDB == nil {
		return nil, fmt.Errorf("无法找到GeoIP City数据库")
	}
	service.cityDB = cityDB

	// 加载Country数据库（可选）
	countryPath := filepath.Join(loadedDir, "GeoLite2-Country.mmdb")
	if db, err := geoip2.Open(countryPath); err == nil {
		service.countryDB = db
	}

	return service, nil
}

// LookupIP 查询IP的地理位置
func (s *GeoIPService) LookupIP(ipStr string) (*Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", ipStr)
	}

	location := &Location{IP: ipStr}

	if s.cityDB != nil {
		if record, err := s.cityDB.City(ip); err == nil {
			location.Country = record.Country.Names["zh-CN"]
			if location.Country == "" {
				location.Country = record.Country.Names["en"]
			}
			location.City = record.City.Names["zh-CN"]
			if location.City == "" {
				location.City = record.City.Names["en"]
			}
			location.Latitude = record.Location.Latitude
			location.Longitude = record.Location.Longitude
		}
	}

	if location.Country == "" && location.City == "" {
		return nil, fmt.Errorf("no geo data for IP: %s", ipStr)
	}

	return location, nil
}

// BatchLookupIPs 批量查询
func (s *GeoIPService) BatchLookupIPs(ips []string) map[string]*Location {
	results := make(map[string]*Location)
	for _, ip := range ips {
		if loc, err := s.LookupIP(ip); err == nil {
			results[ip] = loc
		}
	}
	return results
}

// Close 关闭数据库
func (s *GeoIPService) Close() error {
	if s.cityDB != nil {
		s.cityDB.Close()
	}
	if s.countryDB != nil {
		s.countryDB.Close()
	}
	return nil
}
