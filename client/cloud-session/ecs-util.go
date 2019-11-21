package cloudsession

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func createECSCluster(session *session.Session, name string) (*ecs.CreateClusterOutput, error) {
	svc := ecs.New(session)
	return svc.CreateCluster(&ecs.CreateClusterInput{
		ClusterName: aws.String(name),
	})
}

func createECSTask(session *session.Session, name string, containerDefinition *ecs.ContainerDefinition, volumes []*ecs.Volume, hostNetworkMode bool) (*ecs.RegisterTaskDefinitionOutput, error) {
	networkMode := "bridge"
	if hostNetworkMode {
		networkMode = "host"
	}

	svc := ecs.New(session)
	return svc.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{containerDefinition},
		Family:               aws.String(fmt.Sprintf("COMSM0010-%s-task", name)),
		Memory:               aws.String("400"),
		NetworkMode:          aws.String(networkMode),
		Volumes:              volumes,
	})
}

func startECSTask(session *session.Session, clusterName *string, taskName *string, count int64) (*ecs.RunTaskOutput, error) {
	svc := ecs.New(session)
	return svc.RunTask(&ecs.RunTaskInput{
		Cluster:        clusterName,
		Count:          aws.Int64(count),
		TaskDefinition: taskName,
	})
}

func startDaemonECSService(session *session.Session, clusterName *string, taskName *string, name string) (*ecs.CreateServiceOutput, error) {
	svc := ecs.New(session)
	return svc.CreateService(&ecs.CreateServiceInput{
		Cluster:            clusterName,
		SchedulingStrategy: aws.String("DAEMON"),
		ServiceName:        aws.String(fmt.Sprintf("COMSM0010-%s-service", name)),
		TaskDefinition:     taskName,
	})
}

func stopECSService(session *session.Session, clusterName *string, serviceName *string) (*ecs.DeleteServiceOutput, error) {
	svc := ecs.New(session)
	return svc.DeleteService(&ecs.DeleteServiceInput{
		Cluster: clusterName,
		Service: serviceName,
	})
}

func createMountPoint(sourceVolume string, containerPath string, readOnly bool) *ecs.MountPoint {
	return &ecs.MountPoint{
		SourceVolume:  aws.String(sourceVolume),
		ContainerPath: aws.String(containerPath),
		ReadOnly:      aws.Bool(readOnly),
	}
}

func createPortMapping(container int64, host int64) *ecs.PortMapping {
	return &ecs.PortMapping{
		ContainerPort: aws.Int64(container),
		HostPort:      aws.Int64(host),
	}
}

func createVolume(name string, hostPath string) *ecs.Volume {
	return &ecs.Volume{
		Name: aws.String(name),
		Host: &ecs.HostVolumeProperties{
			SourcePath: aws.String(hostPath),
		},
	}
}
