package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	cloudsession "github.com/jaylees14/pow/client/cloud-session"
)

func checkError(err error, message string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("[%s]: %s", message, err.Error()))
		os.Exit(1)
	}
}

func main() {
	cloudConfig, err := ioutil.ReadFile("cloud-config.yaml")
	if err != nil {
		fmt.Println("Couldn't read cloud-config", err.Error())
		return
	}

	cloudSession, err := cloudsession.New(1, cloudConfig)
	checkError(err, "Couldn't create session")

	err = cloudSession.SendMessageOnQueue(cloudsession.InputQueue, "COMSM0010cloud", 0, 1000, 8, "Compute if golden nonce exists between 0 and 100")
	checkError(err, "Couldn't send message")

	// time.Sleep(90 * time.Second)

	// err = cloudSession.Cleanup()
	// checkError(err, "Couldn't clean up")
}
