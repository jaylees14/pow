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

func checkError(err error, message string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("[%s]: %s", message, err.Error()))
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
	leadingZeros := flag.Int("n", 40, "number of leading zeros")
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
	cloudSession, err := cloudsession.New(1, workerCloudConfig, monitorCloudConfig)
	checkError(err, "Couldn't create session")
	log.Printf("Created cloud session...")

	// Configure Ctrl-C handler to perform graceful shutdown
	configureSIGTERMHandler(cloudSession)

	// Send a test message on the queue
	err = cloudSession.SendMessageOnQueue(cloudsession.InputQueue, *block, 0, ^uint32(0), *leadingZeros, "Compute if golden nonce exists between 0 and 100")
	checkError(err, "Couldn't send message")
	log.Printf("Computing golden nonce...")

	success, err := cloudSession.WaitForResponse()
	checkError(err, "Didn't receive response")
	log.Printf("Was success? %t", success.Success)

	err = cloudSession.Cleanup()
	checkError(err, "Couldn't clean up")
}
