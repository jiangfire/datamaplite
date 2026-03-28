package service

import (
	"context"
	"errors"
	"testing"

	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDDLService(t *testing.T) (*DDLService, *MockStore) {
	mockStore := new(MockStore)
	service := &DDLService{
		store:     mockStore,
		generator: NewDDLGenerator(),
	}
	return service, mockStore
}

func TestDDLService_GenerateDDL_MySQL(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupDDLService(t)

	objectID := "obj-123"

	mockStore.On("GetObjectWithColumns", ctx, objectID).Return(
		&store.SchemaObjectRow{
			ID:     objectID,
			Name:   "users",
			Type:   "table",
			Schema: strPtr("public"),
		},
		[]*store.ColumnRow{
			{
				ID:              "col-1",
				Name:            "id",
				DataType:        "integer",
				FullDataType:    "integer",
				IsNullable:      false,
				IsPrimaryKey:    true,
				OrdinalPosition: 1,
			},
			{
				ID:              "col-2",
				Name:            "name",
				DataType:        "character varying",
				FullDataType:    "character varying(255)",
				IsNullable:      true,
				OrdinalPosition: 2,
			},
			{
				ID:              "col-3",
				Name:            "created_at",
				DataType:        "timestamp without time zone",
				FullDataType:    "timestamp without time zone",
				IsNullable:      false,
				DefaultValue:    strPtr("now()"),
				OrdinalPosition: 3,
			},
		},
		nil,
	)

	resp, err := service.GenerateDDL(ctx, objectID, "mysql")

	require.NoError(t, err)
	assert.Equal(t, objectID, resp.ObjectID)
	assert.Contains(t, resp.SQL, "CREATE TABLE")
	assert.Contains(t, resp.SQL, "users")
	assert.Contains(t, resp.SQL, "id")
	assert.Contains(t, resp.SQL, "name")
	mockStore.AssertExpectations(t)
}

func TestDDLService_GenerateDDL_PostgreSQL(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupDDLService(t)

	objectID := "obj-123"

	mockStore.On("GetObjectWithColumns", ctx, objectID).Return(
		&store.SchemaObjectRow{
			ID:     objectID,
			Name:   "users",
			Type:   "table",
			Schema: strPtr("public"),
		},
		[]*store.ColumnRow{
			{
				ID:              "col-1",
				Name:            "id",
				DataType:        "integer",
				FullDataType:    "integer",
				IsNullable:      false,
				IsPrimaryKey:    true,
				OrdinalPosition: 1,
			},
		},
		nil,
	)

	resp, err := service.GenerateDDL(ctx, objectID, "postgres")

	require.NoError(t, err)
	assert.Equal(t, objectID, resp.ObjectID)
	assert.Contains(t, resp.SQL, "CREATE TABLE")
	mockStore.AssertExpectations(t)
}

func TestDDLService_GenerateDDL_ObjectNotFound(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupDDLService(t)

	objectID := "non-existent"

	mockStore.On("GetObjectWithColumns", ctx, objectID).Return(nil, nil, errors.New("object not found"))

	_, err := service.GenerateDDL(ctx, objectID, "mysql")

	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDDLService_GenerateDDL_EmptyColumns(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupDDLService(t)

	objectID := "obj-123"

	mockStore.On("GetObjectWithColumns", ctx, objectID).Return(
		&store.SchemaObjectRow{
			ID:   objectID,
			Name: "empty_table",
			Type: "table",
		},
		[]*store.ColumnRow{},
		nil,
	)

	_, err := service.GenerateDDL(ctx, objectID, "mysql")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no columns")
	mockStore.AssertExpectations(t)
}
