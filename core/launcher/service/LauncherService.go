package service

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/launcher"
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/lib/utils"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/lo"
	"github.com/struCoder/pidusage"
)

// LauncherService manages starting and stopping of services
// This implements the ILauncher interface
type LauncherService struct {
	// service configuration
	cfg config.LauncherConfig
	f   utils.AppDirs

	// map of service name to running status
	services map[string]*launcher.ServiceInfo
	// list of started commands in startup order
	cmds []*exec.Cmd

	// messaging client for receiving requests
	hc hubclient.IHubClient
	// subscription to receive requests
	mngSub hubclient.ISubscription

	// mutex to keep things safe
	mux sync.Mutex
	// watch plugin folders for updates
	serviceWatcher *fsnotify.Watcher
	// service is running
	isRunning atomic.Bool
	// closing channel
	done chan bool
}

// Add newly discovered executable services
// If the service is already know, only update its size and timestamp
func (svc *LauncherService) findServices(folder string) error {
	count := 0
	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// ignore directories and non executable files
		fileInfo, _ := entry.Info()
		size := fileInfo.Size()
		fileMode := fileInfo.Mode()
		isExecutable := fileMode&0100 != 0
		isFile := !entry.IsDir()
		if isFile && isExecutable && size > 0 {
			count++
			serviceInfo, found := svc.services[entry.Name()]
			if !found {
				serviceInfo = &launcher.ServiceInfo{
					Name:    entry.Name(),
					Path:    path.Join(folder, entry.Name()),
					Uptime:  0,
					Running: false,
				}
				svc.services[serviceInfo.Name] = serviceInfo
			}
			serviceInfo.ModifiedTime = fileInfo.ModTime().Format(time.RFC3339)
			serviceInfo.Size = size
		}
	}
	slog.Info("found services", "count", count, "directory", folder)
	return nil
}

