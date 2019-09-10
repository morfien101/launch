#!/bin/bash
# Use "./build_and_test.sh" to run only the tests
# Use "./build_and_test.sh gobuild" to also rebuild the go project

if [[ $1 = "gobuild" ]]; then
  echo "Building the go project!"
  curdir=$(pwd)
  echo "Building launch"
  CGO_ENABLED=0 go build -a -installsuffix cgo -o launch .
  cd testbin
  echo "Building testbin"
  CGO_ENABLED=0 go build -a -installsuffix cgo -o testbin .
  cd $curdir
  echo "Finished building"
else
  echo "Skipping go build"
fi

docker build -t morfien101/launch-test:latest -f Dockerfile.test .
docker build -t morfien101/launch-test:debug -f Dockerfile.debug .

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