package launchercli

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/services/launcher/launcherclient"
	"github.com/urfave/cli/v2"
	"time"
)

func LauncherListCommand(hc **messaging.Consumer) *cli.Command {

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
			err := HandleListServices(*hc)
			return err
		},
	}
}

func LauncherStartCommand(hc **messaging.Consumer) *cli.Command {

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
			err := HandleStartService(cCtx.Args().First(), *hc)
			return err
		},
	}
}

func LauncherStopCommand(hc **messaging.Consumer) *cli.Command {

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
			err := HandleStopService(cCtx.Args().First(), *hc)
			return err
		},
	}
}

// HandleListServices prints a list of available services
func HandleListServices(hc *messaging.Consumer) error {

	if hc == nil {
		return fmt.Errorf("no Hub connection")
	}
	lc := launcherclient.NewLauncherClient("", hc)
	localTZ, _ := time.Now().Zone()

	fmt.Println("Service                      Size   Starts       PID    CPU   Memory   Status    Since (" + localTZ + ")          Last Error")
	fmt.Println("-------                      ----   ------   -------   ----   ------   -------   -------------------  -----------")
	entries, err := lc.List(false)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		status := "stopped"
		cpu := ""
		memory := ""
		pid := fmt.Sprintf("%d", entry.PID)
		cpu = fmt.Sprintf("%d%%", entry.CPU)
		memory = fmt.Sprintf("%d MB", entry.RSS/1024/1024)

		sinceTime := ""
		if entry.Running {
			status = "running"
			sinceTime = utils.FormatMSE(entry.StartTimeMSE, true)
		} else if entry.StopTimeMSE != 0 {
			sinceTime = utils.FormatMSE(entry.StopTimeMSE, true)
		}
		fmt.Printf("%-25s %4d MB   %6d   %7s   %4s   %6s   %6s   %-20s %s\n",
			entry.Name,
			entry.Size/1024/1024,
			entry.StartCount,
			pid,
			cpu,
			memory,
			status,
			sinceTime,
			entry.Status,
		)
	}
	// for testing
	//time.Sleep(time.Second * 1)
	//hc.Disconnect()
	return nil
}

// HandleStartService starts a service
func HandleStartService(serviceName string, hc *messaging.Consumer) error {
	var err error
	if hc == nil {
		return fmt.Errorf("no Hub connection")
	}
	lc := launcherclient.NewLauncherClient("", hc)

	if serviceName == "all" {
		err := lc.StartAllPlugins()

		if err != nil {
			//fmt.Println("Connect all failed with: ", err)
			return err
		}
		fmt.Printf("All services started\n")
	} else {
		info, err2 := lc.StartPlugin(serviceName)

		if err2 != nil {
			//fmt.Println("Connect failed:", err2)
			return err2
		}
		fmt.Printf("Service '%s' started\n", info.Name)
	}
	// last, show a list of running services
	err = HandleListServices(hc)
	return err
}

// HandleStopService stops a service
func HandleStopService(serviceName string, hc *messaging.Consumer) error {
	var err error

	if hc == nil {
		return fmt.Errorf("no Hub connection")
	}
	lc := launcherclient.NewLauncherClient("", hc)

	if serviceName == "all" {
		err := lc.StopAllPlugins()

		if err != nil {
			fmt.Println("Stop all failed:", err)
			return err
		}
		fmt.Printf("All services stopped\n")

	} else {
		info, err := lc.StopPlugin(serviceName)
		if err != nil {
			fmt.Printf("Stop %s failed: %s\n", serviceName, err)
			return err
		}
		fmt.Printf("Service '%s' stopped\n", info.Name)
	}
	// last, show a list of running services
	err = HandleListServices(hc)
	return err
}
