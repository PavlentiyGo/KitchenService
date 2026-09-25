package http_middleware

import (
	core_response "avitoKitchen/internal/core/http/response"
	core_logger "avitoKitchen/internal/core/logger"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*core_logger.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return &core_logger.Logger{Logger: zap.New(core)}, logs
}

func makeMiddlewareRequest(handler http.Handler, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/middlewareTest", nil)
	for header, value := range headers {
		req.Header.Set(header, value)
	}

	logger, _ := newTestLogger()
	ctx := core_logger.CtxWithLogger(req.Context(), logger)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, req.WithContext(ctx))
	return responseRecorder
}

func TestRequestId_SuccessGeneratedId(t *testing.T) {
	var requestId string
	handler := RequestId()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusNoContent)
	}))

	responseRecorder := makeMiddlewareRequest(handler, nil)

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	parsedRequestId, err := uuid.Parse(requestId)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, parsedRequestId)
}

func TestRequestId_SuccessPreservedId(t *testing.T) {
	expectedRequestId := "request-id-from-client"
	var actualRequestId string
	handler := RequestId()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualRequestId = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusNoContent)
	}))

	responseRecorder := makeMiddlewareRequest(handler, map[string]string{
		"X-Request-ID": expectedRequestId,
	})

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	assert.Equal(t, expectedRequestId, actualRequestId)
}

func TestLogger_AddsLoggerToContext(t *testing.T) {
	logger, logs := newTestLogger()
	requestId := "test-request-id"
	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLogger := core_logger.FromContext(r.Context())
		requestLogger.Info("request handled")
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/middlewareTest", nil)
	req.Header.Set("X-Request-ID", requestId)
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, req)

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	require.Equal(t, 1, logs.Len())
	logContext := logs.All()[0].ContextMap()
	assert.Equal(t, requestId, logContext["RequestId"])
	assert.Equal(t, http.MethodPost, logContext["Method"])
	assert.Equal(t, "/middlewareTest", logContext["URL"])
}

func TestPanicRecoverer_ReturnsInternalError(t *testing.T) {
	handler := PanicRecoverer()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	}))

	responseRecorder := makeMiddlewareRequest(handler, nil)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, "got unexpected panic: test panic", response.Error.Message)
}

func TestSimpleAuth_Success(t *testing.T) {
	userId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	var userIdFromContext uuid.UUID
	handler := SimpleAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		userIdFromContext, err = GetUserIdFromCtx(r.Context())
		require.NoError(t, err)
		w.WriteHeader(http.StatusNoContent)
	}))

	responseRecorder := makeMiddlewareRequest(handler, map[string]string{
		"Authorization": userId.String(),
	})

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	assert.Equal(t, userId, userIdFromContext)
}

func TestSimpleAuth_MissingAuthorization(t *testing.T) {
	nextCalled := false
	handler := SimpleAuth()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	responseRecorder := makeMiddlewareRequest(handler, nil)

	assert.Equal(t, http.StatusUnauthorized, responseRecorder.Code)
	assert.False(t, nextCalled)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request", response.Error.Code)
	assert.Equal(t, "no user id in request", response.Error.Message)
}

func TestSimpleAuth_InvalidAuthorization(t *testing.T) {
	nextCalled := false
	handler := SimpleAuth()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	responseRecorder := makeMiddlewareRequest(handler, map[string]string{
		"Authorization": "not-a-uuid",
	})

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	assert.False(t, nextCalled)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request", response.Error.Code)
	assert.Equal(t, "invalid user id in request", response.Error.Message)
}

func TestSimpleCompanyAuth_Success(t *testing.T) {
	companyId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	var companyIdFromContext uuid.UUID
	handler := SimpleCompanyAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		companyIdFromContext, err = GetCompanyIdFromCtx(r.Context())
		require.NoError(t, err)
		w.WriteHeader(http.StatusNoContent)
	}))

	responseRecorder := makeMiddlewareRequest(handler, map[string]string{
		"Authorization": companyId.String(),
	})

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	assert.Equal(t, companyId, companyIdFromContext)
}

func TestSimpleCompanyAuth_MissingAuthorization(t *testing.T) {
	nextCalled := false
	handler := SimpleCompanyAuth()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	responseRecorder := makeMiddlewareRequest(handler, nil)

	assert.Equal(t, http.StatusUnauthorized, responseRecorder.Code)
	assert.False(t, nextCalled)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request", response.Error.Code)
	assert.Equal(t, "no company id in Authorization header: no user id in request", response.Error.Message)
}

func TestSimpleCompanyAuth_InvalidAuthorization(t *testing.T) {
	nextCalled := false
	handler := SimpleCompanyAuth()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	responseRecorder := makeMiddlewareRequest(handler, map[string]string{
		"Authorization": "not-a-uuid",
	})

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	assert.False(t, nextCalled)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request", response.Error.Code)
	assert.Equal(t, "invalid user id in request", response.Error.Message)
}
