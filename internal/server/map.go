package server

import (
	"fmt"
	"net"
	"time"

	"fastmonitor/internal/store"
	"fastmonitor/pkg/model"
)

// MapDataPoint 地图数据点
type MapDataPoint struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	City        string   `json:"city"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Value       int      `json:"value"`         // 连接数
	Connections int      `json:"connections"`
	UniqueIPs   int      `json:"unique_ips"`
	IPs         []string `json:"ips"`
	Type        string   `json:"type"` // "country" or "city"
}

// MapDataResponse 地图数据响应
type MapDataResponse struct {
	MapPoints        []MapDataPoint `json:"map_points"`
	CountryMapPoints []MapDataPoint `json:"country_map_points"`
	CityMapPoints    []MapDataPoint `json:"city_map_points"`
	CountryStats     []CountryStat  `json:"country_stats"`
	CityStats        []CityStat     `json:"city_stats"`
	TopIPs           []IPStat       `json:"top_ips"`
	TotalSessions    int            `json:"total_sessions"`
	UniqueIPs        int            `json:"unique_ips"`
	LastUpdate       time.Time      `json:"last_update"`
}

// CountryStat 国家统计
type CountryStat struct {
	Country     string   `json:"country"`
	Connections int      `json:"connections"`
	UniqueIPs   int      `json:"unique_ips"`
	Cities      []string `json:"cities"`
	IPs         []string `json:"ips"`
}

// CityStat 城市统计
type CityStat struct {
	Country     string   `json:"country"`
	City        string   `json:"city"`
	Connections int      `json:"connections"`
	UniqueIPs   int      `json:"unique_ips"`
	IPs         []string `json:"ips"`
}

// IPStat IP统计
type IPStat struct {
	IP      string `json:"ip"`
	Count   int    `json:"count"`
	Country string `json:"country"`
	City    string `json:"city"`
}

// GetMapData 获取地图数据（从会话流去重后获取）
func (a *App) GetMapData() (*MapDataResponse, error) {
	fmt.Println("[GetMapData] 开始获取地图数据...")
	
	composite, ok := a.store.(*store.CompositeStore)
	if !ok {
		return nil, fmt.Errorf("store is not composite")
	}

	sqliteStore := composite.GetDB()

	// 1. 从session_flows获取所有会话流
	flows, err := sqliteStore.QuerySessionFlows(model.SessionFlowQuery{
		Limit:     10000, // 获取大量数据用于地图展示
		Offset:    0,
		SortBy:    "packet_count",
		SortOrder: "desc",
	})
	if err != nil {
		fmt.Printf("[GetMapData] ❌ 查询会话流失败: %v\n", err)
		return nil, fmt.Errorf("query session flows: %w", err)
	}

	fmt.Printf("[GetMapData] 查询到 %d 个会话流\n", len(flows.Data))

	if len(flows.Data) == 0 {
		fmt.Println("[GetMapData] ⚠️ 没有会话流数据，返回空结果")
		// 返回空数据
		return &MapDataResponse{
			MapPoints:        []MapDataPoint{},
			CountryMapPoints: []MapDataPoint{},
			CityMapPoints:    []MapDataPoint{},
			CountryStats:     []CountryStat{},
			CityStats:        []CityStat{},
			TopIPs:           []IPStat{},
			TotalSessions:    0,
			UniqueIPs:        0,
			LastUpdate:       time.Now(),
		}, nil
	}

	// 2. 提取唯一的外网IP
	uniqueIPs := make(map[string]bool)
	for _, flow := range flows.Data {
		// 只处理目标IP (dst_ip)
		if flow.DstIP != "" && !isLocalIP(flow.DstIP) {
			uniqueIPs[flow.DstIP] = true
		}
	}

	// 转换为切片
	ipList := make([]string, 0, len(uniqueIPs))
	for ip := range uniqueIPs {
		ipList = append(ipList, ip)
	}

	fmt.Printf("[GetMapData] 提取到 %d 个唯一外网IP\n", len(ipList))
	if len(ipList) > 0 {
		fmt.Printf("[GetMapData] 示例IP: %s\n", ipList[0])
	}

	// 3. 批量解析IP地理位置
	geoLocations, err := a.batchResolveGeoIP(sqliteStore, ipList)
	if err != nil {
		fmt.Printf("[GetMapData] ❌ 批量解析GeoIP失败: %v\n", err)
		return nil, fmt.Errorf("batch resolve geo ip: %w", err)
	}

	fmt.Printf("[GetMapData] 成功解析 %d 个IP的地理位置\n", len(geoLocations))

	// 4. 聚合数据
	countryMap := make(map[string]*CountryStat)
	cityMap := make(map[string]*CityStat)
	ipCountMap := make(map[string]int)

	for _, flow := range flows.Data {
		dstIP := flow.DstIP
		if dstIP == "" || isLocalIP(dstIP) {
			continue
		}

		geo, exists := geoLocations[dstIP]
		if !exists {
			continue
		}

		// 统计IP出现次数
		ipCountMap[dstIP] += int(flow.PacketCount)

		// 国家统计
		if geo.Country != "" {
			if _, exists := countryMap[geo.Country]; !exists {
				countryMap[geo.Country] = &CountryStat{
					Country:     geo.Country,
					Connections: 0,
					Cities:      []string{},
					IPs:         []string{},
				}
			}
			stat := countryMap[geo.Country]
			stat.Connections += int(flow.PacketCount)

			// 添加城市（去重）
			if geo.City != "" && !contains(stat.Cities, geo.City) {
				stat.Cities = append(stat.Cities, geo.City)
			}

			// 添加IP（去重）
			if !contains(stat.IPs, dstIP) {
				stat.IPs = append(stat.IPs, dstIP)
				stat.UniqueIPs++
			}
		}

		// 城市统计
		if geo.City != "" {
			cityKey := geo.Country + "|" + geo.City
			if _, exists := cityMap[cityKey]; !exists {
				cityMap[cityKey] = &CityStat{
					Country:     geo.Country,
					City:        geo.City,
					Connections: 0,
					IPs:         []string{},
				}
			}
			stat := cityMap[cityKey]
			stat.Connections += int(flow.PacketCount)

			if !contains(stat.IPs, dstIP) {
				stat.IPs = append(stat.IPs, dstIP)
				stat.UniqueIPs++
			}
		}
	}

	// 5. 生成地图点数据
	cityMapPoints := []MapDataPoint{}
	for _, cityStat := range cityMap {
		// 从geoLocations获取第一个该城市的坐标
		var lat, lng float64
		for _, ip := range cityStat.IPs {
			if geo, ok := geoLocations[ip]; ok {
				if geo.City == cityStat.City {
					lat = geo.Latitude
					lng = geo.Longitude
					break
				}
			}
		}

		cityMapPoints = append(cityMapPoints, MapDataPoint{
			Name:        cityStat.City + ", " + cityStat.Country,
			Country:     cityStat.Country,
			City:        cityStat.City,
			Latitude:    lat,
			Longitude:   lng,
			Value:       cityStat.Connections,
			Connections: cityStat.Connections,
			UniqueIPs:   cityStat.UniqueIPs,
			IPs:         cityStat.IPs,
			Type:        "city",
		})
	}

	// 国家级地图点
	countryMapPoints := []MapDataPoint{}
	for _, countryStat := range countryMap {
		// 优先从实际IP地理位置计算国家中心坐标
		coords := calculateCountryCoordsFromGeo(countryStat.Country, countryStat.IPs, geoLocations)
		
		// 如果计算失败，使用默认坐标
		if coords.Lat == 0 && coords.Lng == 0 {
			coords = getDefaultCountryCoords(countryStat.Country)
		}

		countryMapPoints = append(countryMapPoints, MapDataPoint{
			Name:        countryStat.Country,
			Country:     countryStat.Country,
			City:        "",
			Latitude:    coords.Lat,
			Longitude:   coords.Lng,
			Value:       countryStat.Connections,
			Connections: countryStat.Connections,
			UniqueIPs:   countryStat.UniqueIPs,
			IPs:         countryStat.IPs,
			Type:        "country",
		})
	}

	// 6. TOP IP统计
	topIPs := []IPStat{}
	for ip, count := range ipCountMap {
		geo := geoLocations[ip]
		topIPs = append(topIPs, IPStat{
			IP:      ip,
			Count:   count,
			Country: geo.Country,
			City:    geo.City,
		})
	}

	// 按count排序
	sortIPStats(topIPs)
	if len(topIPs) > 20 {
		topIPs = topIPs[:20]
	}

	// 7. 转换统计数据为切片
	countryStats := []CountryStat{}
	for _, stat := range countryMap {
		countryStats = append(countryStats, *stat)
	}

	cityStats := []CityStat{}
	for _, stat := range cityMap {
		cityStats = append(cityStats, *stat)
	}

	return &MapDataResponse{
		MapPoints:        cityMapPoints, // 默认使用城市级
		CountryMapPoints: countryMapPoints,
		CityMapPoints:    cityMapPoints,
		CountryStats:     countryStats,
		CityStats:        cityStats,
		TopIPs:           topIPs,
		TotalSessions:    len(flows.Data),
		UniqueIPs:        len(uniqueIPs),
		LastUpdate:       time.Now(),
	}, nil
}

// batchResolveGeoIP 批量解析IP地理位置 (带缓存)
func (a *App) batchResolveGeoIP(sqliteStore *store.SQLiteStore, ips []string) (map[string]*store.IPGeoLocation, error) {
	// 1. 先从缓存获取
	cached, err := sqliteStore.BatchGetIPGeoLocations(ips)
	if err != nil {
		return nil, fmt.Errorf("get cached geo: %w", err)
	}

	result := make(map[string]*store.IPGeoLocation)
	uncached := []string{}

	for _, ip := range ips {
		if geo, exists := cached[ip]; exists {
			result[ip] = geo
		} else {
			uncached = append(uncached, ip)
		}
	}

	// 2. 如果有未缓存的IP，使用GeoIP服务解析
	if len(uncached) > 0 {
		// 使用单例的 GeoIP 服务
		if a.geoipService == nil {
			return nil, fmt.Errorf("GeoIP service not initialized")
		}

		newGeos := []*store.IPGeoLocation{}
		for _, ip := range uncached {
			location, err := a.geoipService.LookupIP(ip)
			if err != nil {
				// 解析失败，跳过
				continue
			}

			geo := &store.IPGeoLocation{
				IP:        ip,
				Country:   location.Country,
				City:      location.City,
				Latitude:  location.Latitude,
				Longitude: location.Longitude,
				CachedAt:  time.Now(),
			}

			newGeos = append(newGeos, geo)
			result[ip] = geo
		}

		// 3. 保存新解析的地理位置到缓存
		if len(newGeos) > 0 {
			if err := sqliteStore.BatchSaveIPGeoLocations(newGeos); err != nil {
				fmt.Printf("Warning: failed to save geo cache: %v\n", err)
			}
		}
	}

	return result, nil
}

// 辅助函数

func isLocalIP(ipStr string) bool {
	if ipStr == "" {
		return true
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true // 无法解析的IP视为本地
	}

	// IPv6地址 - 跳过所有IPv6
	if ip.To4() == nil {
		return true
	}

	// 检查是否为本地IP
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	
	// 组播地址 (224.0.0.0/4)
	if ip.IsMulticast() {
		return true
	}

	// 检查常见的私有IP段
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // 链路本地地址
		"0.0.0.0/8",      // 当前网络
		"255.255.255.255/32", // 广播地址
	}

	for _, cidr := range privateRanges {
		_, ipnet, _ := net.ParseCIDR(cidr)
		if ipnet != nil && ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

type CountryCoords struct {
	Lat float64
	Lng float64
}

// calculateCountryCoordsFromGeo 从实际IP地理位置数据计算国家中心坐标
func calculateCountryCoordsFromGeo(country string, ips []string, geoLocations map[string]*store.IPGeoLocation) CountryCoords {
	var latSum, lngSum float64
	var count int
	
	// 遍历该国家的所有IP,计算平均坐标
	for _, ip := range ips {
		if geo, exists := geoLocations[ip]; exists {
			if geo.Country == country && geo.Latitude != 0 && geo.Longitude != 0 {
				latSum += geo.Latitude
				lngSum += geo.Longitude
				count++
			}
		}
	}
	
	// 如果有有效坐标,返回平均值
	if count > 0 {
		return CountryCoords{
			Lat: latSum / float64(count),
			Lng: lngSum / float64(count),
		}
	}
	
	// 如果没有有效数据,返回零值,外层会使用默认坐标
	return CountryCoords{Lat: 0, Lng: 0}
}

// getDefaultCountryCoords 获取预定义的国家默认坐标(兜底方案)
// 仅当MMDB数据无法提供坐标时使用
func getDefaultCountryCoords(country string) CountryCoords {
	coords := map[string]CountryCoords{
		"中国":      {39.9042, 116.4074},
		"美国":      {39.8283, -98.5795},
		"日本":      {35.6762, 139.6503},
		"英国":      {51.5074, -0.1278},
		"德国":      {52.5200, 13.4050},
		"法国":      {48.8566, 2.3522},
		"加拿大":    {56.1304, -106.3468},
		"澳大利亚":  {-25.2744, 133.7751},
		"俄罗斯":    {55.7558, 37.6173},
		"印度":      {20.5937, 78.9629},
		"巴西":      {-14.2350, -51.9253},
		"韩国":      {37.5665, 126.9780},
		"新加坡":    {1.3521, 103.8198},
		"荷兰":      {52.1326, 5.2913},
		"瑞典":      {60.1282, 18.6435},
		"瑞士":      {46.8182, 8.2275},
		"意大利":    {41.8719, 12.5674},
		"西班牙":    {40.4637, -3.7492},
		"波兰":      {51.9194, 19.1451},
		"土耳其":    {38.9637, 35.2433},
		"墨西哥":    {23.6345, -102.5528},
		"阿根廷":    {-38.4161, -63.6167},
		"南非":      {-30.5595, 22.9375},
		"埃及":      {26.8206, 30.8025},
		"泰国":      {15.8700, 100.9925},
		"越南":      {14.0583, 108.2772},
		"印度尼西亚": {-0.7893, 113.9213},
		"菲律宾":    {12.8797, 121.7740},
		"马来西亚":  {4.2105, 101.9758},
		"香港":      {22.3193, 114.1694},
		"台湾":      {23.6978, 120.9605},
	}

	if coord, ok := coords[country]; ok {
		return coord
	}

	return CountryCoords{0, 0}
}

func sortIPStats(stats []IPStat) {
	// 简单的冒泡排序 (降序)
	for i := 0; i < len(stats)-1; i++ {
		for j := 0; j < len(stats)-i-1; j++ {
			if stats[j].Count < stats[j+1].Count {
				stats[j], stats[j+1] = stats[j+1], stats[j]
			}
		}
	}
}

