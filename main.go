package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	cloudConfig, err := ioutil.ReadFile("cloud-config.yaml")
	if err != nil {
		fmt.Println("Couldn't read cloud-config", err.Error())
		return
	}

	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		fmt.Println("Couldn't create session", err.Error())
		return
	}

	// Create EC2 service client
	svc := ec2.New(session)

	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:        aws.String("ami-00eb20669e0990cb4"),
		InstanceType:   aws.String("t2.micro"),
		KeyName:        aws.String("COMSM0010"),
		MinCount:       aws.Int64(1),
		MaxCount:       aws.Int64(1),
		SecurityGroups: aws.StringSlice([]string{"comsm0010-cloud-open"}),
		UserData:       aws.String(base64.StdEncoding.EncodeToString(cloudConfig)),
	})

	if err != nil {
		fmt.Println("Could not create instance", err)
		return
	}

	fmt.Println("Created instance", *runResult.Instances[0].InstanceId)

	// result, err := worker.CalculateGoldenNonce("COMSM0010cloud", uint32(0), ^uint32(0), 11)
	// if err != nil {
	// 	fmt.Printf("Error calculating nonce: %s\n", err.Error())
	// 	os.Exit(1)
	// 	return
	// }
	// fmt.Printf("Golden nonce is: %d, for hash: %s\n", result.Nonce, result.Hash)
}
