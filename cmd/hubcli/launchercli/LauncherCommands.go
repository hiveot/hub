package launchercli

import (
	"fmt"
	"github.com/hiveot/hub/core/launcher"
	"github.com/urfave/cli/v2"
)

func LauncherListCommand(runFolder *string) *cli.Command {

	return &cli.Command{
		Name: "ls",
		//Aliases: []string{"ls"},
		//ArgsUsage: "(no args)",
		Usage:    "List services and their runtime status",
		Category: "launcher",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				return fmt.Errorf("no arguments expected")
			}
			err := HandleListServices(*runFolder)
			return err
		},
	}
}

func LauncherStartCommand(runFolder *string) *cli.Command {

	return &cli.Command{
		Name: "start",
		//Aliases:   []string{"start"},
		ArgsUsage: "<servicename>|all",
		Usage:     "Start a service or all services",
		Category:  "launcher",
		//ArgsUsage: "start <serviceName> | all",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 1 {
				return fmt.Errorf("expected service name")
			}
			err := HandleStartService(*runFolder, cCtx.Args().First())
			return err
		},
	}
}

func LauncherStopCommand(runFolder *string) *cli.Command {

	return &cli.Command{
		Name: "stop",
		//Aliases:   []string{"stop"},
		ArgsUsage: "<servicename>|all",
		Usage:     "Stop a service or all services",
		Category:  "launcher",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 1 {
				return fmt.Errorf("expected service name")
			}
			err := HandleStopService(*runFolder, cCtx.Args().First())
			return err
		},
	}
}

// HandleListServices prints a list of available services
func HandleListServices(runFolder string) error {
	var ls launcher.ILauncher
	var err error

	if err != nil {
		return err
	}

	fmt.Println("Service                      Size   Starts       PID    CPU   Memory   Status    Last Error")
	fmt.Println("-------                      ----   ------   -------   ----   ------   -------   -----------")
	entries, _ := ls.List(false)
	for _, entry := range entries {
		status := "stopped"
		cpu := ""
		memory := ""
		pid := fmt.Sprintf("%d", entry.PID)
		cpu = fmt.Sprintf("%d%%", entry.CPU)
		memory = fmt.Sprintf("%d MB", entry.RSS/1024/1024)
		if entry.Running {
			status = "running"
		}
		fmt.Printf("%-25s %4d MB   %6d   %7s   %4s   %6s   %6s   %s\n",
			entry.Name,
			entry.Size/1024/1024,
			entry.StartCount,
			pid,
			cpu,
			memory,
			status,
			entry.Status,
		)
	}
	return nil
}

// HandleStartService starts a service
func HandleStartService(runFolder string, serviceName string) error {
	var ls launcher.ILauncher
	var err error

	if serviceName == "all" {
		err := ls.StartAll()

		if err != nil {
			//fmt.Println("Connect all failed with: ", err)
			return err
		}
		fmt.Printf("All services started\n")
	} else {
		info, err2 := ls.StartService(serviceName)

		if err2 != nil {
			//fmt.Println("Connect failed:", err2)
			return err2
		}
		fmt.Printf("Service '%s' started\n", info.Name)
	}
	// last, show a list of running services
	err = HandleListServices(runFolder)
	return err
}

// HandleStopService stops a service
func HandleStopService(runFolder string, serviceName string) error {
	var ls launcher.ILauncher
	var err error

	if err != nil {
		return err
	}

	if serviceName == "all" {
		err := ls.StopAll()

		if err != nil {
			fmt.Println("Stop all failed:", err)
			return err
		}
		fmt.Printf("All services stopped\n")

	} else {
		info, err := ls.StopService(serviceName)
		if err != nil {
			fmt.Printf("Stop %s failed: %s\n", serviceName, err)
			return err
		}
		fmt.Printf("Service '%s' stopped\n", info.Name)
	}
	// last, show a list of running services
	err = HandleListServices(runFolder)
	return err
}
