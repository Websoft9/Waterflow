package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewAdminHandler(t *testing.T) {
	log := zap.NewNop()
	h := NewAdminHandler(log)

	assert.NotNil(t, h)
	assert.NotNil(t, h.logger)
}

func TestAdminHandler_GetLogLevel(t *testing.T) {
	// Initialize logger first
	err := logger.Init("info", "json")
	require.NoError(t, err)

	log := zap.NewNop()
	h := NewAdminHandler(log)

	req := httptest.NewRequest(http.MethodGet, "/admin/log-level", nil)
	w := httptest.NewRecorder()

	h.GetLogLevel(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response LogLevelResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Level)
}

func TestAdminHandler_SetLogLevel(t *testing.T) {
	// Initialize logger first
	err := logger.Init("info", "json")
	require.NoError(t, err)

	log := zap.NewNop()
	h := NewAdminHandler(log)

	t.Run("set valid log level", func(t *testing.T) {
		body := SetLogLevelRequest{Level: "debug"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/admin/log-level", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()

		h.SetLogLevel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response LogLevelResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "debug", response.Level)
		assert.Contains(t, response.Message, "successfully")
	})

	t.Run("set invalid log level", func(t *testing.T) {
		body := SetLogLevelRequest{Level: "invalid"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/admin/log-level", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()

		h.SetLogLevel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/admin/log-level", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		h.SetLogLevel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
