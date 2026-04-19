package logging

import (
	"fmt"
	"strings"

	"goserver/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(cfg config.LoggingConfig) (*zap.Logger, error) {
	zapConfig := zap.NewProductionConfig()
	if cfg.Development {
		zapConfig = zap.NewDevelopmentConfig()
	}

	encoding := strings.TrimSpace(cfg.Encoding)
	if encoding != "" {
		zapConfig.Encoding = encoding
	}

	levelText := strings.TrimSpace(cfg.Level)
	if levelText == "" {
		levelText = "info"
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL %q: %w", levelText, err)
	}

	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return zapConfig.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func NewBootstrap() *zap.Logger {
	logger, err := zap.NewProduction(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return zap.NewNop()
	}

	return logger
}

func Sync(logger *zap.Logger) {
	if logger == nil {
		return
	}

	_ = logger.Sync()
}
