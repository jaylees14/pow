package cloudsession

import (
	"errors"
	"fmt"
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
	session               *session.Session
	inputQueueURL         *string
	outputQueueURL        *string
	ec2WorkerInstanceIds  []*ec2.Instance
	ec2MonitorInstanceIds []*ec2.Instance
	advisorService        *ecs.Service
	grafanaService        *ecs.Service
}

// WorkerResponse represents a worker's response to a task, which may or not be successful
type WorkerResponse struct {
	Success bool
}

// New constructs a CloudSession
func New(instances int64, workerCloudConfig []byte, monitorCloudConfig []byte) (*CloudSession, error) {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return nil, err
	}

	// Create ECS Cluster for workers
	workerCluster, err := createECSCluster(session, "COMSM0010-worker-cluster")
	if err != nil {
		return nil, err
	}

	// Create ECS Cluster for monitoring
	monitorCluster, err := createECSCluster(session, "COMSM0010-monitor-cluster")
	if err != nil {
		return nil, err
	}

	// Create worker ECS task
	workerContainer := &ecs.ContainerDefinition{
		Essential:    aws.Bool(true),
		Image:        aws.String("615057327315.dkr.ecr.us-east-1.amazonaws.com/jaylees/comsm0010-worker:latest"),
		Name:         aws.String("COMSM0010-worker-container"),
		PortMappings: []*ecs.PortMapping{createPortMapping(2112, 2112)},
	}
	workerTask, err := createECSTask(session, "worker", workerContainer, []*ecs.Volume{})
	if err != nil {
		return nil, err
	}

	// Create advisor ECS task
	advisorContainer := &ecs.ContainerDefinition{
		Essential:    aws.Bool(true),
		Image:        aws.String("google/cadvisor:latest"),
		Name:         aws.String("COMSM0010-advisor-container"),
		PortMappings: []*ecs.PortMapping{createPortMapping(8080, 8080)},
		MountPoints: []*ecs.MountPoint{
			createMountPoint("root", "/rootfs", true),
			createMountPoint("var_run", "/var/run", false),
			createMountPoint("sys", "/sys", true),
			createMountPoint("var_lib_docker", "/var/lib/docker", true),
			createMountPoint("cgroup", "/sys/fs/cgroup", true),
		},
	}
	advisorVolumes := []*ecs.Volume{
		createVolume("root", "/"),
		createVolume("var_run", "/var/run"),
		createVolume("sys", "/sys"),
		createVolume("var_lib_docker", "/var/lib/docker"),
		createVolume("cgroup", "/cgroup"),
	}
	advisorTask, err := createECSTask(session, "advisor", advisorContainer, advisorVolumes)
	if err != nil {
		return nil, err
	}

	// Create prometheus ECS task
	promContainer := &ecs.ContainerDefinition{
		Essential:    aws.Bool(true),
		Image:        aws.String("prom/prometheus:latest"),
		Name:         aws.String("COMSM0010-prom-container"),
		PortMappings: []*ecs.PortMapping{createPortMapping(9090, 9090)},
		MountPoints:  []*ecs.MountPoint{createMountPoint("etc_prom", "/etc/prometheus", true)},
	}
	promVolumes := []*ecs.Volume{
		createVolume("etc_prom", "/etc/prometheus"),
	}
	prometheusTask, err := createECSTask(session, "prometheus", promContainer, promVolumes)
	if err != nil {
		return nil, err
	}

	// Create grafana ECS task
	grafanaContainer := &ecs.ContainerDefinition{
		Essential:    aws.Bool(true),
		Image:        aws.String("grafana/grafana:latest"),
		Name:         aws.String("COMSM0010-grafana-container"),
		PortMappings: []*ecs.PortMapping{createPortMapping(3000, 3000)},
	}
	grafanaTask, err := createECSTask(session, "grafana", grafanaContainer, []*ecs.Volume{})
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

	// Wait for EC2 instances to become ready
	clusterSizes := map[string]int{
		*workerCluster.Cluster.ClusterName:  len(ec2WorkerInstances.Instances),
		*monitorCluster.Cluster.ClusterName: len(ec2MonitorInstances.Instances),
	}

	for {
		if ec2InstancesReady(session, clusterSizes) {
			log.Println("EC2 instances ready!")
			break
		}
		log.Println("Waiting for EC2 instances to spin up...")
		time.Sleep(10 * time.Second)
	}

	// Start the prom task
	_, err = startECSTask(session, monitorCluster.Cluster.ClusterName, prometheusTask.TaskDefinition.TaskDefinitionArn, 1)
	if err != nil {
		return nil, err
	}

	// Start the worker task
	_, err = startECSTask(session, workerCluster.Cluster.ClusterName, workerTask.TaskDefinition.TaskDefinitionArn, instances)
	if err != nil {
		return nil, err
	}

	// Start advisor service
	advisorService, err := startDaemonECSService(session, workerCluster.Cluster.ClusterName, advisorTask.TaskDefinition.TaskDefinitionArn)
	if err != nil {
		return nil, err
	}

	grafanaService, err := startDaemonECSService(session, monitorCluster.Cluster.ClusterName, grafanaTask.TaskDefinition.TaskDefinitionArn)
	if err != nil {
		return nil, err
	}

	return &CloudSession{
		session:               session,
		inputQueueURL:         inputQueue.QueueUrl,
		outputQueueURL:        outputQueue.QueueUrl,
		ec2WorkerInstanceIds:  ec2WorkerInstances.Instances,
		ec2MonitorInstanceIds: ec2MonitorInstances.Instances,
		advisorService:        advisorService.Service,
		grafanaService:        grafanaService.Service,
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

	// TODO: Move this to a util
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

	_, err = stopECSService(cs.session, cs.advisorService.ClusterArn, cs.advisorService.ServiceName)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stopECSService(cs.session, cs.grafanaService.ClusterArn, cs.grafanaService.ServiceName)
	if err != nil {
		log.Fatal(err)
	}
}
