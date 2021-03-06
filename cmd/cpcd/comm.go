package main

import (
	"log"
	"time"

	"github.com/bahusvel/ClusterPipe/common"
	"github.com/bahusvel/ClusterPipe/kissrpc"
)

const (
	COM_PORT = ":3334"
)

func ping() {
	log.Println("Got ping")
}

func prepareTask(task common.Task) (common.Task, error) {
	scheduler := schedulers[task.Scheduler]
	scheduler.Schedule(&task)
	_, err := kissrpc.SingleCall(task.Node, "prepareTask", task)
	if err != nil {
		return task, err
	}
	return task, nil
}

func jobStatus(job string) string {
	return ""
}

func getNodes() []*common.CPD {
	currentNodes := []*common.CPD{}
	for _, node := range nodes {
		currentNodes = append(currentNodes, node)
	}
	//return []*common.CPD{&common.CPD{}}
	return currentNodes
}

func registerCPD(node common.CPD) {
	nodes[node.Host] = &node
}

func updateCPD(cpd string, status common.CPDStatus) {
	status.LastCheckin = time.Now()
	nodes[cpd].CurrentStatus = &status
	if useInflux {
		InfluxInsert(cpd, status)
	}
}

func Start() error {
	server := kissrpc.NewServer(COM_PORT)
	server.AddFunc("ping", ping)
	server.AddFunc("registerCPD", registerCPD)
	server.AddFunc("prepareTask", prepareTask)
	server.AddFunc("updateCPD", updateCPD)
	server.AddFunc("getNodes", getNodes)
	return server.Start()
}
