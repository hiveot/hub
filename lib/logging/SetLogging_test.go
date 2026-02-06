package logging_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/hiveot/hub/lib/logging"
)

func TestLogging(t *testing.T) {
	//wd, _ := os.Getwd()
	//logFile := path.Join(wd, "../../test/logs/TestLogging.log")
	logFile := ""

	os.Remove(logFile)
	logging.SetLogging("info", logFile)
	slog.Info("Hello info")
	logging.SetLogging("debug", logFile)
	slog.Debug("Hello debug")
	logging.SetLogging("warn", logFile)
	slog.Warn("Hello warn")
	logging.SetLogging("error", logFile)
	slog.Error("Hello error")
	//assert.FileExists(t, logFile)
	//os.Remove(logFile)
}
