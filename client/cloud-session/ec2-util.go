package cloudsession

import (
	"encoding/base64"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func createEC2Instances(session *session.Session, imageId string, count int64, config []byte) (*ec2.Reservation, error) {
	svc := ec2.New(session)
	iamRole := ec2.IamInstanceProfileSpecification{
		Name: aws.String("ecsInstanceRole"),
	}

	return svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:            aws.String(imageId),
		InstanceType:       aws.String("t2.micro"),
		KeyName:            aws.String("COMSM0010"),
		MinCount:           aws.Int64(count),
		IamInstanceProfile: &iamRole,
		MaxCount:           aws.Int64(count),
		SecurityGroups:     aws.StringSlice([]string{"comsm0010-sg-open"}),
		UserData:           aws.String(base64.StdEncoding.EncodeToString(config)),
	})
}

func getEC2InstanceIP(session *session.Session, instanceID string) (*string, error) {
	svc := ec2.New(session)
	desc, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return desc.Reservations[0].Instances[0].PublicIpAddress, nil
}

func ec2InstancesReady(session *session.Session, clusterSizes map[string]int) bool {
	svc := ecs.New(session)

	// Get all cluster names from map
	clusterNames := make([]string, 0, len(clusterSizes))
	for k := range clusterSizes {
		clusterNames = append(clusterNames, k)
	}

	desc, err := svc.DescribeClusters(&ecs.DescribeClustersInput{
		Clusters: aws.StringSlice(clusterNames),
	})
	if err != nil {
		log.Fatalln("Couldn't read instance status", err.Error())
		return false
	}

	for _, cluster := range desc.Clusters {
		if *cluster.RegisteredContainerInstancesCount != int64(clusterSizes[*cluster.ClusterName]) {
			return false
		}
	}

	return true
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
