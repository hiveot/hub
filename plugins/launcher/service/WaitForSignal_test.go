package service_test

import (
	"github.com/hiveot/hub/plugins/launcher/service"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitForSignal(t *testing.T) {
	m := sync.Mutex{}
	var waitCompleted = false
	go func() {
		service.WaitForSignal()
		m.Lock()
		waitCompleted = true
		m.Unlock()
	}()
	pid := os.Getpid()
	time.Sleep(time.Second)

	// signal.Notify()
	syscall.Kill(pid, syscall.SIGINT)
	time.Sleep(time.Second)
	m.Lock()
	defer m.Unlock()
	assert.True(t, waitCompleted)
}
