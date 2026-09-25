package core_response

import (
	core_logger "avitoKitchen/internal/core/logger"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

type Writer struct {
	http.ResponseWriter
	StatusCode int
	Logger     *core_logger.Logger
}

func WrapWriter(w http.ResponseWriter, ctx context.Context) *Writer {
	if writer, ok := w.(*Writer); ok {
		return writer
	}
	return &Writer{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
		Logger:         core_logger.FromContext(ctx),
	}
}
func (w *Writer) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
func (w *Writer) ErrorResponse(err error) {
	var val errorValue
	ok := false
	for i := 0; i < len(errorSlice); i++ {
		if errors.Is(err, errorSlice[i].mapError) {
			val = errorSlice[i]
			ok = true
			break
		}
	}

	if !ok {
		w.Logger.Error("got INTERNAL error", zap.Error(err))
		w.writeErrorJson(http.StatusInternalServerError, "", err.Error())
		return
	}
	switch val.logLevel {
	case "DEBUG":
		w.Logger.Debug("got error", zap.Error(err))
	case "WARN":
		w.Logger.Warn("got error", zap.Error(err))
	case "ERROR":
		w.Logger.Error("got INTERNAL error", zap.Error(err))
	}
	w.writeErrorJson(val.statusCode, val.error, err.Error())
}
func (w *Writer) writeErrorJson(statusCode int, error string, message string) {

	response := ErrorResponse{
		Error: Error{
			Code:    error,
			Message: message,
		},
	}
	w.WriteJson(statusCode, response)
}
func (w *Writer) WriteJson(statusCode int, body any) {
	bytes, err := json.MarshalIndent(body, "", "    ")
	if err != nil {
		w.Logger.Error("failed to marshal data into json", zap.Error(err))
		return
	}
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bytes)
	if err != nil {
		w.Logger.Error("failed to write data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
