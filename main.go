package main

import (
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

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/internallogger"
	"github.com/morfien101/launch/processlogger"
	"github.com/morfien101/launch/processmanager"
)

const (
	// DefaultTimeout period for binaries
	DefaultTimeout = 30
)

var (
	// version and timestamp are expected to be passed in at build time.
	buildVersion   = "0.1.0"
	buildTimestamp = ""

	timeout = DefaultTimeout
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

	pmlogger.Println("Starting full loggers")

	// Start logging engines
	loggers = processlogger.New(10, config.DefaultLoggerConfig)
	err = loggers.StartLoggers(config.Processes, config.ProcessManager.LoggerConfig)
	if err != nil {
		fmt.Println(err)
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
	pm := processmanager.New(config.Processes, loggers, pmlogger, signals)
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
	pmlogger.Println("Final state: " + endMessage)

	// Shutdown the loggers.
	terminate(0, loggers)
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
