#!/bin/sh

SERVICES="worker grafana"
BASE_REPO_URL=jaylees/comsm0010

# Create a docker image and deploy it to ECR
for SERVICE in $SERVICES 
do
    REPO_URL=${BASE_REPO_URL}-${SERVICE}
    docker build -t $SERVICE -f $SERVICE/Dockerfile $SERVICE 
    docker tag $SERVICE:latest $REPO_URL:latest
    # Requires DOCKER_USERNAME and DOCKER_PASSWORD to be present
    docker push $REPO_URL:latest
done
