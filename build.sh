#!/bin/bash

docker build -t $1 -f $1/Dockerfile $1
docker run $1
