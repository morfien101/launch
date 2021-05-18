package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	// Pull in all available loggers.

	_ "github.com/morfien101/launch/processlogger/allloggers"
	"github.com/morfien101/launch/signalreplicator"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/internallogger"
	"github.com/morfien101/launch/processlogger"
	"github.com/morfien101/launch/processmanager"
)

var (
	// version and timestamp are expected to be passed in at build time.
	buildVersion   = "0.4.0"
	buildTimestamp = ""
)

func main() {
	flagHelp := flag.Bool("h", false, "Shows this help menu.")
	flagVersion := flag.Bool("v", false, "Shows the version.")
	flagVersionExtended := flag.Bool("version", false, "Shows extended version numbering.")
	flagConfigExample := flag.Bool("example-config", false, "Displays and example configration.")
	flagConfigFilePath := flag.String("f", "/launch.yaml", "Location of the config file to read.")
	// Parse and process terminating flags
	flag.Parse()
	if *flagHelp {
		flag.PrintDefaults()
		return
	}
	if *flagVersion {
		fmt.Println(buildVersion)
		return
	}
	if *flagVersionExtended {
		fmt.Printf("Version: %s\nBuild time: %s\nGo version: %s\n", buildVersion, buildTimestamp, runtime.Version())
		return
	}
	if *flagConfigExample {
		out, err := configfile.ExampleConfigFile()
		if err != nil {
			fmt.Printf(`There was an error generating the configuration file example.
Please log an error with the maintainer.
The error was: %s`, err)
			os.Exit(1)
		}
		fmt.Println(out)
		return
	}

	// Setup signal capture
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for receivedSignal := range signals {
			// Signals can come from inside and outside. The signals can come from the user/deamon running the container
			// or it can come from a process termination. In either case we need to send the signals to the replicator
			// to forward it onto the running processes.
			signalreplicator.Send(receivedSignal)
		}
	}()

	// Create a limited logger that will be thrown away once we fired up our actual loggers.
	loggers := processlogger.New(
		10,
		configfile.DefaultLoggerDetails{},
	)
	starterPMConfig := configfile.LoggingConfig{
		Engine: "console",
	}
	err := loggers.StartLoggers(configfile.Processes{}, starterPMConfig)
	if err != nil {
		fmt.Println(err)
	}
	pmlogger := internallogger.New(starterPMConfig, loggers)

	config, err := configfile.New(*flagConfigFilePath)
	if err != nil {
		pmlogger.Errorf("Failed to render the configuration. Error: %s", err)
		terminate(1, loggers)
	}

	// Collect secrets
	// This can only use the temp logger because we can't yet start the
	// the full loggers. Some of them will require secrets from the collection
	// about to take place.
	pmlogger.Println("Attempting to collect secrets")
	err = collectSecrets(config.Processes.SecretProcess, pmlogger)
	if err != nil {
		pmlogger.Errorf("Failed to collect secrets. Error: %s\n", err)
		terminate(1, loggers)
	}

	// Render config again with secret values included
	pmlogger.Println("Rendering configuration again with secrets in place")
	config, err = configfile.New(*flagConfigFilePath)
	if err != nil {
		pmlogger.Errorf("Failed to recreate the configuration. Error: %s", err)
		terminate(1, loggers)
	}

	pmlogger.Println("Starting full loggers")

	// Start logging engines
	loggers = processlogger.New(10, config.DefaultLoggerConfig)
	err = loggers.StartLoggers(config.Processes, config.ProcessManager.LoggerConfig)
	if err != nil {
		pmlogger.Errorf("Could not start full logging. Error: %s", err)
		// Attempt to close what has been opened.
		terminate(1, loggers)
	}

	// Start the internal logger now that we know where to log to
	pmlogger = internallogger.New(config.ProcessManager.LoggerConfig, loggers)
	pmlogger.DebugOn(config.ProcessManager.DebugLogging)
	pmlogger.Debugln("Debugging logging for the process manager has been turned on")
	if config.ProcessManager.DebugOptions.PrintGeneratedConfig {
		pmlogger.Debugf("Using generated config:\n%s", *config)
	}

	// Get a new proccess manager
	pm := processmanager.New(config.Processes, loggers, pmlogger)

	// Start init processes in order one by one
	if output, err := pm.RunInitProcesses(); err != nil {
		pmlogger.Errorf("An init process failed. Error: %s\n", err)
		pmlogger.Println(output)
		terminate(1, loggers)
	}

	// Start processes
	wait, err := pm.RunMainProcesses()
	if err != nil {
		pmlogger.Errorf("Something went wrong starting the main processes. Error: %s", err)
		terminate(1, loggers)
	}

	// Wait for processes to finish
	pmlogger.Debugln("Waiting for main processes to finish.")
	endMessage := <-wait
	pmlogger.Debugln("Finished waiting. Proceeding to shutdown loggers.")
	pmlogger.Println(endMessage)

	// Shutdown the loggers.
	terminate(0, loggers)
}

func collectSecrets(secretConfig []*configfile.SecretProcess, pmlogger *internallogger.InternalLogger) error {
	if len(secretConfig) == 0 {
		return nil
	}

	// Collect the secrets for each secret process.
	// Set the secrets after each completed process as following processes could rely on them.
	for _, secretProc := range secretConfig {
		if secretProc.Skip {
			continue
		}
		stdout, stderr, err := processmanager.RunSecretProcess(*secretProc, pmlogger)
		if err != nil {
			newErr := fmt.Errorf(
				"there was an error collecting the secrets from %s. STDERR: %s. Internal Error: %s",
				secretProc.Name,
				stderr,
				err,
			)
			return newErr
		}
		procsSecrets, err := convertSecretOutput(stdout)
		if err != nil {
			return fmt.Errorf("failed to decode the secrets from %s. Error: %s", secretProc.Name, err)
		}
		if err := addEnvVars(procsSecrets); err != nil {
			return err
		}
	}
	return nil
}

func addEnvVars(envValues map[string]string) error {
	for key, value := range envValues {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func convertSecretOutput(input string) (map[string]string, error) {
	output := map[string]string{}
	err := json.Unmarshal([]byte(input), &output)
	return output, err
}

// terminate will flush the loggers and then exit with the passed in code.
// If the loggers fail then we have no choice but to spit to the console.
func terminate(exitcode int, loggers *processlogger.LogManager) {
	// Shutdown the loggers.
	if errs := loggers.Shutdown(); len(errs) > 0 {
		errString := func() string {
			es := []string{}
			for _, err := range errs {
				es = append(es, err.Error())
			}
			return strings.Join(es, ",")
		}
		log.Fatalf("Error shutting down loggers. Errors: %s" + errString())
	}

	os.Exit(exitcode)
}
