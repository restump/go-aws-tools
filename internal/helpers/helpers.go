package helpers

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/rs/zerolog"
)

func NewLog() *zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	level := LookupEnvWithDefault("LOG_LEVEL", "INFO")
	if strings.EqualFold(level, "TRACE") {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
	if strings.EqualFold(level, "DEBUG") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if strings.EqualFold(level, "WARN") {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
	if strings.EqualFold(level, "ERROR") {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
	if strings.EqualFold(level, "FATAL") {
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	}
	if strings.EqualFold(level, "PANIC") {
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	}
	llog := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &llog
}

func NewLogWithContext(ctx context.Context) *zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	level := LookupEnvWithDefault("LOG_LEVEL", "INFO")
	if strings.EqualFold(level, "TRACE") {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
	if strings.EqualFold(level, "DEBUG") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if strings.EqualFold(level, "WARN") {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
	if strings.EqualFold(level, "ERROR") {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
	if strings.EqualFold(level, "FATAL") {
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	}
	if strings.EqualFold(level, "PANIC") {
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	}

	lc, _ := lambdacontext.FromContext(ctx)
	llog := zerolog.New(os.Stdout).With().Str("requestId", lc.AwsRequestID).Timestamp().Logger()
	return &llog
}

func LookupEnvWithDefault(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

// Return pointer to value using GO v1.18 generics
func ToPtr[T any](v T) *T {
	return &v
}
