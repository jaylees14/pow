package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	cloudsession "github.com/jaylees14/pow/client/cloud-session"
	"github.com/jaylees14/pow/client/cmd"
)

const (
	workerCloudConfigPath  string = "worker-cloud-config.yaml"
	monitorCloudConfigPath string = "monitor-cloud-config.yaml"
)

func checkError(err error, message string, session *cloudsession.CloudSession) {
	if err != nil {
		log.Printf(fmt.Sprintf("[%s]: %s", message, err.Error()))
		if session != nil {
			session.Cleanup()
		}
		os.Exit(1)
	}
}

func configureSIGTERMHandler(session *cloudsession.CloudSession) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Gracefully shutting down...")
		session.Cleanup()
		os.Exit(0)
	}()
}

// Partition the work between each of the workers
func partitionWork(config *cmd.WorkerConfig, cloudSession *cloudsession.CloudSession) error {
	maxValue := ^uint32(0)
	split := maxValue / uint32(config.Workers)
	for i := uint32(0); i < uint32(config.Workers); i++ {
		startValue := i * split
		endValue := (i + 1) * split
		if i == uint32(config.Workers)-1 {
			endValue = maxValue
		}
		err := cloudSession.SendMessageOnQueue(cloudsession.InputQueue, config.Block, startValue, endValue, config.LeadingZeros)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	config, err := cmd.ParseArgs()
	checkError(err, "Couldn't parse arguments: ", nil)

	config.LogConfig()

	// Read worker configurations
	workerCloudConfig, err := ioutil.ReadFile(workerCloudConfigPath)
	checkError(err, "Couldn't read worker-cloud-config", nil)

	monitorCloudConfig, err := ioutil.ReadFile(monitorCloudConfigPath)
	checkError(err, "Couldn't read monitor-cloud-config", nil)

	// Set up the cloud infra, VMs etc.
	cloudSession, err := cloudsession.New(int64(config.Workers), workerCloudConfig, monitorCloudConfig)
	checkError(err, "Couldn't create session", nil)
	log.Printf("Created cloud session")

	// Configure Ctrl-C handler to perform graceful shutdown
	configureSIGTERMHandler(cloudSession)

	err = partitionWork(config, cloudSession)
	checkError(err, "Couldn't send message", cloudSession)

	log.Printf("Computing golden nonce")

	success, err := cloudSession.WaitForResponse(config.Timeout)
	checkError(err, "Didn't receive response", cloudSession)

	if success.Success {
		log.Printf("Success! Found golden nonce %s with hash %s", *success.Nonce, *success.Hash)
	} else {
		log.Printf("Failure: no nonce found")
	}

	cloudSession.Cleanup()
}
