#!/bin/sh

pip install --user awscli
export PATH=$PATH:$HOME/.local/bin

SERVICES="worker grafana"
BASE_REPO_URL=615057327315.dkr.ecr.us-east-1.amazonaws.com/jaylees/comsm0010

eval $(aws ecr get-login --no-include-email --region us-east-1)

# Create a docker image and deploy it to ECR
for SERVICE in $SERVICES 
do
    REPO_URL=${BASE_REPO_URL}-${SERVICE}
    docker build -t $SERVICE -f $SERVICE/Dockerfile $SERVICE 
    docker tag $SERVICE:latest $REPO_URL:latest
    docker push $REPO_URL:latest
done
