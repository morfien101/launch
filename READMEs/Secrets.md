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
To over come this, Launch will read a key value pair in JSON as the stdout of your process and expose those as environment variables for you.

Output expected:

```json
{"key":"value", "key2":"value2"}
```

It is also possible that you process writes to files that other processes can collect. It is expected that no output to STDOUT is given in this case.

With this being said here are the requirements for Secret Processes:

1. They must complete successfully to continue execution.
1. They are executed sequentially as shown in the configuration file.
1. output must be valid JSON and the only output on STDOUT.

Secrets might not always need to be collected. Consider if you are using in a Dev environment.
Make use of the `skip` field to stop a process from running.
You can determine the value by using one of the templating functions.

