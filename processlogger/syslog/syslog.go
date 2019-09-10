package syslog

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
	syslogger "github.com/silverstagtech/srslog"
)

const (
	tlsConnection = "tcp+tls"
	tcpConnection = "tcp"
	udpConnection = "udp"
	// LoggerTag will be used to call this package
	LoggerTag       = "syslog"
	defaultProtocol = tlsConnection
)

var (
	validDialers = map[string]bool{
		tlsConnection: true,
		tcpConnection: true,
		udpConnection: true,
	}
)

func isValidDialer(s string) bool {
	_, ok := validDialers[s]
	return ok
}

func init() {
	processlogger.RegisterLogger(LoggerTag, func() processlogger.Logger {
		return &Syslog{
			protocol:        defaultProtocol,
			loggingFacility: syslogger.LOG_DAEMON,
		}
	})
}

// Syslog is responsible for logging to a syslog endpoint
type Syslog struct {
	location        string
	protocol        string
	tlsconfig       *tls.Config
	config          configfile.LoggingConfig
	defaults        configfile.DefaultLoggerDetails
	logwriter       *syslogger.Writer
	running         bool
	loggingFacility syslogger.Priority

	// hostname is the name that you want to appear in syslog message
	hostname string
	// basename is the containers hostname and can be appended to the hostname
	// then sending the log. Useful when you want to see what container is sending
	// the logs.
	basename string
}

// IsStarted will tell the caller if this logger needs to be started.
func (sl *Syslog) isStarted() bool {
	return sl.running
}

func (sl *Syslog) readCertificates() (*x509.CertPool, error) {
	if sl.defaults.Config.Syslog.CertificateBundlePath == "" {
		return nil, fmt.Errorf("No certificate bundle specified")
	}
	certbundle, err := ioutil.ReadFile(sl.defaults.Config.Syslog.CertificateBundlePath)
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(certbundle)
	if !ok {
		return nil, fmt.Errorf("failed to parse the given certificate bundle")
	}

	return roots, nil
}

// RegisterConfig does nothing here.
func (sl *Syslog) RegisterConfig(config configfile.LoggingConfig, defaults configfile.DefaultLoggerDetails) error {
	sl.config = config
	sl.defaults = defaults

	if sl.defaults.Config.Syslog.ConnectionType == "" {
		sl.defaults.Config.Syslog.ConnectionType = defaultProtocol
	}
	if !isValidDialer(sl.defaults.Config.Syslog.ConnectionType) {
		return fmt.Errorf("%s is not a valid protocol to connect to syslog", sl.defaults.Config.Syslog.ConnectionType)
	}

	if sl.defaults.Config.Syslog.ConnectionType == tlsConnection {
		roots, err := sl.readCertificates()
		if err != nil {
			return err
		}
		sl.tlsconfig = &tls.Config{
			RootCAs: roots,
		}
	}

	var err error
	sl.basename, err = os.Hostname()
	if err != nil {
		sl.basename = "not_available"
	}
	if sl.defaults.Config.Syslog.OverrideHostname != "" {
		sl.hostname = sl.defaults.Config.Syslog.OverrideHostname
	}

	return nil
}

// Start connects the logging engine and links the writer pipe to be able to call StdOut and StdErr.
// Start is safe to be called multiple times.
func (sl *Syslog) Start() error {
	if sl.isStarted() {
		return nil
	}

	// Dial with TLS
	var writer *syslogger.Writer
	var err error
	switch sl.defaults.Config.Syslog.ConnectionType {
	case tlsConnection:
		writer, err = syslogger.DialWithTLSConfig(
			tlsConnection,
			sl.defaults.Config.Syslog.Address,
			syslogger.LOG_INFO|syslogger.LOG_KERN,
			sl.defaults.Config.Syslog.ProgramName,
			sl.tlsconfig,
		)
	case tcpConnection, udpConnection:
		writer, err = syslogger.Dial(
			sl.defaults.Config.Syslog.ConnectionType,
			sl.defaults.Config.Syslog.Address,
			syslogger.LOG_INFO|syslogger.LOG_KERN,
			sl.defaults.Config.Syslog.ProgramName,
		)
	default:
		err = fmt.Errorf("Invalid logger type detected")
	}

	if err != nil {
		return fmt.Errorf("failed to connect to Syslog server because: %s", err)
	}

	sl.logwriter = writer
	sl.running = true
	return nil
}

// send will send the supplied text to syslog server.
func (sl *Syslog) send(facility, priority syslogger.Priority, tag, hostname, text string) error {
	_, err := sl.logwriter.WriteWithOverrides(facility, priority, hostname, tag, text)
	if err != nil {
		return err
	}
	return nil
}

func (sl *Syslog) appendBaseName(in string) string {
	return in + sl.basename
}

// Submit will consume a processlogger.LogMessage and route it to the correct writer.
func (sl *Syslog) Submit(msg processlogger.LogMessage) {
	// We need to know what is sending the log
	var tag string
	if msg.Config.Syslog.ProgramName != "" {
		tag = msg.Config.Syslog.ProgramName
	} else if sl.defaults.Config.Syslog.ProgramName != "" {
		tag = sl.defaults.Config.Syslog.ProgramName
	} else {
		tag = msg.Config.ProcessName
	}

	if sl.defaults.Config.Syslog.AddContainerNameToTag || msg.Config.Syslog.AddContainerNameToTag {
		tag = sl.appendBaseName(tag)
	}

	// We need a hostname
	var hostname string
	if msg.Config.Syslog.OverrideHostname != "" {
		hostname = msg.Config.Syslog.OverrideHostname
	} else if sl.defaults.Config.Syslog.OverrideHostname != "" {
		hostname = sl.defaults.Config.Syslog.OverrideHostname
	} else {
		hostname = sl.basename
	}

	if sl.defaults.Config.Syslog.AddContainerNameToHostname || msg.Config.Syslog.AddContainerNameToHostname {
		hostname = sl.appendBaseName(hostname)
	}

	// How critical is the log
	var level syslogger.Priority
	switch msg.Pipe {
	case processlogger.STDOUT:
		level = syslogger.LOG_INFO
	case processlogger.STDERR:
		level = syslogger.LOG_CRIT
	}

	// Can we detect the criticality from the log?
	if msg.Config.Syslog.ExtractLogLevel {
		detectedlevel, err := extractJSONlevel([]byte(msg.Message))
		if err != nil {
			// got nothing to send this to currently.
			// The process manager should really consume this
			// You get the default anyway...
		} else {
			level = detectedlevel
		}
	}
	// Send the log
	sl.send(sl.loggingFacility, level, tag, hostname, msg.Message)
}

// Shutdown will try to close the connection to the syslog server. This is a best effort close.
// A chan bool is return that will get a true on it once the connection is closed.
func (sl *Syslog) Shutdown() chan error {
	c := make(chan error, 1)
	go func() {
		// If it fails so fast that the logger didn't start, then it will be nil.
		// Nothing started, nothing to close.
		if sl.logwriter == nil {
			c <- nil
			return
		}
		err := sl.logwriter.Close()
		if err != nil {
			c <- fmt.Errorf("failed to close syslog connection. Error: %s", err)
		}
		c <- nil
	}()
	return c
}
