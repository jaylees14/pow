package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func checkError(err error, message string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("[%s]: %s", message, err.Error()))
		os.Exit(1)
	}
}

func createSQSQueue(session *session.Session, queueName string) (*sqs.CreateQueueOutput, error) {
	svc := sqs.New(session)
	return svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]*string{
			"DelaySeconds":           aws.String("60"),
			"MessageRetentionPeriod": aws.String("86400"),
		},
	})
}

func sendMessageOnQueue(session *session.Session, qURL string, message string, lower uint32, upper uint32, target int, desc string) (*sqs.SendMessageOutput, error) {
	svc := sqs.New(session)
	return svc.SendMessage(&sqs.SendMessageInput{
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
}

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

func main() {
	cloudConfig, err := ioutil.ReadFile("cloud-config.yaml")
	if err != nil {
		fmt.Println("Couldn't read cloud-config", err.Error())
		return
	}
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	checkError(err, "Couldn't create session")

	cluster, err := createECSCluster(session)
	checkError(err, "Couldn't create ECS cluster")
	fmt.Println("Created cluster", cluster.GoString())

	task, err := createECSTask(session)
	checkError(err, "Couldn't create ECS task")
	fmt.Println("Created task", task.GoString())

	instances, err := createEC2Instances(session, 1, cloudConfig)
	checkError(err, "Couldn't create EC2 instances")
	fmt.Println("Created EC2 instances", instances.GoString())

	// inputQueue, err := createSQSQueue(session, "INPUT_QUEUE")
	// checkError(err, "Couldn't create input queue")

	// // outputQueue, err := createSQSQueue(session, "OUTPUT_QUEUE")
	// // checkError(err, "Couldn't create input queue")

	// fmt.Println("Created input queue: ", &inputQueue.QueueUrl)
	// // fmt.Println("Created output queue: ", &outputQueue.QueueUrl)

	// sendMessage, err := sendMessageOnQueue(session, *inputQueue.QueueUrl, "COMSM0010cloud", 0, 1000, 8, "Compute if golden nonce exists between 0 and 100")
	// checkError(err, "Couldn't send message")
	// fmt.Println("Created message: ", *sendMessage.MessageId)
}
