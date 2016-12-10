#!/bin/sh

docker build -t=buptmiao/apigateway:v1.0.1 -f=Dockerfile.apigateway .
docker build -t=buptmiao/feed:v1.0.1 -f=Dockerfile.feed .
docker build -t=buptmiao/profile:v1.0.1 -f=Dockerfile.profile .
docker build -t=buptmiao/topic:v1.0.1 -f=Dockerfile.topic .