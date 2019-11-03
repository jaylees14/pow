#!/bin/sh

SERVICE=worker
IMAGE_REPO_URL=jaylees/comsm0010-worker

# Create a docker image and deploy it to docker cloud
docker build -t $SERVICE -f $SERVICE/Dockerfile $SERVICE 
docker tag $SERVICE:latest $IMAGE_REPO_URL:latest
docker push $IMAGE_REPO_URL:latest
