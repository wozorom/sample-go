#!/usr/bin/env bash

rm ./main*

docker build -t sample-go .

echo '-------------------------------------------------------- Size of the image   vvvvvv'
docker images | grep sample-go
echo '-------------------------------------------------------- Size of the image   ^^^^^^'

cd ./deployments

docker-compose up
