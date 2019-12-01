package cloudsession

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	InputQueue  string = "INPUT_QUEUE"
	OutputQueue string = "OUTPUT_QUEUE"
)

// CloudSession maintains information needed to make requests to the cloud
type CloudSession struct {
	session               *session.Session
	inputQueueURL         *string
	outputQueueURL        *string
	ec2WorkerInstanceIds  []*ec2.Instance
	ec2MonitorInstanceIds []*ec2.Instance
}

// WorkerResponse represents a worker's response to a task, which may or not be successful
type WorkerResponse struct {
	Success bool
	Nonce   *string
	Hash    *string
}

// New constructs a CloudSession
func New(instances int64, workerCloudConfig []byte, monitorCloudConfig []byte) (*CloudSession, error) {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return nil, err
	}

	// Create EC2 instances for the worker cluster
	ec2WorkerInstances, err := createEC2Instances(session, instances, workerCloudConfig)
	if err != nil {
		return nil, err
	}
	// Create EC2 instances for the monitoring cluster
	ec2MonitorInstances, err := createEC2Instances(session, 1, monitorCloudConfig)
	if err != nil {
		return nil, err
	}

	// Create an input queue
	inputQueue, err := createQueue(session, InputQueue)
	if err != nil {
		return nil, err
	}

	_, err = clearQueue(session, inputQueue.QueueUrl)
	if err != nil {
		return nil, err
	}

	// Create an output queue
	outputQueue, err := createQueue(session, OutputQueue)
	if err != nil {
		return nil, err
	}
	_, err = clearQueue(session, outputQueue.QueueUrl)
	if err != nil {
		return nil, err
	}

	// go func() {
	// 	time.Sleep(30 * time.Second)
	// 	ip, err := getEC2InstanceIP(session, *ec2MonitorInstances.Instances[0].InstanceId)
	// 	if err == nil {
	// 		log.Printf("Grafana metrics: http://%s:3000/d/gZ3GtvbWz/comsm0010-monitoring?orgId=1&refresh=10s&from=now-5m&to=now", *ip)
	// 	}
	// }()

	return &CloudSession{
		session:               session,
		inputQueueURL:         inputQueue.QueueUrl,
		outputQueueURL:        outputQueue.QueueUrl,
		ec2WorkerInstanceIds:  ec2WorkerInstances.Instances,
		ec2MonitorInstanceIds: ec2MonitorInstances.Instances,
	}, nil
}

// SendMessageOnQueue sends a message on a queue
func (cs *CloudSession) SendMessageOnQueue(queueType string, message *string, lower uint32, upper uint32, target int) error {
	qURL := ""
	if queueType == OutputQueue {
		qURL = *cs.outputQueueURL
	} else if queueType == InputQueue {
		qURL = *cs.inputQueueURL
	} else {
		return errors.New("Invalid queue type, must be InputQueue or OutputQueue")
	}

	// TODO: Move this to a util
	svc := sqs.New(cs.session)
	_, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(0),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Message": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: message,
			},
			"LowerBound": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(strconv.FormatUint(uint64(lower), 10)),
			},
			"UpperBound": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(strconv.FormatUint(uint64(upper), 10)),
			},
			"Target": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(strconv.FormatInt(int64(target), 10)),
			},
		},
		MessageBody: aws.String("FIXME"),
		QueueUrl:    &qURL,
	})
	return err
}

// WaitForResponse waits for a response from the send requests
func (cs *CloudSession) WaitForResponse(timeout int) (*WorkerResponse, error) {
	timeWaited := 0
	responsesReceived := 0

	for timeWaited < timeout {
		result, err := getMessageFromQueue(cs.session, cs.outputQueueURL)
		if err != nil {
			return nil, err
		}

		if len(result.Messages) > 0 {
			// Try and decode
			for _, message := range result.Messages {
				decoded, err := decodeWorkerMessage(message)
				if err != nil {
					return nil, err
				}
				responsesReceived++

				if decoded.Success {
					return decoded, nil
				}
			}
		}

		// If received a failure from every worker
		if responsesReceived == len(cs.ec2WorkerInstanceIds) {
			return nil, fmt.Errorf("No golden nonce found")
		}

		timeWaited += 10
	}

	return nil, fmt.Errorf("No result found after %d seconds", timeWaited)
}

// Cleanup tears down all infrastructure put in place to perform the computation
func (cs *CloudSession) Cleanup() {
	// Remove EC2 instances
	_, err := deleteEC2Instances(cs.session, cs.ec2WorkerInstanceIds)
	if err != nil {
		log.Fatal(err)
	}

	_, err = deleteEC2Instances(cs.session, cs.ec2MonitorInstanceIds)
	if err != nil {
		log.Fatal(err)
	}

	// Clear input queue
	_, err = clearQueue(cs.session, cs.inputQueueURL)
	if err != nil {
		log.Fatal(err)
	}

	// Clear output queue
	_, err = clearQueue(cs.session, cs.outputQueueURL)
	if err != nil {
		log.Fatal(err)
	}
}
