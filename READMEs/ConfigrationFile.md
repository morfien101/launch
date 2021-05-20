# Configuration File

Configuration is used to tell Launch what to do. 

It is used to determine where logs go, what processes run and what data needs to be collected.

An example of the configuration file can be obtained by running:

```bash
# Get example configuration file
./launch -example-config
```

Below is each section of the configuration with the relevant values and details. If you would like more information regarding the configuration all data is layed out in the [configfile folder](../configfile).

## Understanding double rendering

Configuration files have templating built in, see later templating section. This allows for environment variables to be used in the configuration file.
However many of those environment variables will be collected during the secrets and parameters collection phase.

The configuration file is rendered twice when the Launch is started.
The first render will be done on start up, it will produce the configuration with the available environment variables as the container starts.
The Launch will then proceed to collect secrets and parameters. Once complete the configuration is rendered a second time.
This allows the configuration to make use of parameters and secrets as part of the configuration. Previously blanked environment settings will now be filled in.

An example use case is using an override for hostname in the syslog logging configuration, or setting a environment variable.

Normal templating rules will be followed when using both renders. This means that you can still use the environment variables for secret collection but, they __MUST__ be available when the container starts. Eg. making use of docker -e|--env flags.
This also means that any templating that makes use of the `required` and `env` functions together __MUST__ be available on the first render or the configuration will fail to render and cause the Launch to stop.

Secrets are only collected once at startup.

## process_manager

`process_manager` configures the Launch process itself. It needs to know where to send it's logs and also if it needs to enable debug logging.

Example:

```yaml
# Process manager is Launch itself
process_manager:
  # logging_config is the logging the Launch process needs to use.
  logging_config:
  # This section contains a Logging config _see below_
  # debug_logging will toggle on and off the debug logger inside
  # the Launch
  debug_logging: (true|false)
  # Debug options should be off by default. Use them only if you need to.
  debug_options:
    # Prints the configuration that will be used for running processes. This happens after secrets are collected
    # and the second config rendering has taken place.
    show_generated_config: (true|false)
```

## processes

`processes` tells the Launch what start and where to send the logs. There is 3 sections here: secret_processes, init_processes and main_processes

Example:

```yaml
# processes tells Launch what to start and how.
processes:
  # Secret processes start before logging so use the console logger.
  # Secrets can export to environment variables and are the only process
  # that can share environment variables.
  secret_processes:
  - name: Process1
    # Path to executable
    command: /example/bin1
    # List of arguments
    arguments:
    - --arg1
    - two
    # How long to allow the process to run. This helps with run away processes.
    termination_timeout_seconds: 60
    # skip stops this execution if required.
    # Use the template functions to set this value.
    skip: false
  # init_processes start after secrets and run sequentially
  # This is a list and can have as many items as required.
  init_processes:
    # name is a descriptive name for the process.
  - name: first init processes
    # command is the full path to executable
    command: /binary/to/execute
    # List of arguments to pass onto the executable
    arguments:
      # Strings, strings with spaces, and numbers are accepted
      # Add as many as you need
    - -c
    - /tmp/text.txt
    - 10
    - words and more
    # termination_timeout_seconds how long the grace period is for processes to terminate.
    # The default is 1 second.
    termination_timeout_seconds: 3
    # logging_config is used to forward on the logs from this process.
    logging_config:
      # This section contains a Logging config _see below_
  # main_processes looks exactly the same as init_processes.
  main_processes:
  - name: first main
    command: /binary/to/execute
    arguments:
    - -a
    - 1
    - -b
    - this and that
    logging_config:
      # This section contains a Logging config _see below_
  - name: second main
    command: /another/binary/to/execute
    arguments:
    - -a
    - 1
    - -b
    - foo and bar
    logging_config:
      # This section contains a Logging config _see below_
```

## default_logger_config

`default_logger_config` is a section that allows you to put in any defaults that you don't want to repeat.
It contains a `logging_config` but is intended to allow for all configuration for all engines to be defined here.

Example:

```yaml
default_logger_config:
  logging_config:
  # This section contains a Logging config _see below_
```
## Logging

Logging is used to tell the Launch to send logs to a logging engine eg. syslog, console, etc...

Below is the logging configuration. However this is a slightly different one to the rest.

In this configuration you put the details for the logging engine for the process and then select which logging engine you want to make use of. These will appear in all processes.

valid engine names:

- console
- devnull
- syslog
- logfile

Example:

```yaml
logging_config:
  # Engine is where the logs should go
  engine: any valid engine name from above
  # descriptive name for your binary. Shipped to logging engine if possible.
  process_name: amazing_project
  # Only one of the below is required when used on a process.
  # Normally the one that is related the engine selected.
  # Syslog and file logger both require extra config as below.
  syslog:
    # Overrides the process name with this value. Can be defaulted so all processes that
    # don't have a value set will get this one.
    # If your process does have this or a name you will get the hostname which is ugly.
    # So have at least one of them set.
    program_name: syslog program name
    # Tries to read the log level if specific condition are met
    extract_log_level: (true|false)
    # Override the hostname sent to papertrail
    override_hostname: hostname_to_use
    # These both add the containers hostname to the end of the value they control.
    # remember that you need to add your own separator.
    append_container_name_to_tag: (true|false)
    append_container_name_to_hostname: (true|false)
  file_config:
    # filepath is where to store these logs
    filepath: /var/logs/process_name.log
    # size_limit is how large the file can be before rotation happens.
    size_limit: 100mb
    # historical_files_limit is how many files are to be kept.
    historical_files_limit: 3
```

## Template Functions

Template functions are used to make the configuration files somewhat dynamic. They allow values to be put in place at boot time of the process. Reading and rendering the configuration file is the very first thing Launch will do. Therefore if the configuration does not render correctly then it will not start.

Below is a list of the functions available to you and an example of the syntax used.

`hint: it's just golang's template syntax`

Function | Description | Example
---|---|---
env | Sets the value to an available Environment Variable | {{ env "ENVIRONMENT" }}
default | Use this default value if function fails | {{ default .NonExisting "default value" }}
required | This value must be satisfied or the launch will fail | {{ required (env "ALWAYS_THERE") }}
zerolen | Allows you to check if a value is zero length, returns the value for true or false | {{ zerolen (env "SOMETHING") "true value" "false value" }}

Example in configuration file.

```yaml
secrets:
  parameter_store:
  - key_path: {{ env "SECRET_PATH" }}
    recursive_lookup: true
    with_decryption: true
    skip: {{ zerolen (env "SECRET_PATH") "true" "false" }}
```
