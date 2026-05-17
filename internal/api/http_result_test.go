package api

import (
	"encoding/json"
	"testing"

	responsepkg "github.com/jiangfire/datamaplite/pkg/response"
	"github.com/stretchr/testify/require"
)

const successCode = responsepkg.SuccessCode

func decodeHTTPResult(t *testing.T, body []byte) responsepkg.HttpResult {
	t.Helper()

	var resp responsepkg.HttpResult
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}
