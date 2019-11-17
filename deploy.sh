#!/bin/sh

SERVICE=worker
IMAGE_REPO_URL=615057327315.dkr.ecr.us-east-1.amazonaws.com/jaylees/comsm0010-worker

# Create a docker image and deploy it to docker cloud
eval $(aws ecr get-login --no-include-email --region us-east-1)
docker build -t $SERVICE -f $SERVICE/Dockerfile $SERVICE 
docker tag $SERVICE:latest $IMAGE_REPO_URL:latest
docker push $IMAGE_REPO_URL:latest
