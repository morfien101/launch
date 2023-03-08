// testbin is used to test the process manager and is not
// included in the building of launch
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/silverstagtech/randomstring"
)

const (
	//VERSION is the used to display a version number
	VERSION = "0.1.0"
)

var (
	spamOutFlag        = flag.Bool("spam", false, "Send progressivly more out to STDOUT. Use with -stdout and -stderr")
	spamSizeFlag       = flag.Uint("spam-size", 0, "Set size for spam message. If not set they just grow slowly.")
	turnOnSTDOUTFlag   = flag.Bool("stdout", false, "enabled stdout spamming.")
	turnOnSTDERRFlag   = flag.Bool("stderr", false, "enabled stderr spamming.")
	noEnvFlag          = flag.Bool("no-env", false, "Don't display the environment variables.")
	noNewLineFlag      = flag.Bool("no-newline", false, "Remove new line from stdout and stderr output.")
	echoFlag           = flag.String("id", "", "Prints this on execution")
	timeoutSecondsFlag = flag.Int("timeout", 10, "How long to wait before dying in seconds.")
	exitWithFlag       = flag.Int("exit-with", 0, "Exit with the specified exitcode.")
	ignoreSignalsFlag  = flag.Bool("ignore-signals", false, "Ignore the signals that the process gets.")
	logJSONFlag        = flag.Int("log-json", 0, "Log some random json messages. The number says how many logs you want.")

	addTestEnv = flag.Bool("send-env", false, "returns a test environment variable: 'LUANCH_TEST=LIFTOFF'")

	helpFlag    = flag.Bool("h", false, "Show the help menu")
	versionFlag = flag.Bool("v", false, "Displays a version number.")
)

func main() {
	flag.Parse()
	if *helpFlag {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *versionFlag {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	// We check to see if we need to add an environment variable for secrets.
	// If so we need to do this and exit with no output to stdout.
	if *addTestEnv {
		fmt.Println(`{"LAUNCH_TEST":"LIFTOFF"}`)
		time.Sleep(time.Second * time.Duration(*timeoutSecondsFlag))
		os.Exit(0)
	}

	// Start
	log.Printf("Starting %s version %s", os.Args[0], VERSION)

	signals := make(chan os.Signal, 1)
	timeout := make(chan bool, 1)
	done := make(chan string, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if len(*echoFlag) != 0 {
		fmt.Println(*echoFlag)
	}

	fmt.Printf("got arguments: %s\n", os.Args)

	if !*noEnvFlag {
		fmt.Println("Below is the environment variables that I can see.")
		for _, env := range os.Environ() {
			fmt.Println(env)
		}
	}

	fmt.Println("Waiting for a signal or timeout...")
	go func() {
		for {
			select {
			case signal := <-signals:
				msg := fmt.Sprintf("Got signal %s", signal)
				if *ignoreSignalsFlag {
					fmt.Println(msg, "but, told to ignoring it.")
					continue
				}
				done <- fmt.Sprintln(msg)
			case <-timeout:
				done <- fmt.Sprintln("Timed out")
			}
			return
		}
	}()
	time.AfterFunc(time.Second*time.Duration(*timeoutSecondsFlag), func() {
		timeout <- true
	})

	if *spamOutFlag {
		if *turnOnSTDOUTFlag {
			fmt.Println("Starting STDOUT spam generator...")
			go spammer("STDOUT", *spamSizeFlag, *noNewLineFlag, os.Stdout)
		}
		if *turnOnSTDERRFlag {
			fmt.Println("Starting STDERR spam generator...")
			go spammer("STDERR", *spamSizeFlag, *noNewLineFlag, os.Stderr)
		}
	}

	if *logJSONFlag > 0 {
		for i := 0; i < *logJSONFlag; i++ {
			log, err := generateJSONLog()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(log)
			}
		}
	}

	fmt.Println(<-done)

	if *exitWithFlag > 0 {
		os.Stderr.WriteString(fmt.Sprintf("Exiting with special exit code %d.", *exitWithFlag))
	}
	os.Exit(*exitWithFlag)
}

func generateJSONLog() (string, error) {
	r, _ := randomstring.Generate(4, 4, 4, 4, 64)
	output := struct {
		Name       string `json:"name"`
		Timestamp  string `json:"time_stamp"`
		Severity   string `json:"level"`
		SomeRandom string `json:"some_random"`
		SomeStatic string `json:"some_static"`
	}{
		Name:       "container-bootrapper testbin",
		Timestamp:  time.Now().String(),
		Severity:   "crit",
		SomeStatic: "look_for_me",
		SomeRandom: r,
	}
	b, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf(`{"msg":"Error generating log","error":"%s"}`, err), nil
	}

	return string(b), nil

}
