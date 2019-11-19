package cloudsession

import (
	"encoding/base64"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

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
