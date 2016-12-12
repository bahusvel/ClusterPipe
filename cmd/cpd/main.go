package main

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bahusvel/ClusterPipe/common"
	"github.com/bahusvel/ClusterPipe/kissrpc"
	"github.com/urfave/cli"
)

var controllerAddress string
var controller *kissrpc.Client
var thisCPD = common.CPD{}

var procMutex = sync.RWMutex{}
var processes = map[common.TaskID]common.Task{}

func Run(task common.Task) error {
	cmd := exec.Command(task.Command, task.Args...)
	var err error
	if task.Stderr != nil {
		cmd.Stderr, err = task.Stderr.Open()
		if err != nil {
			return err
		}
	}
	if task.Stdout != nil {
		cmd.Stdout, err = task.Stdout.Open()
		if err != nil {
			return err
		}
	}
	if task.Stdin != nil {
		cmd.Stdin, err = task.Stdin.Open()
		if err != nil {
			return err
		}
	}

	procMutex.Lock()
	defer procMutex.Unlock()

	err = cmd.Start()
	if err != nil {
		return err
	}
	task.Process = cmd

	return nil
}

func reportStatus(status common.CPDStatus) {
	thisCPD.CurrentStatus = &status
	_, err := controller.Call("updateCPD", thisCPD.Host, status)
	if err != nil {
		log.Println("Error updating status", err)
	}
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "controller, c",
			Destination: &controllerAddress,
		},
		cli.StringFlag{
			Name:        "ip, i",
			Destination: &thisCPD.Host,
		},
	}

	app.Action = func(c *cli.Context) error {
		if controllerAddress == "" {
			return cli.NewExitError("You must specify controller address -c", -1)
		}
		if thisCPD.Host == "" {
			return cli.NewExitError("You must specify local address -i", -1)
		}

		go common.RunPipeServer()
		go Start()
		err := common.GatherHostInfo(&thisCPD)
		if err != nil {
			return err
		}
		controller, err = kissrpc.NewClient(controllerAddress)
		if err != nil {
			return err
		}
		for {
			_, err = controller.Call("registerCPD", &thisCPD)
			if err != nil {
				log.Println("Error registering CPD", err)
			} else {
				break
			}
			time.Sleep(1 * time.Second)
		}
		go common.StartStatMonitor(reportStatus)
		for {
			time.Sleep(1 * time.Second)
		}
	}

	app.Run(os.Args)

}
