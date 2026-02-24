# Secret Collection Engines

Secrets are collected before anything else has a chance to start.
This is because the loggers often need credentials to send the logs onwards.
Due to this logging is done to the console of the container itself, because we can not be sure we would be able to actually send the log messages on.

Launch allows you to run your own binary/command to collect secrets within the containers. Doing this means that you can use any secret management system you can write code for.

## Secret Processes

A secret process is used to go collect secrets. These could be in AWS Secret Manager, Hashicorp Vault or some inhouse secret manager.
The idea is that you can create your own binary to use to collect secrets and have Launch inject them into the environment.

```text
Because Launch is the parent process, child processes CAN NOT update the environment of the parent.
Therefore your process can not expose environment variables for later processes to see.
```
To overcome this, Launch reads the stdout of your process as a JSON key/value object and exposes those as environment variables.

Output expected:

```json
{"key":"value", "key2":"value2"}
```

It is also possible that your process writes to files that other processes can read. In this case no output to stdout is expected.

Requirements for secret processes:

1. They must exit successfully (exit code 0) to allow execution to continue.
1. They are executed sequentially in the order listed in the configuration file.
1. If they produce stdout, it must be a single valid JSON object and nothing else.

Secrets might not always need to be collected. Consider if you are using in a Dev environment.
Make use of the `skip` field to stop a process from running.
You can determine the value by using one of the templating functions.

