# Launch

A simple runtime-agnostic process manager that eases Docker services startup and logging.
Consider it a half way house between Kubernetes and Docker.

Launch is expected to be process 1 in a container. It allows you to watch other processes in your containers and when they are all finished it will finish allowing containers to stop correctly.

## What can Launch help with

Launch is designed to be a process manager with simple but powerful logging. It borrows some ideas from kubernetes without having to deploy a kubernetes stack.

* You can run multiple processes in a single container.
* You can ship logs from processes to different logging engines.
* You can run init processes that run before your main processes. This allows you to collect artifacts, secrets or just setup an environment.
* A single main process dying will bring down a container, gracefully shutting down the other applications.

## Configuration

More details can be found in the [README Folder](./READMEs/)

## Processes

Launch has 3 processes types when running.

### Secret processes

These are used to collect secret data like username and passwords at startup.

### Init processes

These are used to configure the state of the container or collect more resources that don't need to be exported to environment variables.

### Main Processes

These are the long running processes in your containers.

## Configuration

The configuration YAML file is the driving force behind Launch. The configuration file will tell Launch what processes to run with what arguments, where to send logs.

The configuration file has a templating feature that allows you to make the configuration dynamic.

The configuration file has many sections that are documented in the
[README dedicated to configuration](./READMEs/ConfigurationFile.MD).

## Contributing

This project is still very much in active development. Expect changes and improvements.

There is a simple build script that does the current integration testing.

Use `./build_and_test.sh` to run only the tests
Use `./build_and_test.sh` gobuild" to also rebuild the go project
