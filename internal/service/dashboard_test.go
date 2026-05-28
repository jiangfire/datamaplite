package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDashboardService_GetStats_CacheHit(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)

	counts := &store.DashboardCountsRow{
		TotalSources:    5,
		TotalObjects:    10,
		TotalColumns:    100,
		TotalTerms:      3,
		TotalMappings:   20,
		TotalDQRules:    8,
		ActiveDQRules:   6,
		OverallPassRate: 95.5,
		TotalTags:       4,
		RecentChanges:   12,
		TotalAlertRules: 2,
		TotalUsers:      3,
		UnreadNotifications: 7,
	}

	mockStore.On("GetDashboardCounts", mock.Anything, "user-1").Return(counts, nil).Once()

	stats1, err := svc.GetStats(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats1.TotalSources)
	assert.Equal(t, int64(7), stats1.UnreadNotifications)

	stats2, err := svc.GetStats(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, stats1, stats2)

	mockStore.AssertNumberOfCalls(t, "GetDashboardCounts", 1)
}

func TestDashboardService_GetStats_CacheExpiry(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)
	svc.cacheTTL = 50 * time.Millisecond

	counts1 := &store.DashboardCountsRow{TotalSources: 5}
	counts2 := &store.DashboardCountsRow{TotalSources: 10}

	mockStore.On("GetDashboardCounts", mock.Anything, "user-1").Return(counts1, nil).Once()
	mockStore.On("GetDashboardCounts", mock.Anything, "user-1").Return(counts2, nil).Once()

	stats1, err := svc.GetStats(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats1.TotalSources)

	time.Sleep(60 * time.Millisecond)

	stats2, err := svc.GetStats(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(10), stats2.TotalSources)

	mockStore.AssertNumberOfCalls(t, "GetDashboardCounts", 2)
}

func TestDashboardService_GetStats_InvalidateCache(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)

	counts1 := &store.DashboardCountsRow{TotalSources: 5}
	counts2 := &store.DashboardCountsRow{TotalSources: 10}

	mockStore.On("GetDashboardCounts", mock.Anything, "user-1").Return(counts1, nil).Once()
	mockStore.On("GetDashboardCounts", mock.Anything, "user-1").Return(counts2, nil).Once()

	stats1, err := svc.GetStats(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats1.TotalSources)

	svc.InvalidateCache()

	stats2, err := svc.GetStats(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(10), stats2.TotalSources)

	mockStore.AssertNumberOfCalls(t, "GetDashboardCounts", 2)
}

func TestDashboardService_GetStats_StoreError(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)

	mockStore.On("GetDashboardCounts", mock.Anything, "user-1").Return(nil, fmt.Errorf("db error"))

	stats, err := svc.GetStats(context.Background(), "user-1")
	assert.Error(t, err)
	assert.Nil(t, stats)
}

func TestDashboardService_GetStats_DifferentUsers(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)

	countsA := &store.DashboardCountsRow{TotalSources: 5, UnreadNotifications: 3}
	countsB := &store.DashboardCountsRow{TotalSources: 5, UnreadNotifications: 10}

	mockStore.On("GetDashboardCounts", mock.Anything, "user-a").Return(countsA, nil).Once()
	mockStore.On("GetDashboardCounts", mock.Anything, "user-b").Return(countsB, nil).Once()

	statsA, err := svc.GetStats(context.Background(), "user-a")
	require.NoError(t, err)
	assert.Equal(t, int64(3), statsA.UnreadNotifications)

	statsB, err := svc.GetStats(context.Background(), "user-b")
	require.NoError(t, err)
	assert.Equal(t, int64(10), statsB.UnreadNotifications)

	mockStore.AssertNumberOfCalls(t, "GetDashboardCounts", 2)
}

func TestDashboardService_GetStats_EmptyUserID(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)

	counts := &store.DashboardCountsRow{TotalSources: 5, UnreadNotifications: 0}
	mockStore.On("GetDashboardCounts", mock.Anything, "").Return(counts, nil).Once()

	stats, err := svc.GetStats(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.UnreadNotifications)
}

func TestDashboardService_GetStats_ConcurrentAccess(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewDashboardService(mockStore)

	counts := &store.DashboardCountsRow{TotalSources: 5}
	mockStore.On("GetDashboardCounts", mock.Anything, mock.Anything).Return(counts, nil)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := fmt.Sprintf("user-%d", id%3)
			stats, err := svc.GetStats(context.Background(), userID)
			assert.NoError(t, err)
			assert.NotNil(t, stats)
		}(i)
	}
	wg.Wait()
}
