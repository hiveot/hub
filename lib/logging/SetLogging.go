// Package logging with logging configuration
package logging

import (
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"strings"
)

// SetLogging initializes the global logger
func SetLogging(levelName string, logFile string) *slog.Logger {
	//// init logging
	//const TimeFormat = "Jan _2 15:04:05.0000"
	//zlwr := zerolog.ConsoleWriter{
	//	Out:        os.Stdout,
	//	TimeFormat: TimeFormat,
	//	FormatCaller: func(i interface{}) string {
	//		return filepath.Base(fmt.Sprintf("%s", i))
	//	},
	//}
	//zerolog.TimeFieldFormat = "Jan _2 15:04:05.0000"
	//log.Logger = zerolog.New(zlwr).With().Timestamp().Caller().Logger()
	logLevel := slog.LevelInfo
	if levelName == "debug" {
		logLevel = slog.LevelDebug
	} else if strings.HasPrefix(levelName, "warn") {
		logLevel = slog.LevelWarn
	} else if levelName == "error" {
		logLevel = slog.LevelError
	}
	opts := &tint.Options{
		AddSource:  true,
		Level:      logLevel,
		TimeFormat: "Jan _2 15:04:05.0000",
	}
	handler := tint.NewHandler(os.Stdout, opts)
	//opts := &slog.HandlerOptions{
	//	AddSource: true,
	//	Level:     logLevel,
	//}
	//handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
