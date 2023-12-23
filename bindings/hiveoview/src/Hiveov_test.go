package src

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/service"
	"testing"
	"time"
)

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")

	svc := service.NewHiveovService(8080, true)

	svc.Start()
	time.Sleep(time.Second * 3)
	svc.Stop()
}
