# Logging Engines Documentation

A more detailed description of the loggers that are available is documented below.

## Startup logging

The Launch needs to start logging before it is actually capable of logging fully. This is because some loggers require authentication that is expected to be collected by the secrets.
To overcome this the Launch will log to the console until after the secrets have been collected.
At which point it will read the configuration file and use the specified loggers.

## Default logging

Default values for logging can be set. These values will be used if there is not value present for a process or they will be merged. This allows most the configuration to be setup here, and only specifics to be set at the process level.

Processes will still need to select a logging engine.

## DevNull

DevNull is basically the same as /dev/null. Its a black hole for logs to go and never return.

Use this if you have an application that you simply don't care about the logging.

## Console

Console logging will forward out the logs it receives to the Launch STDOUT and STDERR. This is more useful for development environments where you don't want to forward your logs to a central logging platform.

## File Logging

File logging consumes the messages from your application via STDOUT and STDERR and forwards them to a file. The file is rotated on a regular basis to keep the container footprint small. The configuration values for rotation can be set with the configuration files.

File logging is only really useful in development environments. In most production environments the disks of the containers will be removed once the container is terminated. If you want to use this in production it is recommend that you link the volumes where the files are to be written.

## Syslog

Syslog is a pretty standard linux way of sending logs. These logs are sent as lines and multiline logs are unfortunetly split.

The logger allows you to override the name that you see in syslog. By default it will use the process name. If you set the `program_name` key under the syslog logging configuration it will use that in its place.

Due to the logs being presented to Launch via stdout we are not able to know if the log is critical, warning or informational. This information might be available in the text, however the Launch will simply forward on the message with out inspecting its contents by default.

If you set `extract_log_level: true` the logger will attempt to detect the level from your message. There are limitations here and your messages need to be structured correctly.

1. The logs `MUST` be in JSON format.
1. The logs `MUST` have have `level` key.
1. The `level` key `MUST` have one of the following values. [syslog wikipedia](https://en.wikipedia.org/wiki/Syslog#Severity_level)

* emerg
* alert
* crit
* err
* warning
* notice
* info
* debug

Example:

```json
{
    "name": "my_test",
    "msg": "This is a test log",
    "level": "crit"
}
```

The last point to think about is that large logs (excess of 1000 chars) may have a small impact on performance. This is compounded with rapid logging. Currently it takes about 0.003 Milliseconds to read a 500 char log. If you have hundreds of logs per second, Expect problems with detection.

Syslog logging reports the hostname of the host that sent the message. This is by default the hostname of the container. However for searching the hostname of the container can actually be useless and rather you would want the applications to send under a common hostname. This can be done by making use the `override_hostname` configuration. This can be set as a default logging variable and process level. The process level will win if both are set.
You can combine this with the configuration file templating to make names that reflect your environment such as `override_hostname: awesome_app_{{env NODE_NAME }}_{{env NODE_REGION}}`.

The normal rules for host names will apply. No special chars, no space, etc...

For easier tracking of the instances themselves you can append the containers hostname to either the process or to the hostname. If you do this you need to include any _ or - as it will simple tack on the hostname.
The can be set at the default configuration OR the process configuration.
Use `append_container_name_to_tag` and `append_container_name_to_hostname` to control these features.
