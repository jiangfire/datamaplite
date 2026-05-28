package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// SyncSchedulerService 定时同步调度服务
type SyncSchedulerService struct {
	store         store.Store
	sourceService *SourceService
	cron          *cron.Cron
	entries       map[string]cron.EntryID // schedule_id -> cron entry id
	entriesMu     sync.RWMutex            // 保护 entries map
	running       map[string]bool         // 防止同一个 schedule 并发执行
	runningMu     sync.Mutex
	log           *zap.Logger
}

// NewSyncSchedulerService 创建定时同步调度服务
func NewSyncSchedulerService(st store.Store, sourceService *SourceService, log *zap.Logger) *SyncSchedulerService {
	return &SyncSchedulerService{
		store:         st,
		sourceService: sourceService,
		cron:          cron.New(cron.WithSeconds()),
		entries:       make(map[string]cron.EntryID),
		running:       make(map[string]bool),
		log:           log,
	}
}

// Start 启动调度器
func (s *SyncSchedulerService) Start(ctx context.Context) error {
	if err := s.reloadSchedules(ctx); err != nil {
		return fmt.Errorf("failed to reload schedules: %w", err)
	}
	s.cron.Start()
	s.log.Info("sync scheduler started")
	return nil
}

// Stop 停止调度器
func (s *SyncSchedulerService) Stop() {
	s.cron.Stop()
	s.log.Info("sync scheduler stopped")
}

// reloadSchedules 从数据库重新加载所有启用的 schedule
func (s *SyncSchedulerService) reloadSchedules(ctx context.Context) error {
	// 清除现有任务
	s.entriesMu.Lock()
	for _, entryID := range s.entries {
		s.cron.Remove(entryID)
	}
	s.entries = make(map[string]cron.EntryID)
	s.entriesMu.Unlock()

	schedules, err := s.store.ListSyncSchedules(ctx)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if !schedule.IsActive {
			continue
		}
		if err := s.addCronJob(schedule); err != nil {
			s.log.Warn("failed to add cron job",
				zap.String("schedule_id", schedule.ID),
				zap.String("cron", schedule.CronExpression),
				zap.Error(err))
		}
	}

	return nil
}

// addCronJob 添加一个 cron 任务
func (s *SyncSchedulerService) addCronJob(schedule *store.SyncScheduleRow) error {
	// 拷贝 schedule 数据到闭包，避免循环变量重用问题
	sid := schedule.ID
	sourceID := schedule.SourceID
	cronExpr := schedule.CronExpression

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.runSchedule(sid, sourceID)
	})
	if err != nil {
		return err
	}

	s.entriesMu.Lock()
	s.entries[sid] = entryID
	s.entriesMu.Unlock()

	s.log.Info("added sync schedule",
		zap.String("schedule_id", sid),
		zap.String("source_id", sourceID),
		zap.String("cron", cronExpr))
	return nil
}

// runSchedule 执行定时同步
func (s *SyncSchedulerService) runSchedule(scheduleID, sourceID string) {
	// 防止同一个 schedule 并发执行
	s.runningMu.Lock()
	if s.running[scheduleID] {
		s.runningMu.Unlock()
		s.log.Warn("skipping concurrent sync run",
			zap.String("schedule_id", scheduleID))
		return
	}
	s.running[scheduleID] = true
	s.runningMu.Unlock()

	defer func() {
		s.runningMu.Lock()
		delete(s.running, scheduleID)
		s.runningMu.Unlock()
	}()

	// 使用带 timeout 的 context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 更新状态为 running
	if err := s.store.UpdateSyncScheduleRunStatus(ctx, scheduleID, "running", nil, nil); err != nil {
		s.log.Warn("failed to update schedule status to running", zap.Error(err))
	}

	// 执行同步
	if err := s.sourceService.TriggerSync(ctx, sourceID); err != nil {
		s.log.Error("scheduled sync failed",
			zap.String("schedule_id", scheduleID),
			zap.String("source_id", sourceID),
			zap.Error(err))
		// 更新状态为 failed
		errMsg := err.Error()
		if updateErr := s.store.UpdateSyncScheduleRunStatus(ctx, scheduleID, "failed", &errMsg, nil); updateErr != nil {
			s.log.Warn("failed to update schedule status to failed", zap.Error(updateErr))
		}
		return
	}

	// 计算下次运行时间
	nextRun := ""
	s.entriesMu.RLock()
	if entryID, ok := s.entries[scheduleID]; ok {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			nextRun = entry.Next.Format(time.RFC3339)
		}
	}
	s.entriesMu.RUnlock()

	// 更新状态为 success
	if err := s.store.UpdateSyncScheduleRunStatus(ctx, scheduleID, "success", nil, &nextRun); err != nil {
		s.log.Warn("failed to update schedule status to success", zap.Error(err))
	}

	s.log.Info("scheduled sync completed",
		zap.String("schedule_id", scheduleID),
		zap.String("source_id", sourceID))
}

// RefreshSchedules 刷新所有定时任务（在 schedule 变更后调用）
func (s *SyncSchedulerService) RefreshSchedules(ctx context.Context) error {
	return s.reloadSchedules(ctx)
}

// ValidateCronExpression 验证 cron 表达式是否有效（6 字段格式：秒 分 时 日 月 周）
func ValidateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(expr)
	return err
}
