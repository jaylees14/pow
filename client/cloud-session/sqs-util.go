package cloudsession

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// -- SQS
func createQueue(session *session.Session, queueName string) (*sqs.CreateQueueOutput, error) {
	svc := sqs.New(session)
	return svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]*string{
			"DelaySeconds":           aws.String("60"),
			"MessageRetentionPeriod": aws.String("86400"),
		},
	})
}

func getMessageFromQueue(session *session.Session, queueURL *string) (*sqs.ReceiveMessageOutput, error) {
	// Create a SQS service client.
	svc := sqs.New(session)
	return svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: queueURL,
		AttributeNames: aws.StringSlice([]string{
			"SentTimestamp",
		}),
		MaxNumberOfMessages: aws.Int64(1),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
		WaitTimeSeconds: aws.Int64(10),
	})
}

func decodeWorkerMessage(message *sqs.Message) (*WorkerResponse, error) {
	successStr, ok := message.MessageAttributes["Success"]
	if !ok {
		return nil, errors.New("Message didn't contain key Success")
	}

	success, err := strconv.ParseBool(*successStr.StringValue)
	if err != nil {
		return nil, err
	}

	fmt.Println(*message.Body)

	return &WorkerResponse{
		Success: success,
	}, nil
}

func clearQueue(session *session.Session, queueURL *string) (*sqs.PurgeQueueOutput, error) {
	svc := sqs.New(session)
	return svc.PurgeQueue(&sqs.PurgeQueueInput{
		QueueUrl: queueURL,
	})
}
