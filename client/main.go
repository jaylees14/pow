package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	cloudsession "github.com/jaylees14/pow/client/cloud-session"
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
		fmt.Println("Gracefully shutting down...")
		session.Cleanup()
		os.Exit(0)
	}()
}

func main() {
	// CLI Args
	block := flag.String("block", "COMSM0010cloud", "block of data the nonce is appended to")
	leadingZeros := flag.Int("d", 40, "number of leading zeros")
	workers := flag.Int("n", 1, "number of workers")
	timeout := flag.Int("timeout", 360, "timeout in seconds")
	workerCloudConfigPath := flag.String("worker-cloud-config", "worker-cloud-config.yaml", "path to worker cloud config file")
	monitorCloudConfigPath := flag.String("monitor-cloud-config", "monitor-cloud-config.yaml", "path to monitor cloud config file")
	flag.Parse()

	workerCloudConfig, err := ioutil.ReadFile(*workerCloudConfigPath)
	if err != nil {
		fmt.Println("Couldn't read worker-cloud-config", err.Error())
		return
	}
	monitorCloudConfig, err := ioutil.ReadFile(*monitorCloudConfigPath)
	if err != nil {
		fmt.Println("Couldn't read monitor-cloud-config", err.Error())
		return
	}

	// Set up the cloud infra, VMs etc.
	cloudSession, err := cloudsession.New(int64(*workers), workerCloudConfig, monitorCloudConfig)
	checkError(err, "Couldn't create session", nil)
	log.Printf("Created cloud session...")

	// Configure Ctrl-C handler to perform graceful shutdown
	configureSIGTERMHandler(cloudSession)

	// Partition the work between each of the workers
	maxValue := ^uint32(0)
	split := maxValue / uint32(*workers)
	for i := uint32(0); i < uint32(*workers); i++ {
		startValue := i * split
		endValue := (i + 1) * split
		if i == uint32(*workers)-1 {
			endValue = maxValue
		}
		err = cloudSession.SendMessageOnQueue(cloudsession.InputQueue, *block, startValue, endValue, *leadingZeros, "Compute if golden nonce exists between 0 and 100")
		checkError(err, "Couldn't send message", cloudSession)
	}
	log.Printf("Computing golden nonce...")

	success, err := cloudSession.WaitForResponse(*timeout)
	checkError(err, "Didn't receive response", cloudSession)

	if success.Success {
		log.Printf("Success! Found golden nonce %s with hash %s", *success.Nonce, *success.Hash)
	} else {
		log.Printf("Failure... no nonce found")
	}

	cloudSession.Cleanup()
}
