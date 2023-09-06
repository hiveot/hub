// Package launcher with the launcher interface
package launcher

// ServiceName used to connect to this service
const ServiceName = "launcher"

// ServiceInfo contains the running status of a service
type ServiceInfo struct {
	// CPU usage in %. 0 when not running
	CPU int

	// RSS (Resident Set Size) Memory usage in Bytes. 0 when not running.
	RSS int

	// Service modified time ISO8601
	ModifiedTime string

	// name of the service
	Name string

	// path to service executable
	Path string

	// Program PID when started. This remains after stopping.
	PID int

	// Service is currently running
	Running bool

	// binary size of the service in bytes
	Size int64

	// The last status message received from the process
	Status string

	// Number of times the service was restarted
	StartCount int

	// Starting time of the service in ISO8601
	StartTime string

	// Stopped time of the service in ISO8601
	StopTime string

	// uptime time the service is running in seconds.
	Uptime int
}

// ILauncher defines the POGS based interface of the launcher service
type ILauncher interface {

	// List services
	List(onlyRunning bool) ([]ServiceInfo, error)

	// StartService start a service
	StartService(name string) (ServiceInfo, error)

	// StartAll starts all enabled services
	// This returns the error from the last service that could not be started
	StartAll() error

	// StopService stops a running service
	StopService(name string) (ServiceInfo, error)

	// StopAll running services
	StopAll() error
}
