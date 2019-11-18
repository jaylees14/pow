package cloudsession

import (
	"encoding/base64"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	InputQueue  string = "INPUT_QUEUE"
	OutputQueue string = "OUTPUT_QUEUE"
)

// CloudSession maintains information needed to make requests to the cloud
type CloudSession struct {
	session        *session.Session
	inputQueueURL  *string
	outputQueueURL *string
	ec2InstanceIds []*ec2.Instance
}

// New constructs a CloudSession
func New(instances int64, cloudConfig []byte) (*CloudSession, error) {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return nil, err
	}

	// Create ECS Cluster
	cluster, err := createECSCluster(session)
	if err != nil {
		return nil, err
	}

	// Create ECS task
	task, err := createECSTask(session)
	if err != nil {
		return nil, err
	}

	// Create an input queue
	inputQueue, err := createQueue(session, InputQueue)
	if err != nil {
		return nil, err
	}

	// Create an output queue
	outputQueue, err := createQueue(session, OutputQueue)
	if err != nil {
		return nil, err
	}

	// Create EC2 instances for the ECS cluster
	ec2Instances, err := createEC2Instances(session, instances, cloudConfig)
	if err != nil {
		return nil, err
	}

	// Wait for EC2 instances to become ready
	for {
		if ec2InstancesReady(session, cluster.Cluster.ClusterName, len(ec2Instances.Instances)) {
			log.Println("EC2 instances ready!")
			break
		}
		log.Println("Waiting for EC2 instances to spin up...")
		time.Sleep(5 * time.Second)
	}

	// Start the task
	_, err = startECSTask(session, cluster.Cluster.ClusterName, task.TaskDefinition.TaskDefinitionArn)
	if err != nil {
		return nil, err
	}

	return &CloudSession{
		session:        session,
		inputQueueURL:  inputQueue.QueueUrl,
		outputQueueURL: outputQueue.QueueUrl,
		ec2InstanceIds: ec2Instances.Instances,
	}, nil
}

// SendMessageOnQueue sends a message on a queue
func (cs *CloudSession) SendMessageOnQueue(queueType string, message string, lower uint32, upper uint32, target int, desc string) error {
	qURL := ""
	if queueType == OutputQueue {
		qURL = *cs.outputQueueURL
	} else if queueType == InputQueue {
		qURL = *cs.inputQueueURL
	} else {
		return errors.New("Invalid queue type, must be InputQueue or OutputQueue")
	}

	svc := sqs.New(cs.session)
	_, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(0),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Message": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(message),
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
		MessageBody: aws.String(desc),
		QueueUrl:    &qURL,
	})
	return err
}

// WaitForResponse waits for a response from the send requests
func (cs *CloudSession) WaitForResponse() bool {
	timeWaited := 0

	for timeWaited < 180 {
		result, err := getMessageFromQueue(cs.session, cs.outputQueueURL)
		if err != nil {
			log.Printf("Something went wrong getting output message: %s", err.Error())
		}

		if len(result.Messages) > 0 {
			// Try and decode
			for _, message := range result.Messages {
				log.Printf("Got message: %s", message.MessageAttributes)
				return true
			}
		}
		time.Sleep(10 * time.Second)
		timeWaited += 10
	}
	return false
}

// Cleanup tears down all infrastructure put in place to perform the computation
func (cs *CloudSession) Cleanup() error {
	// Remove EC2 instances
	_, err := deleteEC2Instances(cs.session, cs.ec2InstanceIds)
	if err != nil {
		return err
	}

	// Clear input queue
	_, err = clearQueue(cs.session, cs.inputQueueURL)
	if err != nil {
		return err
	}

	// Clear output queue
	_, err = clearQueue(cs.session, cs.outputQueueURL)
	if err != nil {
		return err
	}

	return nil
}

// -- Helper Methods
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

func clearQueue(session *session.Session, queueURL *string) (*sqs.PurgeQueueOutput, error) {
	svc := sqs.New(session)
	return svc.PurgeQueue(&sqs.PurgeQueueInput{
		QueueUrl: queueURL,
	})
}

// -- ECS
func createECSCluster(session *session.Session) (*ecs.CreateClusterOutput, error) {
	svc := ecs.New(session)
	return svc.CreateCluster(&ecs.CreateClusterInput{
		ClusterName: aws.String("COMSM0010-worker-cluster"),
	})
}

func createECSTask(session *session.Session) (*ecs.RegisterTaskDefinitionOutput, error) {
	svc := ecs.New(session)
	containerDefinition := &ecs.ContainerDefinition{
		Essential: aws.Bool(true),
		Image:     aws.String("615057327315.dkr.ecr.us-east-1.amazonaws.com/jaylees/comsm0010-worker:latest"),
		Name:      aws.String("COMSM0010-worker-container"),
	}
	return svc.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{containerDefinition},
		Family:               aws.String("COMSM0010-worker-task"),
		Memory:               aws.String("400"),
	})
}

func startECSTask(session *session.Session, clusterName *string, taskName *string) (*ecs.RunTaskOutput, error) {
	svc := ecs.New(session)
	return svc.RunTask(&ecs.RunTaskInput{
		Cluster:        clusterName,
		Count:          aws.Int64(1),
		TaskDefinition: taskName,
	})
}

// -- EC2
func createEC2Instances(session *session.Session, count int64, config []byte) (*ec2.Reservation, error) {
	svc := ec2.New(session)
	iamRole := ec2.IamInstanceProfileSpecification{
		Name: aws.String("ecsInstanceRole"),
	}
	return svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:            aws.String("ami-00129b193dc81bc31"),
		InstanceType:       aws.String("t2.micro"),
		KeyName:            aws.String("COMSM0010"),
		MinCount:           aws.Int64(count),
		IamInstanceProfile: &iamRole,
		MaxCount:           aws.Int64(count),
		SecurityGroups:     aws.StringSlice([]string{"comsm0010-sg-open"}),
		UserData:           aws.String(base64.StdEncoding.EncodeToString(config)),
	})
}

func ec2InstancesReady(session *session.Session, clusterName *string, expectedCount int) bool {
	svc := ecs.New(session)
	desc, err := svc.DescribeClusters(&ecs.DescribeClustersInput{
		Clusters: aws.StringSlice([]string{*clusterName}),
	})
	if err != nil {
		log.Fatalln("Couldn't read instance status", err.Error())
		return false
	}

	return *desc.Clusters[0].RegisteredContainerInstancesCount == int64(expectedCount)
}

func deleteEC2Instances(session *session.Session, instances []*ec2.Instance) (*ec2.TerminateInstancesOutput, error) {
	svc := ec2.New(session)

	ids := make([]*string, len(instances))
	for i, instance := range instances {
		ids[i] = instance.InstanceId
	}

	return svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: ids,
	})
}
