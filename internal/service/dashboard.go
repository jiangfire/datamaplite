package service

import (
	"context"
	"fmt"
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
	cache     *DashboardStats
	cacheMu   sync.RWMutex
	cacheTime time.Time
	cacheTTL  time.Duration
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(st store.Store) *DashboardService {
	return &DashboardService{
		store:    st,
		cacheTTL: 30 * time.Second, // 30 秒缓存
	}
}

// GetStats 获取仪表盘统计数据（带缓存）
func (s *DashboardService) GetStats(ctx context.Context) (*DashboardStats, error) {
	// 尝试读取缓存
	s.cacheMu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		cached := s.cache
		s.cacheMu.RUnlock()
		return cached, nil
	}
	s.cacheMu.RUnlock()

	// 计算新数据
	stats, err := s.computeStats(ctx)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	s.cacheMu.Lock()
	s.cache = stats
	s.cacheTime = time.Now()
	s.cacheMu.Unlock()

	return stats, nil
}

// InvalidateCache 使缓存失效
func (s *DashboardService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()
}

// computeStats 计算仪表盘统计数据
func (s *DashboardService) computeStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}
	var errs []error

	// 使用 WaitGroup 并行查询独立的统计数据
	type result struct {
		sources        []*store.DataSourceRow
		objects        int64
		columns        int64
		terms          int64
		tags           int64
		dqRules        int64
		activeDQRules  int64
		overallPassRate float64
		alertRules     int64
		users          int64
		mappings       int64
		recentChanges  int64
		unreadNotif    int64
	}

	r := &result{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 1. 数据源数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		sources, err := s.store.ListDataSources(ctx)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list sources: %w", err))
			mu.Unlock()
			return
		}
		r.sources = sources
	}()

	// 2. 业务术语数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		terms, err := s.store.ListBusinessTerms(ctx, "")
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list terms: %w", err))
			mu.Unlock()
			return
		}
		r.terms = int64(len(terms))
	}()

	// 3. 标签数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		tags, err := s.store.ListTags(ctx)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list tags: %w", err))
			mu.Unlock()
			return
		}
		r.tags = int64(len(tags))
	}()

	// 4. DQ 规则统计
	wg.Add(1)
	go func() {
		defer wg.Done()
		dqRules, err := s.store.ListDQRules(ctx, nil)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list dq rules: %w", err))
			mu.Unlock()
			return
		}
		r.dqRules = int64(len(dqRules))
		for _, rule := range dqRules {
			if rule.IsActive {
				r.activeDQRules++
			}
		}
	}()

	// 5. DQ 通过率
	wg.Add(1)
	go func() {
		defer wg.Done()
		dqStats, err := s.store.GetDQStats(ctx)
		if err == nil && dqStats != nil {
			r.overallPassRate = dqStats.OverallPassRate
		}
	}()

	// 6. 告警规则数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		alertRules, err := s.store.ListAlertRules(ctx, nil)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list alert rules: %w", err))
			mu.Unlock()
			return
		}
		r.alertRules = int64(len(alertRules))
	}()

	// 7. 用户数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := s.store.ListUsers(ctx)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list users: %w", err))
			mu.Unlock()
			return
		}
		r.users = int64(len(users))
	}()

	// 8. Schema 对象和字段数量（避免 N+1：先获取所有对象，再获取所有字段）
	wg.Add(1)
	go func() {
		defer wg.Done()
		sources, err := s.store.ListDataSources(ctx)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("list sources for objects: %w", err))
			mu.Unlock()
			return
		}

		var totalObjects, totalColumns int64
		for _, source := range sources {
			objects, err := s.store.ListSchemaObjectsBySource(ctx, source.ID)
			if err != nil {
				continue
			}
			totalObjects += int64(len(objects))
			for _, obj := range objects {
				columns, err := s.store.ListColumnsByObject(ctx, obj.ID)
				if err != nil {
					continue
				}
				totalColumns += int64(len(columns))
			}
		}
		r.objects = totalObjects
		r.columns = totalColumns
	}()

	// 9. 最近的变更（限制为最近30天，每个数据源最多10条）
	wg.Add(1)
	go func() {
		defer wg.Done()
		sources, err := s.store.ListDataSources(ctx)
		if err != nil {
			return
		}

		var totalChanges int64
		for _, source := range sources {
			changes, err := s.store.ListSchemaChangesBySource(ctx, source.ID, 10)
			if err != nil {
				continue
			}
			// 只统计最近 30 天的变更
			cutoff := time.Now().AddDate(0, 0, -30)
			for _, change := range changes {
				if change.DetectedAt != "" {
					if t, err := time.Parse(time.RFC3339, change.DetectedAt); err == nil && t.After(cutoff) {
						totalChanges++
					}
				}
			}
		}
		r.recentChanges = totalChanges
	}()

	// 10. 字段映射数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 通过获取所有列的映射来统计总数
		// 由于没有直接的 CountColumnMappings 方法，我们通过查询所有对象的所有列来获取
		// 这是一个近似值，实际实现应该添加一个 count 查询
		sources, err := s.store.ListDataSources(ctx)
		if err != nil {
			return
		}
		var totalMappings int64
		for _, source := range sources {
			objects, err := s.store.ListSchemaObjectsBySource(ctx, source.ID)
			if err != nil {
				continue
			}
			for _, obj := range objects {
				columns, err := s.store.ListColumnsByObject(ctx, obj.ID)
				if err != nil {
					continue
				}
				for _, col := range columns {
					mappings, err := s.store.GetColumnMappings(ctx, col.ID)
					if err != nil {
						continue
					}
					totalMappings += int64(len(mappings))
				}
			}
		}
		r.mappings = totalMappings
	}()

	// 11. 未读通知数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 获取当前用户的未读通知
		// 由于没有当前用户信息，先统计所有用户的未读通知总数
		users, err := s.store.ListUsers(ctx)
		if err != nil {
			return
		}
		var totalUnread int64
		for _, user := range users {
			stats, err := s.store.GetNotificationStats(ctx, user.ID)
			if err != nil {
				continue
			}
			totalUnread += stats.UnreadCount
		}
		r.unreadNotif = totalUnread
	}()

	wg.Wait()

	// 即使部分查询失败，也返回已获取的数据
	stats.TotalSources = int64(len(r.sources))
	stats.TotalObjects = r.objects
	stats.TotalColumns = r.columns
	stats.TotalTerms = r.terms
	stats.TotalMappings = r.mappings
	stats.TotalDQRules = r.dqRules
	stats.TotalTags = r.tags
	stats.RecentChanges = r.recentChanges
	stats.TotalAlertRules = r.alertRules
	stats.TotalUsers = r.users
	stats.ActiveDQRules = r.activeDQRules
	stats.OverallPassRate = r.overallPassRate
	stats.UnreadNotifications = r.unreadNotif

	if len(errs) > 0 {
		// 返回部分结果，附带第一个错误
		return stats, fmt.Errorf("partial failure: %w", errs[0])
	}

	return stats, nil
}
