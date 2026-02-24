# Launch

A lightweight process manager designed to run as PID 1 in Docker containers.

Launch lets you run multiple processes in a single container with centralized logging, ordered startup hooks, and graceful signal propagation.

## Features

- Run **multiple processes** in one container.
- **Secret collection** runs first and injects output as environment variables for all later processes.
- **Init processes** run sequentially after secrets — any failure aborts startup.
- **Main processes** run in parallel; if one exits, Launch sends SIGTERM to all remaining ones and waits for shutdown.
- Ship logs to **console, file, syslog, or /dev/null** independently per process.
- **Templated YAML config** — inject environment variables and use built-in helper functions.

## How it works

```
Startup sequence:
  1. Read and render config file (first pass — container env vars available)
  2. Run secret processes → capture stdout JSON and inject as env vars
  3. Re-render config file (second pass — secret env vars now available)
  4. Run init processes sequentially — any non-zero exit aborts startup
  5. Start main processes in parallel
  6. Wait — when any main process exits, send SIGTERM to all others and shut down
```

## Processes

Launch supports three types of processes, each with a distinct role.

### Secret processes

Run before anything else. They collect credentials or other secrets and expose them as environment variables. Because full logging is not yet available at this stage, output goes to the console.

### Init processes

Run sequentially after secrets. Use these for setup tasks — running migrations, fetching config, preparing the filesystem. All init processes must succeed before main processes start.

### Main processes

The long-running processes in your container. They start in parallel and are expected to run for the lifetime of the container. If any one of them exits, Launch sends SIGTERM to all the others.

## Configuration

The configuration is driven by a YAML file. Run the following to generate an annotated example:

```bash
./launch -example-config
```

Full documentation:

- [Configuration reference](./READMEs/ConfigurationFile.md) — all sections and options
- [Logging engines](./READMEs/Logging.md) — console, file, syslog, `/dev/null`
- [Secrets](./READMEs/Secrets.md) — how secret collection works

## Contributing

This project is still in active development. There is a simple build script for integration testing.

```bash
# Run tests only
./build_and_test.sh

# Rebuild the binary and run tests
./build_and_test.sh gobuild
```
