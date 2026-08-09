package logger

import (
	"go.uber.org/zap"
)

func NewLogger() (*zap.Logger, error) {
	return NewLoggerWithConfig(zap.NewDevelopmentConfig())
}

func NewLoggerWithConfig(cfg zap.Config) (*zap.Logger, error) {
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}
