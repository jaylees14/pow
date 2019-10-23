package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// GoldenNonce computed from nonce appended to input string
type goldenNonce struct {
	Nonce uint32
	Hash  string
}

type workerConfig struct {
	Contents   string
	LowerBound uint32
	UpperBound uint32
	Target     int
	DebugDesc  string
}

type noNonceFoundError struct {
	err string
}

func (e *noNonceFoundError) Error() string {
	return fmt.Sprintf("Couldn't find nonce: %s", e.err)
}

// CalculateGoldenNonce computes golden nonce for the string concatenated with all nonces in range [start, end)
func calculateGoldenNonce(config *workerConfig) (*goldenNonce, error) {
	for i := config.LowerBound; i < config.UpperBound; i++ {
		hash, err := hash(config.Contents, i)
		if err != nil {
			return nil, err
		}

		zeros := leadingZeros(hash)
		if zeros >= config.Target {
			return &goldenNonce{i, hex.EncodeToString(hash)}, nil
		}
	}
	return nil, &noNonceFoundError{fmt.Sprintf("No nonce found of length %d between %d and %d", config.Target, config.LowerBound, config.UpperBound)}
}

func hash(block string, nonce uint32) ([]byte, error) {
	// Convert the nonce to a byte[]
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, nonce)
	if err != nil {
		return nil, err
	}

	blockBytes := []byte(block)
	bytes := append(blockBytes, buf.Bytes()...)

	// Complete one or two hashes
	firstHash := sha256.Sum256(bytes)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:], nil
}

func leadingZeros(arr []byte) int {
	leadingZeros := 0

	for _, b := range arr {
		for i := 7; i >= 0; i-- {
			mask := byte(1 << uint(i))
			if b&mask != 0 {
				return leadingZeros
			}
			leadingZeros++
		}
	}
	return leadingZeros
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

func checkError(err error, message string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("[%s]: %s", message, err.Error()))
		os.Exit(1)
	}
}

func decodeWorkerMessage(message *sqs.Message) (*workerConfig, error) {
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

	return &workerConfig{
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

	nonce, err := calculateGoldenNonce(decoded)
	if err != nil {
		if err, ok := err.(*noNonceFoundError); ok {
			fmt.Printf("Error: %s", err.Error())
			return
		}
		checkError(err, "Couldn't calculate golden nonce")
		return
	}
	fmt.Printf("Nonce is %d for hash: %s\n", nonce.Nonce, nonce.Hash)
}
