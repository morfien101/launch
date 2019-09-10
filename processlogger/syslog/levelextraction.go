package syslog

import (
	"encoding/json"
	"fmt"
	"strings"

	syslogger "github.com/silverstagtech/srslog"
)

type level struct {
	LVL string `json:"level"`
}

func extractJSONlevel(jsonlog []byte) (syslogger.Priority, error) {
	lvl := &level{}
	err := json.Unmarshal(jsonlog, lvl)
	if err != nil {
		return syslogger.LOG_INFO, fmt.Errorf("Failed to read json log. Error: %s", err)
	}
	if lvl.LVL == "" {
		return syslogger.LOG_INFO, fmt.Errorf("Failed to detect level in json log")
	}

	return detectLevel(strings.ToLower(lvl.LVL)), nil
}

func detectLevel(level string) syslogger.Priority {
	switch level {
	case "emerg":
		return syslogger.LOG_EMERG
	case "alert":
		return syslogger.LOG_ALERT
	case "crit":
		return syslogger.LOG_CRIT
	case "err":
		return syslogger.LOG_ERR
	case "error":
		return syslogger.LOG_ERR
	case "warning":
		return syslogger.LOG_WARNING
	case "warn":
		return syslogger.LOG_WARNING
	case "notice":
		return syslogger.LOG_NOTICE
	case "info":
		return syslogger.LOG_INFO
	case "debug":
		return syslogger.LOG_DEBUG
	default:
		return syslogger.LOG_INFO
	}
}
