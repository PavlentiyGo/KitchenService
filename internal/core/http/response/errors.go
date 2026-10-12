package core_response

import (
	core_errors "foodDelivery/internal/core/errors"
	"net/http"
)

type errorValue struct {
	mapError   error
	statusCode int
	error      string
	logLevel   string
}
type ErrorResponse struct {
	Error Error `json:"error"`
}
type Error struct {
	Code    string `json:"code" example:"invalid request"`
	Message string `json:"message" example:"invalid companyId path parameter"`
}

var errorSlice = []errorValue{
	{mapError: core_errors.ErrInvalidRequest, statusCode: http.StatusBadRequest, logLevel: "WARN", error: core_errors.ErrInvalidRequest.Error()},
	{mapError: core_errors.ErrNotAuthorized, statusCode: http.StatusUnauthorized, logLevel: "WARN", error: core_errors.ErrInvalidRequest.Error()},
	{mapError: core_errors.ErrInvalidAuthorization, statusCode: http.StatusBadRequest, logLevel: "WARN", error: core_errors.ErrInvalidRequest.Error()},
	{mapError: core_errors.ErrForbidden, statusCode: http.StatusForbidden, logLevel: "WARN", error: core_errors.ErrForbidden.Error()},
	{mapError: core_errors.ErrNotFound, statusCode: http.StatusNotFound, logLevel: "WARN", error: core_errors.ErrNotFound.Error()},
	{mapError: core_errors.ErrCompanyInactive, statusCode: http.StatusConflict, logLevel: "WARN", error: "conflict"},
	{mapError: core_errors.ErrProductUnavailable, statusCode: http.StatusConflict, logLevel: "WARN", error: "conflict"},
	{mapError: core_errors.ErrInsufficientStock, statusCode: http.StatusConflict, logLevel: "WARN", error: "conflict"},
	{mapError: core_errors.ErrConflict, statusCode: http.StatusConflict, logLevel: "WARN", error: "conflict"},
}
