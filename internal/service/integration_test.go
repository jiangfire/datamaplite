package service

import (
	"context"
	"path/filepath"
	"testing"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMetadataServiceIntegration_TermAssignmentAndColumnDetail(t *testing.T) {
	ctx := context.Background()
	st := newSQLiteServiceIntegrationStore(t)
	defer st.Close()

	metadataService := NewMetadataService(st)
	termService := NewTermService(st)

	sourceID := mustCreateIntegrationSource(ctx, t, st, "mysql-prod")
	objectID := mustCreateIntegrationObject(ctx, t, st, sourceID, "users")
	columnID := mustCreateIntegrationColumn(ctx, t, st, objectID, "customer_id", "varchar", 1)

	createdTerm, err := termService.CreateTerm(ctx, &model.BusinessTermRequest{
		Name:     "Customer ID",
		Category: "identifier",
	})
	require.NoError(t, err)

	err = termService.AssignTermToColumn(ctx, columnID, &model.AssignTermRequest{
		TermID: &createdTerm.ID,
	})
	require.NoError(t, err)

	columnRow, err := st.GetColumn(ctx, columnID)
	require.NoError(t, err)
	require.NotNil(t, columnRow.TermID)
	assert.Equal(t, createdTerm.ID, *columnRow.TermID)

	detail, err := metadataService.GetColumnDetail(ctx, columnID)
	require.NoError(t, err)
	require.NotNil(t, detail.Term)
	assert.Equal(t, createdTerm.ID, detail.Term.ID)
	assert.Equal(t, "Customer ID", detail.Term.Name)

	err = termService.AssignTermToColumn(ctx, columnID, &model.AssignTermRequest{
		TermID: nil,
	})
	require.NoError(t, err)

	columnRow, err = st.GetColumn(ctx, columnID)
	require.NoError(t, err)
	assert.Nil(t, columnRow.TermID)

	detail, err = metadataService.GetColumnDetail(ctx, columnID)
	require.NoError(t, err)
	assert.Nil(t, detail.Term)
}

func TestMetadataServiceIntegration_MappingCreateDeleteAndDetailProjection(t *testing.T) {
	ctx := context.Background()
	st := newSQLiteServiceIntegrationStore(t)
	defer st.Close()

	metadataService := NewMetadataService(st)

	sourceID := mustCreateIntegrationSource(ctx, t, st, "ods-mysql")
	objectID := mustCreateIntegrationObject(ctx, t, st, sourceID, "users")
	columnID := mustCreateIntegrationColumn(ctx, t, st, objectID, "user_id", "bigint", 1)

	targetSourceID := mustCreateIntegrationSource(ctx, t, st, "dw-mysql")
	targetObjectID := mustCreateIntegrationObject(ctx, t, st, targetSourceID, "dim_users")
	targetColumnID := mustCreateIntegrationColumn(ctx, t, st, targetObjectID, "dim_user_id", "bigint", 1)

	err := metadataService.CreateColumnMapping(ctx, &model.ColumnMappingRequest{
		SourceColumnID: columnID,
		TargetColumnID: targetColumnID,
		MappingType:    "alias",
		Confidence:     0.87,
	})
	require.NoError(t, err)

	rawMappings, err := st.GetColumnMappings(ctx, columnID)
	require.NoError(t, err)
	require.Len(t, rawMappings, 1)
	assert.Equal(t, targetColumnID, rawMappings[0].TargetColumnID)

	mappings, err := metadataService.GetColumnMappings(ctx, columnID)
	require.NoError(t, err)
	require.Len(t, mappings, 1)
	assert.Equal(t, targetColumnID, mappings[0].TargetColumn.ID)
	assert.Equal(t, "dim_user_id", mappings[0].TargetColumn.Name)
	assert.Equal(t, "dim_users", mappings[0].TargetColumn.ObjectName)
	assert.Equal(t, "dw-mysql", mappings[0].TargetColumn.SourceName)
	assert.Equal(t, 0.87, mappings[0].Confidence)

	detail, err := metadataService.GetColumnDetail(ctx, columnID)
	require.NoError(t, err)
	require.Len(t, detail.MappedColumns, 1)
	assert.Equal(t, mappings[0].ID, detail.MappedColumns[0].ID)
	assert.Equal(t, "dim_user_id", detail.MappedColumns[0].Name)
	assert.Equal(t, "dim_users", detail.MappedColumns[0].ObjectName)
	assert.Equal(t, "dw-mysql", detail.MappedColumns[0].SourceName)
	assert.Equal(t, "alias", detail.MappedColumns[0].MappingType)

	err = metadataService.DeleteColumnMapping(ctx, mappings[0].ID)
	require.NoError(t, err)

	rawMappings, err = st.GetColumnMappings(ctx, columnID)
	require.NoError(t, err)
	assert.Empty(t, rawMappings)

	mappings, err = metadataService.GetColumnMappings(ctx, columnID)
	require.NoError(t, err)
	assert.Empty(t, mappings)

	detail, err = metadataService.GetColumnDetail(ctx, columnID)
	require.NoError(t, err)
	assert.Empty(t, detail.MappedColumns)
}

func TestMetadataServiceIntegration_ImpactAnalysisUsesRecursiveStoreResults(t *testing.T) {
	ctx := context.Background()
	st := newSQLiteServiceIntegrationStore(t)
	defer st.Close()

	metadataService := NewMetadataService(st)

	odsSourceID := mustCreateIntegrationSource(ctx, t, st, "ods-mysql")
	odsObjectID := mustCreateIntegrationObject(ctx, t, st, odsSourceID, "ods_users")
	rootColumnID := mustCreateIntegrationColumn(ctx, t, st, odsObjectID, "user_id", "bigint", 1)

	dwSourceID := mustCreateIntegrationSource(ctx, t, st, "dw-mysql")
	dwObjectID := mustCreateIntegrationObject(ctx, t, st, dwSourceID, "dim_users")
	dimColumnID := mustCreateIntegrationColumn(ctx, t, st, dwObjectID, "dim_user_id", "bigint", 1)

	adsSourceID := mustCreateIntegrationSource(ctx, t, st, "ads-mysql")
	adsObjectID := mustCreateIntegrationObject(ctx, t, st, adsSourceID, "ads_user_profile")
	adsColumnID := mustCreateIntegrationColumn(ctx, t, st, adsObjectID, "user_profile_key", "bigint", 1)

	require.NoError(t, st.CreateLineageEdge(ctx, &store.LineageEdgeCreate{
		SourceID:   rootColumnID,
		TargetID:   dimColumnID,
		SourceType: "column",
		TargetType: "column",
	}))
	require.NoError(t, st.CreateLineageEdge(ctx, &store.LineageEdgeCreate{
		SourceID:   dimColumnID,
		TargetID:   adsObjectID,
		SourceType: "column",
		TargetType: "object",
	}))
	require.NoError(t, st.CreateLineageEdge(ctx, &store.LineageEdgeCreate{
		SourceID:   rootColumnID,
		TargetID:   adsObjectID,
		SourceType: "column",
		TargetType: "object",
	}))
	require.NoError(t, st.CreateLineageEdge(ctx, &store.LineageEdgeCreate{
		SourceID:   adsObjectID,
		TargetID:   adsColumnID,
		SourceType: "object",
		TargetType: "column",
	}))

	impact, err := metadataService.GetImpactAnalysis(ctx, rootColumnID)
	require.NoError(t, err)
	require.Len(t, impact.ImpactObjects, 3)
	assert.Equal(t, 3, impact.TotalObjects)

	dimImpact := findImpactObjectByID(t, impact.ImpactObjects, dimColumnID)
	assert.Equal(t, "dim_user_id", dimImpact.Name)
	assert.Equal(t, "dim_users", dimImpact.ObjectName)
	assert.Equal(t, "dw-mysql", dimImpact.SourceName)
	assert.Equal(t, 1, dimImpact.Distance)
	assert.Equal(t, "user_id -> dim_user_id", dimImpact.ImpactPath)

	adsObjectImpact := findImpactObjectByID(t, impact.ImpactObjects, adsObjectID)
	assert.Equal(t, "ads_user_profile", adsObjectImpact.Name)
	assert.Equal(t, "ads_user_profile", adsObjectImpact.ObjectName)
	assert.Equal(t, "ads-mysql", adsObjectImpact.SourceName)
	assert.Equal(t, 1, adsObjectImpact.Distance)
	assert.Equal(t, "user_id -> ads_user_profile", adsObjectImpact.ImpactPath)

	adsColumnImpact := findImpactObjectByID(t, impact.ImpactObjects, adsColumnID)
	assert.Equal(t, "user_profile_key", adsColumnImpact.Name)
	assert.Equal(t, "ads_user_profile", adsColumnImpact.ObjectName)
	assert.Equal(t, "ads-mysql", adsColumnImpact.SourceName)
	assert.Equal(t, 2, adsColumnImpact.Distance)
	assert.Equal(t, "user_id -> ads_user_profile -> user_profile_key", adsColumnImpact.ImpactPath)
}

func newSQLiteServiceIntegrationStore(t *testing.T) *store.SQLiteStore {
	t.Helper()

	st, err := store.NewSQLiteStore(context.Background(), &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     filepath.Join(t.TempDir(), "service-integration.db"),
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)

	return st
}

func mustCreateIntegrationSource(ctx context.Context, t *testing.T, st store.Store, name string) string {
	t.Helper()

	id, err := st.CreateDataSource(ctx, &store.DataSourceCreate{
		Name:             name,
		Type:             "mysql",
		Host:             name + ".localhost",
		Port:             3306,
		Database:         name + "_db",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	return id
}

func mustCreateIntegrationObject(ctx context.Context, t *testing.T, st store.Store, sourceID string, name string) string {
	t.Helper()

	id, err := st.CreateSchemaObject(ctx, &store.SchemaObjectCreate{
		SourceID:    sourceID,
		Name:        name,
		Type:        "table",
		ColumnCount: 4,
	})
	require.NoError(t, err)

	return id
}

func mustCreateIntegrationColumn(ctx context.Context, t *testing.T, st store.Store, objectID string, name string, dataType string, ordinal int) string {
	t.Helper()

	err := st.CreateColumn(ctx, &store.ColumnCreate{
		ObjectID:        objectID,
		Name:            name,
		DataType:        dataType,
		FullDataType:    dataType,
		IsNullable:      true,
		OrdinalPosition: ordinal,
		Confidence:      1.0,
	})
	require.NoError(t, err)

	columns, err := st.ListColumnsByObject(ctx, objectID)
	require.NoError(t, err)
	for _, column := range columns {
		if column.Name == name {
			return column.ID
		}
	}

	t.Fatalf("column %s not found after creation", name)
	return ""
}
