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
	spamout        = flag.Bool("spam", false, "Send progressivly more out to STDOUT. Use with -stdout and -stderr")
	turnOnSTDOUT   = flag.Bool("stdout", false, "enabled stdout spamming.")
	turnOnSTDERR   = flag.Bool("stderr", false, "enabled stderr spamming.")
	noenvflag      = flag.Bool("no-env", false, "Don't display the environment variables.")
	noNewLineFlag  = flag.Bool("no-newline", false, "Remove new line from stdout and stderr output.")
	echoFlag       = flag.String("id", "", "Prints this on execution")
	timeoutSeconds = flag.Int("timeout", 10, "How long to wait before dying in seconds.")
	exitBad        = flag.Int("exit-bad", 0, "Exit with an exitcode of 1 to indicate a bad exit code.")
	ignoreSignals  = flag.Bool("ignore-signals", false, "Ignore the signals that the process gets.")
	logjson        = flag.Int("log-json", 0, "Log some random json messages. The number says how many logs you want.")
	helpflag       = flag.Bool("h", false, "Show the help menu")
	versionflag    = flag.Bool("v", false, "Displays a version number.")
)

func main() {
	flag.Parse() 
	if *helpflag {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *versionflag {
		fmt.Println(VERSION)
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

	if !*noenvflag {
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
				if *ignoreSignals {
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
	time.AfterFunc(time.Second*time.Duration(*timeoutSeconds), func() {
		timeout <- true
	})

	if *spamout {
		fmt.Println("Starting spam generators...")
		if *turnOnSTDOUT {
			go spammer("STDOUT", *noNewLineFlag, os.Stdout)
		}
		if *turnOnSTDERR {
			go spammer("STDERR", *noNewLineFlag, os.Stderr)
		}
	}

	if *logjson > 0 {
		for i := 0; i < *logjson; i++ {
			log, err := generateJSONLog()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(log)
			}
		}
	}

	fmt.Println(<-done)

	if *exitBad > 0 {
		os.Stderr.WriteString(fmt.Sprintf("Exiting badly because of -bad-exit flag. Using exitcode %d", *exitBad))
		os.Exit(*exitBad)
	}
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
