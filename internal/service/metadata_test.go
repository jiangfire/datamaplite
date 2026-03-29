package service

import (
	"context"
	"errors"
	"testing"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupMetadataService(t *testing.T) (*MetadataService, *MockStore) {
	mockStore := new(MockStore)
	service := NewMetadataService(mockStore)
	return service, mockStore
}

func TestMetadataService_GetSchemaTree(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	sourceID := "source-123"

	// Mock数据源存在
	mockStore.On("GetDataSource", ctx, sourceID).Return(&store.DataSourceRow{
		ID:   sourceID,
		Name: "test-mysql",
		Type: "mysql",
	}, nil)

	// Mock对象列表
	mockStore.On("ListSchemaObjectsBySource", ctx, sourceID).Return([]*store.SchemaObjectRow{
		{
			ID:          "obj-1",
			SourceID:    sourceID,
			Name:        "users",
			Type:        "table",
			ColumnCount: 2,
		},
	}, nil)

	// Mock字段列表 - 使用mock.Anything匹配任何objectID
	mockStore.On("ListColumnsByObject", ctx, mock.Anything).Return([]*store.ColumnRow{
		{
			ID:              "col-1",
			ObjectID:        "obj-1",
			Name:            "id",
			DataType:        "int",
			FullDataType:    "int(11)",
			IsNullable:      false,
			IsPrimaryKey:    true,
			OrdinalPosition: 1,
		},
		{
			ID:              "col-2",
			ObjectID:        "obj-1",
			Name:            "name",
			DataType:        "varchar",
			FullDataType:    "varchar(255)",
			IsNullable:      true,
			OrdinalPosition: 2,
		},
	}, nil)

	tree, err := service.GetSchemaTree(ctx, sourceID)

	require.NoError(t, err)
	assert.Equal(t, sourceID, tree.SourceID)
	assert.Len(t, tree.Objects, 1)
	assert.Equal(t, "users", tree.Objects[0].Name)
	assert.Len(t, tree.Objects[0].Columns, 2)
	assert.Equal(t, "id", tree.Objects[0].Columns[0].Name)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetSchemaTree_SourceNotFound(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	sourceID := "non-existent"
	mockStore.On("GetDataSource", ctx, sourceID).Return(nil, errors.New("source not found"))

	_, err := service.GetSchemaTree(ctx, sourceID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source not found")
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetColumnDetail(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"
	objectID := "obj-1"
	sourceID := "source-123"

	mockStore.On("GetColumn", ctx, columnID).Return(&store.ColumnRow{
		ID:              columnID,
		ObjectID:        objectID,
		Name:            "user_id",
		DataType:        "int",
		FullDataType:    "int(11)",
		IsNullable:      false,
		IsPrimaryKey:    true,
		OrdinalPosition: 1,
	}, nil)

	mockStore.On("GetSchemaObject", ctx, objectID).Return(&store.SchemaObjectRow{
		ID:       objectID,
		SourceID: sourceID,
		Name:     "users",
		Type:     "table",
	}, nil)

	mockStore.On("GetDataSource", ctx, sourceID).Return(&store.DataSourceRow{
		ID:   sourceID,
		Name: "test-mysql",
		Type: "mysql",
	}, nil)
	mockStore.On("GetColumnMappings", ctx, columnID).Return([]*store.ColumnMappingRow{}, nil)

	detail, err := service.GetColumnDetail(ctx, columnID)

	require.NoError(t, err)
	assert.Equal(t, columnID, detail.ID)
	assert.Equal(t, "user_id", detail.Name)
	assert.Equal(t, "users", detail.Object.Name)
	assert.Equal(t, "test-mysql", detail.Source.Name)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetColumnDetail_WithTerm(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"
	objectID := "obj-1"
	sourceID := "source-123"
	termID := "term-1"

	mockStore.On("GetColumn", ctx, columnID).Return(&store.ColumnRow{
		ID:              columnID,
		ObjectID:        objectID,
		Name:            "customer_id",
		DataType:        "int",
		TermID:          &termID,
		OrdinalPosition: 1,
	}, nil)

	mockStore.On("GetSchemaObject", ctx, objectID).Return(&store.SchemaObjectRow{
		ID:       objectID,
		SourceID: sourceID,
		Name:     "customers",
		Type:     "table",
	}, nil)

	mockStore.On("GetDataSource", ctx, sourceID).Return(&store.DataSourceRow{
		ID:   sourceID,
		Name: "test-mysql",
		Type: "mysql",
	}, nil)
	mockStore.On("GetBusinessTerm", ctx, termID).Return(&store.BusinessTermRow{
		ID:   termID,
		Name: "客户ID",
	}, nil)
	mockStore.On("GetColumnMappings", ctx, columnID).Return([]*store.ColumnMappingRow{
		{
			ID:               "map-1",
			SourceColumnID:   columnID,
			TargetColumnID:   "col-2",
			TargetColumnName: "customer_code",
			TargetObjectName: "customers",
			TargetSourceName: "dw",
			MappingType:      "alias",
			Confidence:       0.9,
		},
	}, nil)

	detail, err := service.GetColumnDetail(ctx, columnID)

	require.NoError(t, err)
	assert.NotNil(t, detail.Term)
	assert.Equal(t, termID, detail.Term.ID)
	assert.Equal(t, "客户ID", detail.Term.Name)
	assert.Len(t, detail.MappedColumns, 1)
	assert.Equal(t, "customer_code", detail.MappedColumns[0].Name)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_SearchColumns(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	query := "user"
	limit := 10

	mockStore.On("SearchColumns", ctx, query, limit).Return([]*store.ColumnSearchRow{
		{
			ColumnRow: store.ColumnRow{
				ID:       "col-1",
				Name:     "user_id",
				DataType: "int",
			},
			ObjectName: "users",
			SourceID:   "source-1",
			SourceName: "mysql-prod",
			SourceType: "mysql",
		},
		{
			ColumnRow: store.ColumnRow{
				ID:       "col-2",
				Name:     "username",
				DataType: "varchar",
			},
			ObjectName: "accounts",
			SourceID:   "source-1",
			SourceName: "mysql-prod",
			SourceType: "mysql",
		},
	}, nil)

	results, err := service.SearchColumns(ctx, query, limit)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "user_id", results[0].Name)
	assert.Equal(t, "username", results[1].Name)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_SearchColumns_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	service, _ := setupMetadataService(t)

	_, err := service.SearchColumns(ctx, "", 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

func TestMetadataService_SearchColumns_WhitespaceQuery(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	_, err := service.SearchColumns(ctx, "   ", 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
	mockStore.AssertNotCalled(t, "SearchColumns", mock.Anything, mock.Anything, mock.Anything)
}

func TestMetadataService_SearchColumns_DefaultLimit(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	// 当limit为0时，应该使用默认值20
	mockStore.On("SearchColumns", ctx, "test", 20).Return([]*store.ColumnSearchRow{}, nil)

	results, err := service.SearchColumns(ctx, "test", 0)

	require.NoError(t, err)
	assert.Empty(t, results)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetColumnMappings(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetColumnMappings", ctx, columnID).Return([]*store.ColumnMappingRow{
		{
			ID:               "map-1",
			SourceColumnID:   columnID,
			TargetColumnID:   "col-2",
			MappingType:      "alias",
			Confidence:       0.95,
			TargetColumnName: "user_identifier",
			TargetObjectName: "accounts",
			TargetSourceName: "mysql-prod",
		},
	}, nil)

	mappings, err := service.GetColumnMappings(ctx, columnID)

	require.NoError(t, err)
	assert.Len(t, mappings, 1)
	assert.Equal(t, "map-1", mappings[0].ID)
	assert.Equal(t, "user_identifier", mappings[0].TargetColumn.Name)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_CreateColumnMapping(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	req := &model.ColumnMappingRequest{
		SourceColumnID: "col-1",
		TargetColumnID: "col-2",
		MappingType:    "transform",
		Confidence:     0.85,
	}

	mockStore.On("CreateColumnMapping", ctx, mock.AnythingOfType("*store.ColumnMappingCreate")).Return(nil)

	err := service.CreateColumnMapping(ctx, req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_CreateColumnMapping_DefaultConfidence(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	req := &model.ColumnMappingRequest{
		SourceColumnID: "col-1",
		TargetColumnID: "col-2",
		MappingType:    "alias",
		// Confidence未设置，应该默认为1.0
	}

	var capturedMapping *store.ColumnMappingCreate
	mockStore.On("CreateColumnMapping", ctx, mock.Anything).Run(func(args mock.Arguments) {
		capturedMapping = args.Get(1).(*store.ColumnMappingCreate)
	}).Return(nil)

	err := service.CreateColumnMapping(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1.0, capturedMapping.Confidence)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_DeleteColumnMapping(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	mappingID := "map-1"
	mockStore.On("DeleteColumnMapping", ctx, mappingID).Return(nil)

	err := service.DeleteColumnMapping(ctx, mappingID)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetLineage(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetLineageUpward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{
		{
			ID:               "edge-1",
			SourceID:         "col-0",
			TargetID:         columnID,
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "upstream_id",
			TargetName:       "user_id",
			SourceDataType:   "bigint",
			TargetDataType:   "int",
			SourceSourceName: "ods_mysql",
			TargetSourceName: "dwd_mysql",
		},
	}, nil)

	mockStore.On("GetLineageDownward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{
		{
			ID:               "edge-2",
			SourceID:         columnID,
			TargetID:         "col-2",
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "user_id",
			TargetName:       "dim_user_id",
			SourceDataType:   "int",
			TargetDataType:   "bigint",
			SourceSourceName: "dwd_mysql",
			TargetSourceName: "dm_mysql",
		},
	}, nil)

	lineage, err := service.GetLineage(ctx, columnID)

	require.NoError(t, err)
	assert.Equal(t, columnID, lineage.ColumnID)
	assert.Len(t, lineage.Upward, 1)
	assert.Len(t, lineage.Downward, 1)
	assert.Equal(t, "col-0", lineage.Upward[0].Source.ID)
	assert.Equal(t, "upstream_id", lineage.Upward[0].Source.Name)
	assert.Equal(t, "ods_mysql", lineage.Upward[0].Source.Source)
	assert.Equal(t, "col-2", lineage.Downward[0].Target.ID)
	assert.Equal(t, "dim_user_id", lineage.Downward[0].Target.Name)
	assert.Equal(t, "dm_mysql", lineage.Downward[0].Target.Source)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetImpactAnalysis(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetLineageDownward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{
		{
			ID:               "edge-1",
			SourceID:         columnID,
			TargetID:         "col-2",
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "user_id",
			TargetName:       "dim_user_id",
			TargetObjectName: "dim_users",
			TargetSourceName: "dm_mysql",
		},
		{
			ID:               "edge-2",
			SourceID:         columnID,
			TargetID:         "obj-1",
			SourceType:       "column",
			TargetType:       "object",
			SourceName:       "user_id",
			TargetName:       "ads_users",
			TargetObjectName: "ads_users",
			TargetSourceName: "ads_mysql",
		},
	}, nil)

	impact, err := service.GetImpactAnalysis(ctx, columnID)

	require.NoError(t, err)
	assert.Equal(t, columnID, impact.ColumnID)
	assert.Len(t, impact.ImpactObjects, 2)
	assert.Equal(t, 2, impact.TotalObjects)

	dimImpact := findImpactObjectByID(t, impact.ImpactObjects, "col-2")
	assert.Equal(t, "dim_user_id", dimImpact.Name)
	assert.Equal(t, "dim_users", dimImpact.ObjectName)
	assert.Equal(t, "dm_mysql", dimImpact.SourceName)
	assert.Equal(t, "user_id -> dim_user_id", dimImpact.ImpactPath)
	assert.Equal(t, 1, dimImpact.Distance)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetImpactAnalysis_MultiHopShortestPath(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetLineageDownward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{
		{
			ID:               "edge-1",
			SourceID:         columnID,
			TargetID:         "col-2",
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "user_id",
			TargetName:       "dim_user_id",
			TargetObjectName: "dim_users",
			TargetSourceName: "dm_mysql",
		},
		{
			ID:               "edge-2",
			SourceID:         "col-2",
			TargetID:         "obj-1",
			SourceType:       "column",
			TargetType:       "object",
			SourceName:       "dim_user_id",
			TargetName:       "ads_user_profile",
			TargetObjectName: "ads_user_profile",
			TargetSourceName: "ads_mysql",
		},
		{
			ID:               "edge-3",
			SourceID:         columnID,
			TargetID:         "obj-1",
			SourceType:       "column",
			TargetType:       "object",
			SourceName:       "user_id",
			TargetName:       "ads_user_profile",
			TargetObjectName: "ads_user_profile",
			TargetSourceName: "ads_mysql",
		},
		{
			ID:               "edge-4",
			SourceID:         "obj-1",
			TargetID:         "col-3",
			SourceType:       "object",
			TargetType:       "column",
			SourceName:       "ads_user_profile",
			TargetName:       "user_profile_key",
			TargetObjectName: "ads_user_profile",
			TargetSourceName: "ads_mysql",
		},
	}, nil)

	impact, err := service.GetImpactAnalysis(ctx, columnID)

	require.NoError(t, err)
	require.Len(t, impact.ImpactObjects, 3)

	dimImpact := findImpactObjectByID(t, impact.ImpactObjects, "col-2")
	assert.Equal(t, "dim_user_id", dimImpact.Name)
	assert.Equal(t, 1, dimImpact.Distance)
	assert.Equal(t, "user_id -> dim_user_id", dimImpact.ImpactPath)

	objectImpact := findImpactObjectByID(t, impact.ImpactObjects, "obj-1")
	assert.Equal(t, "ads_user_profile", objectImpact.Name)
	assert.Equal(t, 1, objectImpact.Distance)
	assert.Equal(t, "user_id -> ads_user_profile", objectImpact.ImpactPath)

	columnImpact := findImpactObjectByID(t, impact.ImpactObjects, "col-3")
	assert.Equal(t, "user_profile_key", columnImpact.Name)
	assert.Equal(t, 2, columnImpact.Distance)
	assert.Equal(
		t,
		"user_id -> ads_user_profile -> user_profile_key",
		columnImpact.ImpactPath,
	)

	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetImpactAnalysis_Empty(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetLineageDownward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{}, nil)

	impact, err := service.GetImpactAnalysis(ctx, columnID)

	require.NoError(t, err)
	assert.Equal(t, columnID, impact.ColumnID)
	assert.Empty(t, impact.ImpactObjects)
	assert.Equal(t, 0, impact.TotalObjects)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetImpactAnalysis_CycleKeepsShortestPath(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetLineageDownward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{
		{
			ID:               "edge-1",
			SourceID:         columnID,
			TargetID:         "col-2",
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "user_id",
			TargetName:       "dim_user_id",
			TargetObjectName: "dim_users",
			TargetSourceName: "dm_mysql",
		},
		{
			ID:               "edge-2",
			SourceID:         "col-2",
			TargetID:         "obj-1",
			SourceType:       "column",
			TargetType:       "object",
			SourceName:       "dim_user_id",
			TargetName:       "ads_users",
			TargetObjectName: "ads_users",
			TargetSourceName: "ads_mysql",
		},
		{
			ID:               "edge-3",
			SourceID:         "obj-1",
			TargetID:         columnID,
			SourceType:       "object",
			TargetType:       "column",
			SourceName:       "ads_users",
			TargetName:       "user_id",
			TargetObjectName: "users",
			TargetSourceName: "dwd_mysql",
		},
		{
			ID:               "edge-4",
			SourceID:         "col-2",
			TargetID:         "col-3",
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "dim_user_id",
			TargetName:       "user_profile_key",
			TargetObjectName: "dim_users",
			TargetSourceName: "dm_mysql",
		},
		{
			ID:               "edge-5",
			SourceID:         "obj-1",
			TargetID:         "col-3",
			SourceType:       "object",
			TargetType:       "column",
			SourceName:       "ads_users",
			TargetName:       "user_profile_key",
			TargetObjectName: "ads_users",
			TargetSourceName: "ads_mysql",
		},
	}, nil)

	impact, err := service.GetImpactAnalysis(ctx, columnID)

	require.NoError(t, err)
	require.Len(t, impact.ImpactObjects, 3)

	assert.Empty(t, filterImpactObjectsByID(impact.ImpactObjects, columnID))

	columnImpact := findImpactObjectByID(t, impact.ImpactObjects, "col-3")
	assert.Equal(t, 2, columnImpact.Distance)
	assert.Equal(t, "user_id -> dim_user_id -> user_profile_key", columnImpact.ImpactPath)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_GetImpactAnalysis_FallsBackToColumnIDWhenRootNameMissing(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	columnID := "col-1"

	mockStore.On("GetLineageDownward", ctx, columnID, 10).Return([]*store.LineageEdgeRow{
		{
			ID:               "edge-1",
			SourceID:         columnID,
			TargetID:         "col-2",
			SourceType:       "column",
			TargetType:       "column",
			SourceName:       "",
			TargetName:       "downstream_id",
			TargetObjectName: "dim_users",
			TargetSourceName: "dm_mysql",
		},
	}, nil)

	impact, err := service.GetImpactAnalysis(ctx, columnID)

	require.NoError(t, err)
	require.Len(t, impact.ImpactObjects, 1)
	assert.Equal(t, "col-1 -> downstream_id", impact.ImpactObjects[0].ImpactPath)
	mockStore.AssertExpectations(t)
}

func findImpactObjectByID(t *testing.T, impacts []model.ImpactObject, id string) model.ImpactObject {
	t.Helper()

	for _, impact := range impacts {
		if impact.ID == id {
			return impact
		}
	}

	t.Fatalf("impact object %s not found", id)
	return model.ImpactObject{}
}

func filterImpactObjectsByID(impacts []model.ImpactObject, id string) []model.ImpactObject {
	filtered := make([]model.ImpactObject, 0)
	for _, impact := range impacts {
		if impact.ID == id {
			filtered = append(filtered, impact)
		}
	}
	return filtered
}

func TestMetadataService_ListSchemaChanges(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	sourceID := "source-123"
	limit := 50

	mockStore.On("ListSchemaChangesBySource", ctx, sourceID, limit).Return([]*store.SchemaChangeRow{
		{
			ID:           "change-1",
			SourceID:     sourceID,
			ChangeType:   "add_column",
			ObjectType:   "column",
			ObjectName:   "email",
			DetectedAt:   "2024-01-01T00:00:00Z",
			Acknowledged: false,
		},
	}, nil)

	changes, err := service.ListSchemaChanges(ctx, sourceID, limit)

	require.NoError(t, err)
	assert.Len(t, changes, 1)
	assert.Equal(t, "add_column", changes[0].ChangeType)
	assert.Equal(t, "email", changes[0].ObjectName)
	mockStore.AssertExpectations(t)
}

func TestMetadataService_ListSchemaChanges_DefaultLimit(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupMetadataService(t)

	sourceID := "source-123"

	// 当limit为0时，应该使用默认值50
	mockStore.On("ListSchemaChangesBySource", ctx, sourceID, 50).Return([]*store.SchemaChangeRow{}, nil)

	changes, err := service.ListSchemaChanges(ctx, sourceID, 0)

	require.NoError(t, err)
	assert.Empty(t, changes)
	mockStore.AssertExpectations(t)
}
