package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAlertService_ProcessSchemaChange_RejectsNilChange(t *testing.T) {
	service := NewAlertService(new(MockStore), zap.NewNop())

	err := service.ProcessSchemaChange(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema change is required")
}

func TestAlertService_ProcessSchemaChange_RejectsBlankSourceID(t *testing.T) {
	service := NewAlertService(new(MockStore), zap.NewNop())

	err := service.ProcessSchemaChange(context.Background(), &SchemaChangeInfo{
		ID:         "chg-1",
		SourceID:   "   ",
		ChangeType: "alter_column",
		ObjectType: "column",
		ObjectName: "users.email",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema change source_id is required")
}

func TestNormalizeAlertChangeTypes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "all", input: " all ", want: "all"},
		{name: "trim and dedupe", input: " drop_column, alter_column ,drop_column ", want: "drop_column,alter_column"},
		{name: "all wins", input: "alter_column, all, drop_column", want: "all"},
		{name: "empty after cleanup", input: " , , ", wantErr: "change types are required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAlertChangeTypes(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
