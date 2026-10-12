package http_middleware

import (
	"context"

	"fmt"
	core_errors "foodDelivery/internal/core/errors"
	core_response "foodDelivery/internal/core/http/response"
	core_logger "foodDelivery/internal/core/logger"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestId() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestId := r.Header.Get("X-Request-ID")
			if requestId == "" {
				requestId = uuid.NewString()
			}
			r.Header.Set("X-Request-ID", requestId)
			next.ServeHTTP(w, r)
		})
	}
}
func Logger(logger *core_logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			loggerWith := logger.With(
				zap.String("RequestId", r.Header.Get("X-Request-ID")),
				zap.String("Method", r.Method),
				zap.String("URL", r.URL.String()),
			)

			ctx := core_logger.CtxWithLogger(r.Context(), loggerWith)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PanicRecoverer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				p := recover()
				if p != nil {
					writer := core_response.WrapWriter(w, r.Context())
					writer.ErrorResponse(fmt.Errorf("got unexpected panic: %v", p))
					return
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func Trace() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			writer := core_response.WrapWriter(w, r.Context())

			timeNow := time.Now()
			writer.Logger.Debug("starting request")
			next.ServeHTTP(writer, r)
			writer.Logger.Debug(
				"end request",
				zap.Duration("requestDuration", time.Since(timeNow)),
				zap.Int("statusCode", writer.StatusCode),
			)
		})
	}
}

func SimpleAuth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := core_response.WrapWriter(w, r.Context())
			userId := r.Header.Get("Authorization")
			if userId == "" {
				writer.ErrorResponse(core_errors.ErrNotAuthorized)
				return
			}
			_, err := uuid.Parse(userId)
			if err != nil {
				writer.ErrorResponse(core_errors.ErrInvalidAuthorization)
				return
			}

			ctx := context.WithValue(r.Context(), userIdKey{}, userId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
func SimpleCompanyAuth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := core_response.WrapWriter(w, r.Context())
			companyId := r.Header.Get("Authorization")
			if companyId == "" {
				writer.ErrorResponse(fmt.Errorf("no company id in Authorization header: %w", core_errors.ErrNotAuthorized))
				return
			}
			_, err := uuid.Parse(companyId)
			if err != nil {
				writer.ErrorResponse(core_errors.ErrInvalidAuthorization)
				return
			}

			ctx := context.WithValue(r.Context(), companyIdKey{}, companyId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
