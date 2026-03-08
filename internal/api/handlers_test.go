package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// respondJSON helper
// ---------------------------------------------------------------------------

func TestRespondJSON_StatusAndContentType(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestRespondJSON_Body(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusCreated, map[string]int{"count": 42})

	var result map[string]int
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, 42, result["count"])
}

func TestRespondJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "null")
}

func TestRespondJSON_EmptySlice(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, []string{})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "[]")
}

func TestRespondJSON_CustomStatus(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusNoContent, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ---------------------------------------------------------------------------
// HealthCheck with nil dependencies
// ---------------------------------------------------------------------------

func TestHealthCheck_AllNil(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "degraded", result["status"])

	services := result["services"].(map[string]interface{})
	assert.Equal(t, "not configured", services["postgres"])
	assert.Equal(t, "not configured", services["redis"])
	assert.Equal(t, "not configured", services["kafka"])
}

func TestHealthCheck_HasTimestamp(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	assert.Contains(t, result, "timestamp")
}

// ---------------------------------------------------------------------------
// AddStock validation
// ---------------------------------------------------------------------------

func TestAddStock_EmptyBody(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/stocks", strings.NewReader(""))
	w := httptest.NewRecorder()

	h.AddStock(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddStock_EmptySymbol(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/stocks", strings.NewReader(`{"symbol":""}`))
	w := httptest.NewRecorder()

	h.AddStock(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "symbol is required")
}

func TestAddStock_InvalidJSON(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/stocks", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()

	h.AddStock(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

// ---------------------------------------------------------------------------
// CreateFeedback validation
// ---------------------------------------------------------------------------

func TestCreateFeedback_InvalidJSON(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/feedback", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	h.CreateFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateFeedback_MissingSymbol(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/feedback",
		strings.NewReader(`{"signal":"BUY","action":"traded"}`))
	w := httptest.NewRecorder()

	h.CreateFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "symbol, signal, and action are required")
}

func TestCreateFeedback_MissingSignal(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/feedback",
		strings.NewReader(`{"symbol":"AAPL","action":"traded"}`))
	w := httptest.NewRecorder()

	h.CreateFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateFeedback_MissingAction(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/feedback",
		strings.NewReader(`{"symbol":"AAPL","signal":"BUY"}`))
	w := httptest.NewRecorder()

	h.CreateFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetFeedback query param validation
// ---------------------------------------------------------------------------

func TestGetFeedback_InvalidSinceDays(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/feedback?since_days=abc", nil)
	w := httptest.NewRecorder()

	h.GetFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid since_days")
}

func TestGetFeedback_NegativeSinceDays(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/feedback?since_days=-5", nil)
	w := httptest.NewRecorder()

	h.GetFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetFeedback_InvalidLimit(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/feedback?limit=xyz", nil)
	w := httptest.NewRecorder()

	h.GetFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid limit")
}

func TestGetFeedback_NegativeLimit(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/feedback?limit=-1", nil)
	w := httptest.NewRecorder()

	h.GetFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Route setup
// ---------------------------------------------------------------------------

func TestSetupRoutes_AllRoutesRegistered(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	router := SetupRoutes(h)

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/api/v1/stocks"},
		{"POST", "/api/v1/stocks"},
		{"GET", "/api/v1/stocks/AAPL"},
		{"DELETE", "/api/v1/stocks/AAPL"},
		{"POST", "/api/v1/feedback"},
		{"GET", "/api/v1/feedback"},
		{"GET", "/api/v1/feedback/summary"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			match := &mux.RouteMatch{}
			assert.True(t, router.Match(req, match),
				"route not registered: %s %s", tc.method, tc.path)
		})
	}
}

func TestSetupRoutes_WrongMethod(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	router := SetupRoutes(h)

	// POST to /health should not match (only GET)
	req := httptest.NewRequest("POST", "/health", nil)
	match := &mux.RouteMatch{}
	matched := router.Match(req, match)
	// mux returns MethodNotAllowed (match is true but handler returns 405)
	if matched {
		assert.Equal(t, http.StatusMethodNotAllowed, match.MatchErr)
	}
}
