package core_logger

import (
	"context"
	"fmt"
	core_config "foodDelivery/internal/core/config"
	"os"
	"path"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
	file *os.File
}

type ctxLogger struct{}

func CtxWithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, ctxLogger{}, logger)
}

func FromContext(ctx context.Context) *Logger {
	val, ok := ctx.Value(ctxLogger{}).(*Logger)
	if !ok {
		panic("failed to get logger from context")
	}
	return val
}

func NewLogger(config *core_config.LoggerConfig) (*Logger, error) {
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to get logger level: %w", err)
	}
	lvl := zap.NewAtomicLevelAt(level)

	err = os.MkdirAll(config.Folder, 0750)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder for logger: %w", err)
	}
	timestamp := time.Now().UTC().Format(time.StampMicro)

	logFilePath := path.Join(config.Folder, fmt.Sprintf("%s.log", timestamp))

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create logFile: %v: %w", logFilePath, err)
	}
	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.StampMicro)

	zapEncoder := zapcore.NewConsoleEncoder(zapConfig)

	cores := zapcore.NewTee(
		zapcore.NewCore(zapEncoder, zapcore.AddSync(os.Stdout), lvl),
		zapcore.NewCore(zapEncoder, zapcore.AddSync(file), lvl),
	)

	logger := zap.New(cores, zap.AddCaller())

	return &Logger{
		file:   file,
		Logger: logger,
	}, nil
}

func (l *Logger) Close() error {
	err := l.file.Close()
	if err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}
	return nil
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		file:   l.file,
		Logger: l.Logger.With(fields...),
	}
}
