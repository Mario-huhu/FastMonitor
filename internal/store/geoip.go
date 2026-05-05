package store

import (
	"database/sql"
	"time"
)

// IPGeoLocation IP地理位置信息
type IPGeoLocation struct {
	IP        string    `json:"ip"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	ASN       string    `json:"asn"`
	Org       string    `json:"org"`
	CachedAt  time.Time `json:"cached_at"`
}

// InitGeoIPSchema 初始化GeoIP缓存表
func (s *SQLiteStore) InitGeoIPSchema() error {
	schema := `
	-- IP地理位置缓存表
	CREATE TABLE IF NOT EXISTS ip_geo_cache (
		ip TEXT PRIMARY KEY NOT NULL,
		country TEXT,
		city TEXT,
		latitude REAL,
		longitude REAL,
		asn TEXT,
		org TEXT,
		cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_geo_country ON ip_geo_cache(country);
	CREATE INDEX IF NOT EXISTS idx_geo_city ON ip_geo_cache(city);
	CREATE INDEX IF NOT EXISTS idx_geo_updated_at ON ip_geo_cache(updated_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// GetIPGeoLocation 从缓存获取IP地理位置
func (s *SQLiteStore) GetIPGeoLocation(ip string) (*IPGeoLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var geo IPGeoLocation
	err := s.db.QueryRow(`
		SELECT ip, country, city, latitude, longitude, asn, org, cached_at
		FROM ip_geo_cache
		WHERE ip = ?
	`, ip).Scan(
		&geo.IP,
		&geo.Country,
		&geo.City,
		&geo.Latitude,
		&geo.Longitude,
		&geo.ASN,
		&geo.Org,
		&geo.CachedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 未找到缓存
	}
	if err != nil {
		return nil, err
	}

	return &geo, nil
}

// BatchGetIPGeoLocations 批量获取IP地理位置
func (s *SQLiteStore) BatchGetIPGeoLocations(ips []string) (map[string]*IPGeoLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*IPGeoLocation)

	if len(ips) == 0 {
		return result, nil
	}

	// 构建IN查询
	query := `SELECT ip, country, city, latitude, longitude, asn, org, cached_at FROM ip_geo_cache WHERE ip IN (`
	args := make([]interface{}, len(ips))
	for i, ip := range ips {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = ip
	}
	query += ")"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var geo IPGeoLocation
		err := rows.Scan(
			&geo.IP,
			&geo.Country,
			&geo.City,
			&geo.Latitude,
			&geo.Longitude,
			&geo.ASN,
			&geo.Org,
			&geo.CachedAt,
		)
		if err != nil {
			return nil, err
		}
		result[geo.IP] = &geo
	}

	return result, rows.Err()
}

// SaveIPGeoLocation 保存IP地理位置到缓存
func (s *SQLiteStore) SaveIPGeoLocation(geo *IPGeoLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO ip_geo_cache 
		(ip, country, city, latitude, longitude, asn, org, cached_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		geo.IP,
		geo.Country,
		geo.City,
		geo.Latitude,
		geo.Longitude,
		geo.ASN,
		geo.Org,
		time.Now(),
		time.Now(),
	)

	return err
}

// BatchSaveIPGeoLocations 批量保存IP地理位置
func (s *SQLiteStore) BatchSaveIPGeoLocations(geoList []*IPGeoLocation) error {
	if len(geoList) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO ip_geo_cache 
		(ip, country, city, latitude, longitude, asn, org, cached_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, geo := range geoList {
		_, err = stmt.Exec(
			geo.IP,
			geo.Country,
			geo.City,
			geo.Latitude,
			geo.Longitude,
			geo.ASN,
			geo.Org,
			now,
			now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CleanOldGeoCache 清理过期的地理位置缓存（超过30天）
func (s *SQLiteStore) CleanOldGeoCache(days int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	_, err := s.db.Exec(`DELETE FROM ip_geo_cache WHERE updated_at < ?`, cutoff)
	return err
}

// GetGeoStats 获取地理位置统计信息
func (s *SQLiteStore) GetGeoStats() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// 总缓存数
	var totalCount int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM ip_geo_cache`).Scan(&totalCount)
	if err != nil {
		return nil, err
	}
	stats["total_cached"] = totalCount

	// 国家数
	var countryCount int
	err = s.db.QueryRow(`SELECT COUNT(DISTINCT country) FROM ip_geo_cache WHERE country != ''`).Scan(&countryCount)
	if err != nil {
		return nil, err
	}
	stats["unique_countries"] = countryCount

	// 城市数
	var cityCount int
	err = s.db.QueryRow(`SELECT COUNT(DISTINCT city) FROM ip_geo_cache WHERE city != ''`).Scan(&cityCount)
	if err != nil {
		return nil, err
	}
	stats["unique_cities"] = cityCount

	return stats, nil
}

