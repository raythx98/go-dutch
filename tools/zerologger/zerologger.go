package zerologger

import (
	"context"
	"os"

	"github.com/raythx98/gohelpme/tool/logger"
	"github.com/raythx98/gohelpme/tool/reqctx"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	log *zerolog.Logger
}

func New(isDebugMode bool) *Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Logger.Hook(TracingHook{})
	if isDebugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	return &Logger{
		log: &log.Logger,
	}
}

type TracingHook struct{}

func (h TracingHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	reqCtx := reqctx.GetValue(e.GetCtx())
	if reqCtx == nil {
		return
	}
	e.Any("context", reqCtx)
}

func (l *Logger) GetInstance() interface{} {
	return l.log
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	AppendFieldsToEvent(l.log.Debug().Ctx(ctx), fields...).Msg(msg)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...logger.Field) {
	AppendFieldsToEvent(l.log.Info().Ctx(ctx), fields...).Msg(msg)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	AppendFieldsToEvent(l.log.Warn().Ctx(ctx), fields...).Msg(msg)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...logger.Field) {
	AppendFieldsToEvent(l.log.Error().Stack().Ctx(ctx), fields...).Msg(msg)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...logger.Field) {
	AppendFieldsToEvent(l.log.Fatal().Stack().Ctx(ctx), fields...).Msg(msg)
}

func (l *Logger) Panic(ctx context.Context, msg string, fields ...logger.Field) {
	AppendFieldsToEvent(l.log.Panic().Stack().Ctx(ctx), fields...).Msg(msg)
}

func AppendFieldsToEvent(e *zerolog.Event, fields ...logger.Field) *zerolog.Event {
	for _, o := range fields {
		for k, v := range o() {
			if err, ok := v.(error); ok {
				e = e.Err(err)
			} else if errs, ok := v.([]error); ok {
				e = e.Errs("errors", errs)
			} else {
				e = e.Interface(k, v)
			}
		}
	}
	return e
}
