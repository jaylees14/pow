# pow
COMSM0010 Cloud Computing Assignment: Horizontal Scaling

## Infrastructure:
- Use ecs optimized AMI image (starts daemon) - https://docs.aws.amazon.com/AmazonECS/latest/developerguide/launch_container_instance.html
- Ensure IAM role ecsInstanceRole is created and attached to EC2 - this has AmazonEC2ContainerServiceforEC2Role attached to it
- Ensure SG is created for ssh from any ports
- Ensure "jay" iam user is configured, that needs to be attached to client


Ports Exposed:
- 22 for SSH
- 8080 for cadvisor 
- 9090 for prom 
- 3000 for grafana 
- 2112 for worker to give extra metrics
