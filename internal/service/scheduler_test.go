package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/crypto"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestSourceService(t *testing.T, st store.Store) *SourceService {
	t.Helper()
	key := "12345678901234567890123456789012"
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)
	registry := scanner.NewRegistry()
	return NewSourceService(st, cipher, registry)
}

func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"valid 6-field", "0 0 2 * * *", false},
		{"valid every 5 min", "0 */5 * * * *", false},
		{"valid midnight", "0 0 0 * * *", false},
		{"valid complex", "30 15 3 ? * MON-FRI", false},
		{"5-field missing seconds", "0 2 * * *", true},
		{"invalid string", "invalid", true},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"too few fields", "0 0", true},
		{"too many fields", "0 0 0 0 0 0 0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronExpression(tt.expr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncSchedulerService_AddCronJob_ValidExpression(t *testing.T) {
	mockStore := new(MockStore)
	sourceSvc := newTestSourceService(t, mockStore)
	log := zap.NewNop()

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	schedule := &store.SyncScheduleRow{
		ID:             "sched-1",
		SourceID:       "src-1",
		CronExpression: "0 0 2 * * *",
		IsActive:       true,
	}

	err := svc.addCronJob(schedule)
	require.NoError(t, err)

	svc.entriesMu.RLock()
	_, exists := svc.entries["sched-1"]
	svc.entriesMu.RUnlock()
	assert.True(t, exists, "cron entry should be registered")

	svc.Stop()
}

func TestSyncSchedulerService_AddCronJob_InvalidExpression(t *testing.T) {
	mockStore := new(MockStore)
	sourceSvc := newTestSourceService(t, mockStore)
	log := zap.NewNop()

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	schedule := &store.SyncScheduleRow{
		ID:             "sched-bad",
		SourceID:       "src-1",
		CronExpression: "not-a-cron",
		IsActive:       true,
	}

	err := svc.addCronJob(schedule)
	assert.Error(t, err)

	svc.entriesMu.RLock()
	_, exists := svc.entries["sched-bad"]
	svc.entriesMu.RUnlock()
	assert.False(t, exists, "invalid cron should not register entry")

	svc.Stop()
}

func TestSyncSchedulerService_RefreshSchedules(t *testing.T) {
	mockStore := new(MockStore)
	sourceSvc := newTestSourceService(t, mockStore)
	log := zap.NewNop()

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	schedules := []*store.SyncScheduleRow{
		{ID: "s1", SourceID: "src-1", CronExpression: "0 0 3 * * *", IsActive: true},
		{ID: "s2", SourceID: "src-2", CronExpression: "0 30 1 * * *", IsActive: true},
		{ID: "s3", SourceID: "src-3", CronExpression: "0 0 4 * * *", IsActive: false},
	}

	mockStore.On("ListSyncSchedules", mock.Anything).Return(schedules, nil)

	err := svc.RefreshSchedules(context.Background())
	require.NoError(t, err)

	svc.entriesMu.RLock()
	assert.Len(t, svc.entries, 2, "only active schedules should be registered")
	_, hasS1 := svc.entries["s1"]
	_, hasS2 := svc.entries["s2"]
	_, hasS3 := svc.entries["s3"]
	svc.entriesMu.RUnlock()

	assert.True(t, hasS1)
	assert.True(t, hasS2)
	assert.False(t, hasS3, "inactive schedule should not be registered")

	mockStore.AssertExpectations(t)
	svc.Stop()
}

func TestSyncSchedulerService_RefreshSchedules_ClearsOldEntries(t *testing.T) {
	mockStore := new(MockStore)
	sourceSvc := newTestSourceService(t, mockStore)
	log := zap.NewNop()

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	first := []*store.SyncScheduleRow{
		{ID: "old-1", SourceID: "src-1", CronExpression: "0 0 2 * * *", IsActive: true},
	}
	mockStore.On("ListSyncSchedules", mock.Anything).Return(first, nil).Once()

	err := svc.RefreshSchedules(context.Background())
	require.NoError(t, err)

	svc.entriesMu.RLock()
	assert.Len(t, svc.entries, 1)
	svc.entriesMu.RUnlock()

	second := []*store.SyncScheduleRow{
		{ID: "new-1", SourceID: "src-2", CronExpression: "0 0 3 * * *", IsActive: true},
		{ID: "new-2", SourceID: "src-3", CronExpression: "0 0 4 * * *", IsActive: true},
	}
	mockStore.On("ListSyncSchedules", mock.Anything).Return(second, nil).Once()

	err = svc.RefreshSchedules(context.Background())
	require.NoError(t, err)

	svc.entriesMu.RLock()
	_, hasOld := svc.entries["old-1"]
	_, hasNew1 := svc.entries["new-1"]
	_, hasNew2 := svc.entries["new-2"]
	count := len(svc.entries)
	svc.entriesMu.RUnlock()

	assert.False(t, hasOld, "old entry should be cleared")
	assert.True(t, hasNew1)
	assert.True(t, hasNew2)
	assert.Equal(t, 2, count)

	mockStore.AssertExpectations(t)
	svc.Stop()
}

func TestSyncSchedulerService_ConcurrentRunProtection(t *testing.T) {
	mockStore := new(MockStore)
	sourceSvc := newTestSourceService(t, mockStore)
	log := zap.NewNop()

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	svc.runningMu.Lock()
	svc.running["sched-1"] = true
	svc.runningMu.Unlock()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc.runSchedule("sched-1", "src-1")
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runSchedule should have returned immediately due to concurrent protection")
	}
}

func TestSyncSchedulerService_RunSchedule_UpdatesStatus(t *testing.T) {
	mockStore := new(MockStore)
	log := zap.NewNop()

	sourceServiceStore := new(MockStore)
	key := "12345678901234567890123456789012"
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)
	registry := scanner.NewRegistry()
	sourceSvc := NewSourceService(sourceServiceStore, cipher, registry)

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	sourceServiceStore.On("GetDataSource", mock.Anything, "nonexistent-source").Return(nil, fmt.Errorf("not found"))

	mockStore.On("UpdateSyncScheduleRunStatus", mock.Anything, "sched-1", "running", mock.Anything, mock.Anything).Return(nil).Once()
	mockStore.On("UpdateSyncScheduleRunStatus", mock.Anything, "sched-1", "failed", mock.Anything, mock.Anything).Return(nil).Once()

	svc.runSchedule("sched-1", "nonexistent-source")

	mockStore.AssertExpectations(t)
	sourceServiceStore.AssertExpectations(t)
}

func TestSyncSchedulerService_Start_FailsOnReloadError(t *testing.T) {
	mockStore := new(MockStore)
	key := "12345678901234567890123456789012"
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)
	registry := scanner.NewRegistry()
	sourceSvc := NewSourceService(mockStore, cipher, registry)
	log := zap.NewNop()

	svc := NewSyncSchedulerService(mockStore, sourceSvc, log)

	mockStore.On("ListSyncSchedules", mock.Anything).Return(nil, fmt.Errorf("db error"))

	err = svc.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reload schedules")

	mockStore.AssertExpectations(t)
}
