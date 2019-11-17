package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/jaylees14/pow/worker/nonce"
)

func checkError(err error, message string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("[%s]: %s", message, err.Error()))
		os.Exit(1)
	}
}

func getMessageFromQueue(session *session.Session, queueName string) (*sqs.ReceiveMessageOutput, error) {
	// Create a SQS service client.
	svc := sqs.New(session)
	// Get QueueURL
	resultURL, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, err
	}

	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: resultURL.QueueUrl,
		AttributeNames: aws.StringSlice([]string{
			"SentTimestamp",
		}),
		MaxNumberOfMessages: aws.Int64(1),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
		WaitTimeSeconds: aws.Int64(10),
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func decodeWorkerMessage(message *sqs.Message) (*nonce.WorkerConfig, error) {
	lowerBoundStr, ok := message.MessageAttributes["LowerBound"]
	if !ok {
		return nil, errors.New("Message didn't contain key LowerBound")
	}

	upperBoundStr, ok := message.MessageAttributes["UpperBound"]
	if !ok {
		return nil, errors.New("Message didn't contain key UpperBound")
	}

	targetStr, ok := message.MessageAttributes["Target"]
	if !ok {
		return nil, errors.New("Message didn't contain key Target")
	}

	messageStr, ok := message.MessageAttributes["Message"]
	if !ok {
		return nil, errors.New("Message didn't contain key Message")
	}

	lowerBound, err := strconv.ParseUint(*lowerBoundStr.StringValue, 10, 32)
	if err != nil {
		return nil, err
	}

	upperBound, err := strconv.ParseUint(*upperBoundStr.StringValue, 10, 32)
	if err != nil {
		return nil, err
	}

	target, err := strconv.Atoi(*targetStr.StringValue)
	if err != nil {
		return nil, err
	}

	return &nonce.WorkerConfig{
		Contents:   *messageStr.StringValue,
		LowerBound: uint32(lowerBound),
		UpperBound: uint32(upperBound),
		Target:     target,
		DebugDesc:  *message.Body,
	}, nil
}

func main() {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	checkError(err, "Couldn't create session")

	message, err := getMessageFromQueue(session, "INPUT_QUEUE")
	checkError(err, "Couldn't receive message")

	if len(message.Messages) == 0 {
		checkError(errors.New("Empty messages"), "No messages returned")
	}

	decoded, err := decodeWorkerMessage(message.Messages[0])
	checkError(err, "Couldn't decode message")

	n, err := nonce.CalculateGoldenNonce(decoded)
	if err != nil {
		if err, ok := err.(*nonce.NoNonceFoundError); ok {
			log.Fatalf("Error: %s", err.Error())
			return
		}
		checkError(err, "Couldn't calculate golden nonce")
		return
	}
	log.Printf("Nonce is %d for hash %s\n", n.Nonce, n.Hash)
}
