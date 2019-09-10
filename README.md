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

## TODO

features still in the back burner:

* [ ] delayed start on processes.
* [ ] health check processes - not sure how to do this yet
* [ ] restart failed main processes if configured
* [ ] rendering of configuration file trigger after init processes. To allow for secrets or configuration to be collected.

## Logging

Launch has the following statement in grained into its design which guides its actions on logging:

1. Development teams should not have to worry about where logs are going during creation of their projects.
1. A change to where logs end up should not result in projects having to refactor source code.
1. It is the responsibility of the developer to push all logging out to STDOUT and STDERR. Launch will collect and forward the logs onto the relevant logging engine defined by the configuration of the project.

Launch does however give the choice to the developer of where they would like the logs to end up. This does not contradict points 1 and 2 because the developer has the choice at run time rather than at development time.

Logging is important to all applications. However that importance does not trump the running of the service. Therefore all logging plugins will send logs in `Best Effort` mode.

Logging engines available:

1. Console
1. DevNull
1. File with rotation
1. Syslog

See
[Logging Documentation](./READMEs/Logging.MD)
for details on each logging engine.

The use of Launch's logging engines are optional. You could if you wanted to, setup filebeat in the container and read from files.
Use the console logger for any stray logs.
Another option is to use the file logger to feed the files that filebeat can watch.

## Processes

Launch has 2 processes types when running.

* Init processes
* Main Processes

### Init Processes

Initialization Processes are used to get the environment ready for the main processes to run.
Init processes are run sequentially in the order that they are defined in the configuration file.
These processes **MUST** finish successfully (exit code 0) for the next process to start. Only once all init processes are complete will the main processes start. If a single init process fails Launch will stop further processes from starting and terminate.

### Main Processes

Main processes are processes that need to run continually. They dictate the lifespan of the container. Multiple main processes can be run however currently all of then **MUST** be running in order for the container to be considered as healthy.

If a main process terminates for any reason then launch will send termination signals to all remaining main processes. It will give them a grace period to terminate after which it will forcefully terminate them. The grace period is configuration driven so you have can give the process the time it needs to wrap up any tasks. The default grace period is 30 seconds.

## Run time

Launch effectively acts as an init system. It does this to ease administration in container clusters by wrapping everything a application needs to run including any sidecar services for logging, metrics and data harvesting.

Launch is aware that it is not the first process and knows that it can be terminated by a controller of some kind. To this effect Launch will forward on selected signalling that it gets to the underlying processes.

Signals currently supported for forwarding:

* SIGTERM
* SIGINT

Launch is designed to work in a container no other signalling is expected.

## Configuration

The configuration YAML file is the driving force behind Launch. The configuration file will tell Launch what processes to run with what arguments, where to send logs and what tags to put on logs.

The configuration file has a templating feature that allows you to make the configuration dynamic.

The configuration file has many sections that are documented in the
[README dedicated to configuration](./READMEs/ConfigurationFile.MD).

## Contributing

This project is still very much in active development. Expect changes and improvements.

There is a simple build script that does the current integration testing.

Use `./build_and_test.sh` to run only the tests
Use `./build_and_test.sh` gobuild" to also rebuild the go project
