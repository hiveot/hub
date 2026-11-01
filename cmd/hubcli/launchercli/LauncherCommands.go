package launchercli

import (
	"context"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/cells/consumer"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	launcher_client "github.com/hiveot/hub/services/launcher/client"
	"github.com/urfave/cli/v3"
)

func LauncherListCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {

	return &cli.Command{
		Name: "ls",
		//Aliases: []string{"ls"},
		//ArgsUsage: "(no args)",
		Usage:    "List services and their runtime status",
		Category: "launcher",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 0 {
				return fmt.Errorf("no arguments expected")
			}
			err := HandleListServices((*r).Consumer)
			return err
		},
	}
}

func LauncherStartCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {

	return &cli.Command{
		Name: "start",
		//Aliases:   []string{"start"},
		ArgsUsage: "<servicename>|all",
		Usage:     "Start a service or all services",
		Category:  "launcher",
		//ArgsUsage: "start <serviceName> | all",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				return fmt.Errorf("expected service name")
			}
			err := HandleStartService(cmd.Args().First(), (*r).Consumer)
			return err
		},
	}
}

func LauncherStopCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {

	return &cli.Command{
		Name: "stop",
		//Aliases:   []string{"stop"},
		ArgsUsage: "<servicename>|all",
		Usage:     "Stop a service or all services",
		Category:  "launcher",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				return fmt.Errorf("expected service name")
			}
			err := HandleStopService(cmd.Args().First(), (*r).Consumer)
			return err
		},
	}
}

// HandleListServices prints a list of available services
func HandleListServices(co *consumer.Consumer) error {

	if co == nil {
		return fmt.Errorf("no Hub connection")
	}
	//lc := launcherclient.NewLauncherClient("", co)
	localTZ, _ := time.Now().Zone()

	fmt.Println("Service                      Size   Starts       PID    CPU   Memory   Status    Since (" + localTZ + ")          Last Error")
	fmt.Println("-------                      ----   ------   -------   ----   ------   -------   -------------------  -----------")
	entries, err := launcher_client.NewLauncherClient("", co).ListPlugins(false)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		status := "stopped"
		cpu := ""
		memory := ""
		pid := fmt.Sprintf("%d", entry.Pid)
		cpu = fmt.Sprintf("%d%%", entry.Cpu)
		memory = fmt.Sprintf("%d MB", entry.Rss/1024/1024)

		sinceTime := ""
		if entry.Running {
			status = "running"
			sinceTime = utils.FormatDateTime(entry.StartedTime)
		} else if entry.StoppedTime != "" {
			sinceTime = utils.FormatDateTime(entry.StoppedTime)
		}
		fmt.Printf("%-25s %4d MB   %6d   %7s   %4s   %6s   %6s   %-20s %s\n",
			entry.PluginID,
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
	//co.Disconnect()
	return nil
}

// HandleStartService starts a service
func HandleStartService(pluginID string, co *consumer.Consumer) error {
	var err error
	if co == nil {
		return fmt.Errorf("no Hub connection")
	}
	lc := launcher_client.NewLauncherClient("", co)

	if pluginID == "all" {
		err = lc.StartAllPlugins()
		if err != nil {
			//fmt.Println("Connect all failed with: ", err)
			return err
		}
		fmt.Printf("All services started\n")
	} else {
		info, err2 := lc.StartPlugin(pluginID)

		if err2 != nil {
			//fmt.Println("Connect failed:", err2)
			return err2
		}
		fmt.Printf("Service '%s' started\n", info.PluginID)
	}
	// last, show a list of running services
	err = HandleListServices(co)
	return err
}

// HandleStopService stops a service
func HandleStopService(serviceName string, co *consumer.Consumer) error {
	var err error

	if co == nil {
		return fmt.Errorf("no Hub connection")
	}
	lc := launcher_client.NewLauncherClient("", co)

	if serviceName == "all" {
		err := lc.StopAllPlugins(false)

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
		fmt.Printf("Service '%s' stopped\n", info.PluginID)
	}
	// last, show a list of running services
	err = HandleListServices(co)
	return err
}
