package slimcommon

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey int

const (
	loggerCtxKey ctxKey = iota
)

func InitContextWithLogger(ctx context.Context, log *zap.Logger) context.Context {
	// if the context already has a logger, do not overwrite it
	if ctx.Value(loggerCtxKey) != nil {
		return ctx
	}
	return context.WithValue(ctx, loggerCtxKey, log)
}

func LoggerFromContextOrDefault(ctx context.Context) *zap.Logger {
	v := ctx.Value(loggerCtxKey)
	if v == nil {
		return zap.Must(zap.NewDevelopment())
	}

	l, ok := v.(*zap.Logger)
	if !ok {
		return zap.Must(zap.NewDevelopment())
	}

	return l
}
