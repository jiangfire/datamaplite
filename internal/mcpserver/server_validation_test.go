package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_MutationTools_RejectBlankIdentifiers(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	tests := []struct {
		name       string
		tool       string
		args       map[string]any
		wantErrMsg string
	}{
		{
			name:       "assign term column id",
			tool:       "assign_term_to_column",
			args:       map[string]any{"column_id": "   "},
			wantErrMsg: "column_id is required",
		},
		{
			name:       "assign tags column id",
			tool:       "assign_tags_to_column",
			args:       map[string]any{"column_id": " ", "tag_ids": []string{"tag-1"}},
			wantErrMsg: "column_id is required",
		},
		{
			name:       "create mapping source column id",
			tool:       "create_column_mapping",
			args:       map[string]any{"source_column_id": " ", "target_column_id": "col-2", "mapping_type": "alias"},
			wantErrMsg: "source_column_id is required",
		},
		{
			name:       "create mapping target column id",
			tool:       "create_column_mapping",
			args:       map[string]any{"source_column_id": "col-1", "target_column_id": " ", "mapping_type": "alias"},
			wantErrMsg: "target_column_id is required",
		},
		{
			name:       "create mapping type",
			tool:       "create_column_mapping",
			args:       map[string]any{"source_column_id": "col-1", "target_column_id": "col-2", "mapping_type": " "},
			wantErrMsg: "mapping_type is required",
		},
		{
			name:       "trigger source sync",
			tool:       "trigger_source_sync",
			args:       map[string]any{"source_id": "   "},
			wantErrMsg: "source_id is required",
		},
		{
			name:       "replay outbox",
			tool:       "replay_governance_outbox_event",
			args:       map[string]any{"outbox_id": "   "},
			wantErrMsg: "outbox_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tt.tool,
				Arguments: tt.args,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.IsError)
			require.Len(t, result.Content, 1)
			text, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok)
			assert.Contains(t, text.Text, tt.wantErrMsg)
		})
	}
}
