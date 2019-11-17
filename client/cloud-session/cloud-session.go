package cloudsession

import (
	"encoding/base64"
	"errors"
	"strconv"

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

	// // Create ECS Cluster
	// _, err = createECSCluster(session)
	// if err != nil {
	// 	return nil, err
	// }

	// // Create ECS task
	// _, err = createECSTask(session)
	// if err != nil {
	// 	return nil, err
	// }

	// Create EC2 instances for the ECS cluster
	ec2Instances, err := createEC2Instances(session, instances, cloudConfig)
	if err != nil {
		return nil, err
	}

	// // Create an input queue
	// inputQueue, err := createQueue(session, InputQueue)
	// if err != nil {
	// 	return nil, err
	// }

	// // // Create an output queue
	// outputQueue, err := createQueue(session, OutputQueue)
	// if err != nil {
	// 	return nil, err
	// }

	return &CloudSession{
		session: session,
		// inputQueueURL:  inputQueue.QueueUrl,
		// outputQueueURL: outputQueue.QueueUrl,
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
		Image:     aws.String("hello-world"),
		Name:      aws.String("COMSM0010-worker-container"),
	}
	return svc.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{containerDefinition},
		Family:               aws.String("COMSM0010-worker-task"),
		Memory:               aws.String("400"),
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
