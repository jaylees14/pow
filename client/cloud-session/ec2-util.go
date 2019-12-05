package cloudsession

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func createSecurityGroup(session *session.Session, name string, ports []int64) (*string, error) {
	svc := ec2.New(session)

	// Check security group doesn't already exist
	desc, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupNames: []*string{
			aws.String(name),
		},
	})
	if err == nil {
		for _, group := range desc.SecurityGroups {
			if *group.GroupName == name {
				return group.GroupId, nil
			}
		}
	}

	sg, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		Description: aws.String(fmt.Sprintf("Security group for %s", name)),
	})
	if err != nil {
		return nil, err
	}

	_, err = authoriseSecurityGroupTCPIngress(session, sg.GroupId, ports)
	return sg.GroupId, err
}

func authoriseSecurityGroupTCPIngress(session *session.Session, securityGroupID *string, ports []int64) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	svc := ec2.New(session)
	ipPermissions := make([]*ec2.IpPermission, 0, len(ports))
	for _, port := range ports {
		ipPermissions = append(ipPermissions, &ec2.IpPermission{
			FromPort:   aws.Int64(port),
			ToPort:     aws.Int64(port),
			IpProtocol: aws.String("tcp"),
			IpRanges: []*ec2.IpRange{
				{
					CidrIp: aws.String("0.0.0.0/0"),
				},
			},
		})
	}

	return svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       securityGroupID,
		IpPermissions: ipPermissions,
	})
}

func createEC2Instances(session *session.Session, imageID string, count int64, config []byte, securityGroupID *string, iamRoleArn *string) (*ec2.Reservation, error) {
	svc := ec2.New(session)
	iamRole := ec2.IamInstanceProfileSpecification{
		Arn: iamRoleArn,
	}

	return svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:            aws.String(imageID),
		InstanceType:       aws.String("t2.micro"),
		KeyName:            aws.String("COMSM0010"),
		MinCount:           aws.Int64(count),
		IamInstanceProfile: &iamRole,
		MaxCount:           aws.Int64(count),
		SecurityGroupIds:   []*string{securityGroupID},
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
