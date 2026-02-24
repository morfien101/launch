#!/bin/bash
set -e

# Use "./build_and_test.sh" to run only the tests
# Use "./build_and_test.sh gobuild" to also rebuild the go project

if [[ $1 = "gobuild" ]]; then
  echo "Building the go project!"
  curdir=$(pwd)
  echo "Building launch"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o launch .
  cd testbin
  echo "Building testbin"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o testbin .
  cd $curdir
  echo "Finished building"
else
  echo "Skipping go build"
fi

# It's worth noting that distroless only supports linux, so we need to make sure we build for linux/amd64.
docker build --no-cache --platform linux/amd64 -t morfien101/launch-test:latest -f Dockerfile.test .
docker build --no-cache --platform linux/amd64 -t morfien101/launch-test:debug -f Dockerfile.debug .

echo "#########################"
echo "## Running full config ##"
echo "#########################"
docker run \
  -it \
  -v $(pwd)/launch.yaml:/launch.yaml \
  morfien101/launch-test:latest

#echo "############################"
#echo "## Running minimal config ##"
#echo "############################"
#
#docker run \
#  -it \
#  -v $(pwd)/launch_minimal.yaml:/launch.yaml \
#  morfien101/launch-test:latest