// List all available or just the running services and their status
// This returns the list of services sorted by name
func (svc *LauncherService) List(onlyRunning bool) ([]launcher.ServiceInfo, error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	// get the keys of the services to include and sort them
	keys := make([]string, 0, len(svc.services))
	for key, val := range svc.services {
		if !onlyRunning || val.Running {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	res := make([]launcher.ServiceInfo, 0, len(keys))
	for _, key := range keys {
		svcInfo := svc.services[key]
		svc.updateStatus(svcInfo)
		res = append(res, *svcInfo)
	}
	return res, nil
}

// ScanServices scans the plugin folder for changes and updates the services list
func (svc *LauncherService) ScanServices() error {
	err := svc.findServices(svc.f.Plugins)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
func (svc *LauncherService) StartService(name string) (info launcher.ServiceInfo, err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	// step 1: pre-checks
	serviceInfo, found := svc.services[name]
	if !found {
		info.Status = fmt.Sprintf("service '%s' not found", name)
		slog.Error(info.Status)
		return info, errors.New(info.Status)
	}
	if serviceInfo.Running {
		err = fmt.Errorf("starting service '%s' failed. The service is already running", name)
		slog.Error(err.Error())
		return *serviceInfo, err
	}
	// don't start twice
	for _, cmd := range svc.cmds {
		if cmd.Path == serviceInfo.Path {
			err = fmt.Errorf("process for service '%s' already exists using PID %d",
				serviceInfo.Name, cmd.Process.Pid)
			slog.Error(err.Error())
			return *serviceInfo, err
		}
	}

	// step 2: create the command to start the service ... but wait for step 3
	svcCmd := exec.Command(serviceInfo.Path)

	// step3: setup logging before starting service
	slog.Info("Starting service", "name", name)

	if svc.cfg.LogPlugins {
		// inspired by https://gist.github.com/jerblack/4b98ba48ed3fb1d9f7544d2b1a1be287
		logfile := path.Join(svc.f.Logs, name+".log")
		fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			if svc.cfg.AttachStderr {
				// log stderr to launcher stderr and to file
				multiwriter := io.MultiWriter(os.Stderr, fp)
				svcCmd.Stderr = multiwriter
			} else {
				// just log stderr to file
				svcCmd.Stderr = fp
			}
			if svc.cfg.AttachStdout {
				// log stdout to launcher stdout and to file
				multiwriter := io.MultiWriter(os.Stdout, fp)
				svcCmd.Stdout = multiwriter
			} else {
				// just log stdout to file
				svcCmd.Stdout = fp
			}
		} else {
			slog.Error("creating logfile failed", "err", err, "file", logfile)
		}
	} else {
		if svc.cfg.AttachStderr {
			svcCmd.Stderr = os.Stderr
		}
		if svc.cfg.AttachStdout {
			svcCmd.Stdout = os.Stdout
		}
	}
	// step 4: start the command and setup serviceInfo
	err = svcCmd.Start()
	if err != nil {
		serviceInfo.Status = fmt.Sprintf("failed starting '%s': %s", name, err.Error())
		err = errors.New(serviceInfo.Status)
		slog.Error(err.Error())
		return *serviceInfo, err
	}
	svc.cmds = append(svc.cmds, svcCmd)
	//slog.Warning("Service has started", "serviceName",name)

	serviceInfo.StartTime = time.Now().Format(time.RFC3339)
	serviceInfo.PID = svcCmd.Process.Pid
	serviceInfo.Status = ""
	serviceInfo.StartCount++
	serviceInfo.Running = true

	// step 5: handle command termination and cleanup
	go func() {
		// cleanup after the process ends
		status := svcCmd.Wait()
		_ = status
		svc.mux.Lock()
		defer svc.mux.Unlock()

		serviceInfo.StopTime = time.Now().Format(time.RFC3339)
		serviceInfo.Running = false
		// processState holds exit info
		procState := svcCmd.ProcessState

		if status != nil {
			serviceInfo.Status = fmt.Sprintf("Service '%s' has stopped with: %s", name, status.Error())
		} else if procState != nil {
			serviceInfo.Status = fmt.Sprintf("Service '%s' has stopped with exit code %d: sys='%v'", name, procState.ExitCode(), procState.Sys())
		} else {
			serviceInfo.Status = fmt.Sprintf("Service '%s' has stopped without info", name)
		}
		slog.Warn(serviceInfo.Status)
		svc.updateStatus(serviceInfo)
		// find the service to delete
		i := lo.IndexOf(svc.cmds, svcCmd)
		//lo.Delete(svc.cmds, i)  - why doesn't this exist?
		svc.cmds = append(svc.cmds[:i], svc.cmds[i+1:]...) // this is so daft!
	}()

	// Give it some time to get up and running in case it is needed as a dependency
	// TODO: wait for channel
	time.Sleep(time.Millisecond * 100)

	// last, update the CPU and memory status
	svc.updateStatus(serviceInfo)
	return *serviceInfo, err
}

// StartAll starts all enabled services
func (svc *LauncherService) StartAll() (err error) {
	slog.Info("Starting all enabled services")

	// ensure they start in order
	for _, svcName := range svc.cfg.Autostart {
		svcInfo := svc.services[svcName]
		if svcInfo != nil && svcInfo.Running {
			// skip
		} else {
			_, err2 := svc.StartService(svcName)
			if err2 != nil {
				err = err2
			}
		}
	}
	// start the remaining services
	for svcName, svcInfo := range svc.services {
		if !svcInfo.Running {
			_, err2 := svc.StartService(svcName)
			if err2 != nil {
				err = err2
			}
		}
	}
	return err
}

func (svc *LauncherService) StopService(name string) (info launcher.ServiceInfo, err error) {
	slog.Info("Stopping service", "name", name)

	serviceInfo, found := svc.services[name]
	if !found {
		info.Status = fmt.Sprintf("service '%s' not found", name)
		err = errors.New(info.Status)
		slog.Error("service not found", "name", name)
		return info, err
	}
	err = Stop(serviceInfo.Name, serviceInfo.PID)
	if err == nil {
		svc.mux.Lock()
		defer svc.mux.Unlock()
		serviceInfo.Running = false
		serviceInfo.Status = "stopped by user"
	}
	return *serviceInfo, err
}

// StopAll stops all running services in reverse order they were started
func (svc *LauncherService) StopAll() (err error) {

	svc.mux.Lock()
	slog.Info("Stopping all services", "count", len(svc.cmds))

	// use a copy of the commands as the command list will be mutated
	cmdsToStop := svc.cmds[:]

	svc.mux.Unlock()

	// stop each service
	for i := len(cmdsToStop) - 1; i >= 0; i-- {
		c := cmdsToStop[i]
		err = Stop(c.Path, c.Process.Pid)
	}
	time.Sleep(time.Millisecond)
	return err
}

// updateStatus updates the service  status
func (svc *LauncherService) updateStatus(svcInfo *launcher.ServiceInfo) {
	if svcInfo.PID != 0 {

		//Option A: use pidusage - doesn't work on Windows though
		//warning, pidusage is not very fast
		pidStats, _ := pidusage.GetStat(svcInfo.PID)
		if pidStats != nil {
			svcInfo.RSS = int(pidStats.Memory) // RSS is in KB
			svcInfo.CPU = int(pidStats.CPU)
		} else {
			svcInfo.CPU = 0
			svcInfo.RSS = 0
		}

		// Option B: use go-osstat - slower
		//cpuStat, err := cpu.Get()
		//if err == nil {
		//	svcInfo.CPU = cpuStat.CPUCount // FIXME: this is a counter, not %
		//}
		//memStat, err := memory.Get()
		//if err == nil {
		//	svcInfo.RSS = int(memStat.Used)
		//}

		//Option C: read statm directly. Fastest but only gets memory.
		//path := fmt.Sprintf("/proc/%d/statm", svcInfo.PID)
		//statm, err := ioutil.ReadFile(path)
		//if err == nil {
		//	fields := strings.Split(string(statm), " ")
		//	if len(fields) < 2 {
		//		// invalid data
		//	} else {
		//		rss, err := strconv.ParseInt(fields[1], 10, 64)
		//		if err != nil {
		//			// invalid data
		//		} else {
		//			svcInfo.RSS = int(rss * int64(os.Getpagesize()))
		//		}
		//	}
		//}
	}

}

// WatchServices watches the bin and plugins folder for changes and reloads
// This will detect adding new services without requiring a restart.
func (svc *LauncherService) WatchServices() error {
	svc.serviceWatcher, _ = fsnotify.NewWatcher()
	err := svc.serviceWatcher.Add(svc.f.Bin)
	if err == nil && svc.f.Plugins != "" {
		err = svc.serviceWatcher.Add(svc.f.Plugins)
	}
	if err == nil {
		go func() {
			for {
				select {
				case <-svc.done:
					slog.Info("service watcher ended")
					return
				case event := <-svc.serviceWatcher.Events:
					isRunning := svc.isRunning.Load()
					if isRunning {
						slog.Info("watcher event", "event", event)
						_ = svc.ScanServices()
					} else {
						slog.Info("service watcher stopped")
						return
					}
				case err := <-svc.serviceWatcher.Errors:
					slog.Error("error", "err", err)
				}
			}
		}()

	}
	return err
}

// Start the launcher service in the background
// Call stop to end.
func (svc *LauncherService) Start() error {
	svc.isRunning.Store(true)

	_ = svc.WatchServices()
	err := svc.ScanServices()
	if err != nil {
		return err
	}

	// autostart the services
	for _, name := range svc.cfg.Autostart {
		_, err2 := svc.StartService(name)
		if err2 != nil {
			err = err2
		}
	}
	return err
}

// StartListener subscribes to service requests using the given client
func (svc *LauncherService) StartListener(hc hubclient.IHubClient) (err error) {
	svc.hc = hc
	svc.mngSub, err = svc.hc.SubServiceRPC(
		launcher.LauncherManageCapability, svc.HandleRequest)
	return err
}

// Stop the launcher and all running services
func (svc *LauncherService) Stop() error {
	if svc.mngSub != nil {
		svc.mngSub.Unsubscribe()
		svc.mngSub = nil
	}
	svc.isRunning.Store(false)
	return svc.StopAll()
}

// NewLauncherService returns a new launcher instance for the services in the given services folder.
// This scans the folder for executables, adds these to the list of available services and autostarts services
// Logging will be enabled based on LauncherConfig.
func NewLauncherService(
	f utils.AppDirs,
	cfg config.LauncherConfig,
) *LauncherService {

	ls := &LauncherService{
		f:        f,
		cfg:      cfg,
		services: make(map[string]*launcher.ServiceInfo),
		cmds:     make([]*exec.Cmd, 0),
	}

	return ls
}
