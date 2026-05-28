package service

import (
	"context"
	"sync"
	"time"

	"github.com/jiangfire/datamaplite/internal/store"
)

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalSources        int64   `json:"total_sources"`
	TotalObjects        int64   `json:"total_objects"`
	TotalColumns        int64   `json:"total_columns"`
	TotalTerms          int64   `json:"total_terms"`
	TotalMappings       int64   `json:"total_mappings"`
	TotalDQRules        int64   `json:"total_dq_rules"`
	TotalTags           int64   `json:"total_tags"`
	RecentChanges       int64   `json:"recent_changes"`
	TotalAlertRules     int64   `json:"total_alert_rules"`
	TotalUsers          int64   `json:"total_users"`
	ActiveDQRules       int64   `json:"active_dq_rules"`
	OverallPassRate     float64 `json:"overall_pass_rate"`
	UnreadNotifications int64   `json:"unread_notifications"`
}

// DashboardService 仪表盘服务
type DashboardService struct {
	store     store.Store
	cache     map[string]*cachedStats
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
}

type cachedStats struct {
	stats     *DashboardStats
	cacheTime time.Time
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(st store.Store) *DashboardService {
	return &DashboardService{
		store:    st,
		cache:    make(map[string]*cachedStats),
		cacheTTL: 30 * time.Second,
	}
}

// GetStats 获取仪表盘统计数据（带缓存，按用户隔离未读通知）
func (s *DashboardService) GetStats(ctx context.Context, userID string) (*DashboardStats, error) {
	cacheKey := userID
	if cacheKey == "" {
		cacheKey = "__anonymous__"
	}

	s.cacheMu.RLock()
	if cached, ok := s.cache[cacheKey]; ok && time.Since(cached.cacheTime) < s.cacheTTL {
		stats := cached.stats
		s.cacheMu.RUnlock()
		return stats, nil
	}
	s.cacheMu.RUnlock()

	counts, err := s.store.GetDashboardCounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats := &DashboardStats{
		TotalSources:        counts.TotalSources,
		TotalObjects:        counts.TotalObjects,
		TotalColumns:        counts.TotalColumns,
		TotalTerms:          counts.TotalTerms,
		TotalMappings:       counts.TotalMappings,
		TotalDQRules:        counts.TotalDQRules,
		ActiveDQRules:       counts.ActiveDQRules,
		OverallPassRate:     counts.OverallPassRate,
		TotalTags:           counts.TotalTags,
		RecentChanges:       counts.RecentChanges,
		TotalAlertRules:     counts.TotalAlertRules,
		TotalUsers:          counts.TotalUsers,
		UnreadNotifications: counts.UnreadNotifications,
	}

	s.cacheMu.Lock()
	s.cache[cacheKey] = &cachedStats{stats: stats, cacheTime: time.Now()}
	s.cacheMu.Unlock()

	return stats, nil
}

// InvalidateCache 使缓存失效
func (s *DashboardService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cache = make(map[string]*cachedStats)
	s.cacheMu.Unlock()
}
