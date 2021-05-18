// Package processlogger is responsible for collecting logs that are submitted and
// forwarding them to the correct logging engine. The logging engines need to satisfiy
// the Logger interface.
// Loggers handle themselves and processlogger forwards on the log messages that they
// need to send.
// processlogger will only start the logging engines that it requires although all
// logging engines are available.
package processlogger

import (
	"fmt"
	"sync"

	"github.com/morfien101/launch/configfile"
)

// Pipe describes if the message came out of stdout or stderr
type Pipe string

const (
	// STDERR is used to indicate a message was from stderr
	STDERR = Pipe("e")
	// STDOUT is used to indicate a message was from stdout
	STDOUT = Pipe("o")
	// logBufferSize is how many logs can be in queue for each logging end point before we
	// start dropping messages
	logBufferSize = 100
)

// LogMessage is the box that needs to be created to ship a message to a
// log forwarder.
type LogMessage struct {
	Source  string
	Pipe    Pipe
	Config  configfile.LoggingConfig
	Message string
}

// LogManager is used to collect, route and submit logs to the correct logging engines.
type LogManager struct {
	logChan          chan LogMessage
	workersList      []*logworker
	waitgroup        sync.WaitGroup
	defaultConfig    configfile.DefaultLoggerDetails
	availableLoggers map[string]Logger
	activeLoggers    map[string]Logger
	activeLoggerQ    map[string]chan *LogMessage
	terminated       bool
}

// New will create a new LogManager
func New(loggingBuffer int, defaultConfig configfile.DefaultLoggerDetails) *LogManager {
	lm := &LogManager{
		logChan:       make(chan LogMessage, loggingBuffer),
		workersList:   make([]*logworker, 0),
		defaultConfig: defaultConfig,
		activeLoggers: make(map[string]Logger),
		activeLoggerQ: make(map[string]chan *LogMessage),
	}
	lm.loadAvailableLoggers()
	return lm
}

func (lm *LogManager) loadAvailableLoggers() {
	lm.availableLoggers = make(map[string]Logger)
	// Load all the available loggers here
	for name, regfunc := range registeredLoggers {
		lm.availableLoggers[name] = regfunc()
	}
}

// StartLoggers will start all of the required loggers for this process
func (lm *LogManager) StartLoggers(processes configfile.Processes, PMConf configfile.LoggingConfig) error {
	// Start the logger to the process manager itself.
	// If we can't log ourselves then we need to error.
	if err := lm.startLogger(PMConf); err != nil {
		return err
	}

	// Start the loggers for each process.
	setup := func(pSlice []*configfile.Process) error {
		for _, proc := range pSlice {
			err := lm.startLogger(proc.LoggerConfig)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// If we can't start the loggers then we need to error out.
	if err := setup(processes.InitProcesses); err != nil {
		return err
	}
	if err := setup(processes.MainProcesses); err != nil {
		return err
	}

	// Now that we have a list of the loggers that are going to be used.
	// We can start logger and start the router worker for the logger.
	for id, logger := range lm.activeLoggers {
		err := logger.Start()
		if err != nil {
			return err
		}
		lm.startLogRouter(id, logger)
	}

	return nil
}

func (lm *LogManager) startLogRouter(id string, logger Logger) {
	c := make(chan *LogMessage, logBufferSize)
	lm.activeLoggerQ[id] = c

	worker := newWorker(logger)
	lm.workersList = append(lm.workersList, worker)

	go worker.route(c)
}

func (lm *LogManager) startLogger(conf configfile.LoggingConfig) error {
	if _, ok := lm.availableLoggers[conf.Engine]; !ok {
		return fmt.Errorf("logging engine %s is not recognized. Please check your configuration file", conf.Engine)
	}

	// We register the configuration for the loggers here.
	// We need to tell the loggers to start later.
	err := lm.availableLoggers[conf.Engine].RegisterConfig(conf, lm.defaultConfig)
	if err != nil {
		return err
	}
	if lm.activeLoggers[conf.Engine] != lm.availableLoggers[conf.Engine] {
		lm.activeLoggers[conf.Engine] = lm.availableLoggers[conf.Engine]
	}
	return nil
}

// Submit is used to push log messages in to the router queue.
// If the queue is full we will drop the message.
func (lm *LogManager) Submit(log LogMessage) {
	if lm.terminated {
		// We should send a metric or something here to show that we ditched
		// a log.
		return
	}
	select {
	case lm.activeLoggerQ[log.Config.Engine] <- &log:
		// We should also possibly log metrics for successful logs in queue
	default:
		// TODO
		// We should be logging metrics here for each message that we have dropped.
		errMsg := fmt.Errorf("can't log to %s because it is overflowing with logs. Log is from: %s", log.Config.Engine, log.Source)
		fmt.Println(errMsg)
	}
}

// Shutdown is used to gracefully shutdown all the loggers and log routers.
func (lm *LogManager) Shutdown() []error {
	// There may be some lagging go funcs that are sending log messages.
	// Unfortunately we can't know for sure. So we can only assume that
	// the point we have called shutdown, all the important logs have arrived.
	// Mark the log manager as terminated and that we can't accept more
	// logs from this point on.
	lm.terminated = true

	// Drain the queues
	//close the channels for the loggers
	for _, ch := range lm.activeLoggerQ {
		close(ch)
	}
	// Collect the waitgroups. Wait for each one to
	// free up
	for _, worker := range lm.workersList {
		worker.waitGroup().Wait()
	}

	// Each logger needs to shutdown.
	// So we need to loop through all the active loggers and call the shutdown function.
	// Shutdown returns a channel that gets the error if there is one.
	outputs := make(map[string]error)
	mu := sync.Mutex{}
	wg := &sync.WaitGroup{}
	for id, logger := range lm.activeLoggers {
		innerID := id
		innerLogger := logger
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case err, ok := <-innerLogger.Shutdown():
				if !ok {
					// got no error. Safe to return
					return
				}
				mu.Lock()
				outputs[innerID] = err
				mu.Unlock()
			}
		}()
	}
	// We can then wait for all of them to complete using the wait group.
	wg.Wait()
	// Then check to see if they returned any errors and pass that back to the caller.
	// We return a list of errors because we are responsible for closeing multiple loggers.
	errList := make([]error, 0)
	for key, err := range outputs {
		if err != nil {
			errList = append(errList, fmt.Errorf("Logger %s got an error on shutdown call. Error: %s", key, err))
		}
	}

	// return once we are done
	return errList
}

// The logworker is used to route logs into the logger that they want to use.
type logworker struct {
	myLogger Logger
	wg       *sync.WaitGroup
}

func newWorker(logger Logger) *logworker {
	return &logworker{
		myLogger: logger,
		wg:       &sync.WaitGroup{},
	}

}

// route is is the worker function. It will accept message as they come in on the input channel
// and call the Submit func for the logger that has been allocated to it.
func (lw *logworker) route(input chan *LogMessage) {
	lw.wg.Add(1)
	for {
		select {
		case log, ok := <-input:
			if !ok {
				// Queue is closed, no more logs expected therefore stopping.
				lw.wg.Done()
				return
			}
			// Push the message into the logger.
			// The submit is a hand over to the logger. It is responsible for the
			// log from this point on.
			lw.myLogger.Submit(*log)
		}
	}
}

// waitGroup gives the reference the sync wait group.
// This can be used to determine when this worker is finished.
// Its expected that the caller should also close the input channel to the worker
// to get the worker to free the wait group.
func (lw *logworker) waitGroup() *sync.WaitGroup {
	return lw.wg
}
