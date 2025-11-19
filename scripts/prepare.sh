#!/bin/bash

docker build --platform linux/amd64 -t tonysoprano228/go-server .
docker push tonysoprano228/go-server
