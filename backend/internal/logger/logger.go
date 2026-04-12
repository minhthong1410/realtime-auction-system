package logger

import (
	"go.uber.org/zap"
)

var L *zap.Logger

func Init(l *zap.Logger) {
	L = l
}

func Info(msg string, fields ...zap.Field) {
	L.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	L.Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	L.Warn(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	L.Debug(msg, fields...)
}
