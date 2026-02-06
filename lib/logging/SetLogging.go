// Package logging with logging configuration
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmittmann/tint"
)

// SetLogging initializes the global logger
func SetLogging(levelName string, logFilename string) *slog.Logger {
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
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// include sourcefile
			if attr.Key == slog.SourceKey {
				source := attr.Value.Any().(*slog.Source)
				source.File = filepath.Base(source.File)
				//sourcePath := fmt.Sprintf("%s:%d %s", src.File, src.Line, src.Function)
				//return slog.String(slog.SourceKey, sourcePath)
			}
			return attr
		},
	}

	// if a file is provided then also log to file
	var logWriter io.Writer
	logWriter = os.Stdout
	if logFilename != "" {
		logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			// log to both stdout and to file
			logWriter = io.MultiWriter(os.Stdout, logFile)
		}
	}
	handler := tint.NewHandler(logWriter, opts)
	//opts := &slog.HandlerOptions{
	//	AddSource: true,
	//	Level:     logLevel,
	//}
	//handler := slog.NewTextHandler(os.Stdout, opts)

	logger := slog.New(handler)
	slog.SetDefault(logger)
	//logLogger := slog.NewLogLogger(handler, logLevel)
	//_ = http.Server{ErrorLog: logLogger}
	return logger
}

// NewFileLogger returns a new file logger that forks to stdout
// This returns the logger and the file.
func NewFileLogger(logfileName string, asJSON bool) (*slog.Logger, *os.File) {
	// setup request logging
	if logfileName != "" {
		var logWriter io.Writer
		var logger *slog.Logger

		logFile, err := os.OpenFile(logfileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

		// log to both stdout and to file
		logWriter = io.MultiWriter(os.Stdout, logFile)
		if err == nil {
			// file logging in JSON
			if asJSON {
				logHandler := slog.NewJSONHandler(logWriter, nil)
				logger = slog.New(logHandler)
			} else {
				// or just pretty print
				logHandler := tint.NewHandler(logWriter, &tint.Options{
					AddSource:  true,
					Level:      slog.LevelInfo,
					TimeFormat: "Jan _2 15:04:05.0000"},
				)
				logger = slog.New(logHandler)
			}
		}
		return logger, logFile
	}
	return slog.Default(), nil
}
