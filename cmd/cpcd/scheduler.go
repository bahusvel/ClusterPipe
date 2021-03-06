package main

import (
	"fmt"

	"github.com/bahusvel/ClusterPipe/common"
)

var rrScheduler = &RoundRobin{}

var schedulers = map[string]Scheduler{
	"":         rrScheduler,
	"rr":       rrScheduler,
	"lowest":   NoneScheduler{},
	"specific": NoneScheduler{},
	"same":     NoneScheduler{},
}

type Scheduler interface {
	Schedule(task *common.Task) error
}

var taskIDIncrement = common.TaskID(0)

type RoundRobin struct {
	counter int
}

func (this *RoundRobin) Schedule(task *common.Task) error {
	nodes := getNodes()
	if len(nodes) == 0 {
		return fmt.Errorf("Cluster does not have any nodes")
	}
	task.Node = nodes[this.counter%len(nodes)].Host
	this.counter++
	task.TID = taskIDIncrement
	taskIDIncrement++
	return nil
}

type NoneScheduler struct {
}

func (this NoneScheduler) Schedule(task *common.Task) error {
	return fmt.Errorf("This method is not yet implemented")
}
