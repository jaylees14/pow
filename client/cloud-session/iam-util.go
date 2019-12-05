package cloudsession

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

func createIamRole(session *session.Session, name string, roleTrustJSON string) (*iam.InstanceProfile, error) {
	svc := iam.New(session)
	instanceProfileName := fmt.Sprintf("%s-profile", name)

	// check if profile exists
	profile, err := svc.GetInstanceProfile(&iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(instanceProfileName),
	})
	if err == nil {
		return profile.InstanceProfile, nil
	}

	// Check if role already exists
	roleOutput, err := svc.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	role := roleOutput.Role

	if err != nil {
		// Create role
		createOutput, err := svc.CreateRole(&iam.CreateRoleInput{
			AssumeRolePolicyDocument: aws.String(roleTrustJSON),
			Path:                     aws.String("/"),
			RoleName:                 aws.String(name),
		})
		if err != nil {
			return nil, err
		}
		role = createOutput.Role
	}

	// Create profile
	_, err = svc.CreateInstanceProfile(&iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(instanceProfileName),
	})
	if err != nil {
		return nil, err
	}

	// Attach role to profile
	_, err = svc.AddRoleToInstanceProfile(&iam.AddRoleToInstanceProfileInput{
		RoleName:            role.RoleName,
		InstanceProfileName: aws.String(instanceProfileName),
	})
	if err != nil {
		return nil, err
	}

	// Refetch profile after role attached
	instanceProfile, err := svc.GetInstanceProfile(&iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(instanceProfileName),
	})

	return instanceProfile.InstanceProfile, err
}

func attachIamPermissions(session *session.Session, roleName *string, roles []string) error {
	svc := iam.New(session)

	for _, role := range roles {
		_, err := svc.AttachRolePolicy(&iam.AttachRolePolicyInput{
			PolicyArn: aws.String(role),
			RoleName:  roleName,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
