package syslog

import (
	"testing"

	syslogger "github.com/silverstagtech/srslog"
)

func TestLevelExtraction(t *testing.T) {
	tests := []struct {
		name          string
		JSONBlob      []byte
		expectedLevel syslogger.Priority
	}{
		{
			name:          "info",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"info"}`),
			expectedLevel: syslogger.LOG_INFO,
		},
		{
			name:          "INFO",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"INFO"}`),
			expectedLevel: syslogger.LOG_INFO,
		},
		{
			name:          "Info",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"Info"}`),
			expectedLevel: syslogger.LOG_INFO,
		},
		{
			name:          "notice",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"notice"}`),
			expectedLevel: syslogger.LOG_NOTICE,
		},
		{
			name:          "warning",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"warning"}`),
			expectedLevel: syslogger.LOG_WARNING,
		},
		{
			name:          "warn",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"warn"}`),
			expectedLevel: syslogger.LOG_WARNING,
		},
		{
			name:          "err",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"err"}`),
			expectedLevel: syslogger.LOG_ERR,
		},
		{
			name:          "error",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"error"}`),
			expectedLevel: syslogger.LOG_ERR,
		},
		{
			name:          "crit",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"crit"}`),
			expectedLevel: syslogger.LOG_CRIT,
		},
		{
			name:          "alert",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"alert"}`),
			expectedLevel: syslogger.LOG_ALERT,
		},
		{
			name:          "emerg",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"emerg"}`),
			expectedLevel: syslogger.LOG_EMERG,
		},
		{
			name:          "debug",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"debug"}`),
			expectedLevel: syslogger.LOG_DEBUG,
		},
		{
			name:          "unknown",
			JSONBlob:      []byte(`{"test_key":"test_value","level":"not_valid"}`),
			expectedLevel: syslogger.LOG_INFO,
		},
	}

	for _, test := range tests {
		out, err := extractJSONlevel(test.JSONBlob)
		if err != nil {
			t.Logf("Test failed to see level, error %s", err)
			t.Fail()
		}
		if test.expectedLevel != out {
			t.Logf("Log level for test %s returned is not expected. Want: %v, Got: %v", test.name, test.expectedLevel, out)
			t.Fail()
		}
		t.Logf("%s translates to log level: %v", test.name, out)
	}
}

func benchExtractJSON(b *testing.B, JSONBlob []byte) syslogger.Priority {
	var lvl syslogger.Priority
	for n := 0; n < b.N; n++ {
		lvl, _ = extractJSONlevel(JSONBlob)
	}
	return lvl
}

func BenchmarkExtractJSONLevelSmall(b *testing.B) {
	JSONBlob := []byte(`{"test_key_1":"test_value_1","test_key_2":"test_value_2","level":"crit"}`)
	result := benchExtractJSON(b, JSONBlob)
	b.Logf("Blob size: %d. Last result: %v", len(JSONBlob), result)
}

func BenchmarkExtractJSONLevelMed(b *testing.B) {
	JSONBlob := []byte(`{"test_key_1":"test_value_1","test_key_2":"test_value_2","test_key_text":"this is a log messsage. It needs to be a bit long to make sure that we don't slow down too much while extracting the level","level":"crit"}`)
	result := benchExtractJSON(b, JSONBlob)
	b.Logf("Blob size: %d. Last result: %v", len(JSONBlob), result)
}

func BenchmarkExtractJSONLevelLarge(b *testing.B) {
	JSONBlob := []byte(`{"test_key_1":"test_value_1","test_key_2":"test_value_2","test_key_text_1":"this is a log messsage. It needs to be a bit long to make sure that we don't slow down too much while extracting the level","test_key_text_2":"this is a log messsage. It needs to be a bit long to make sure that we don't slow down too much while extracting the level","test_key_text_3":"this is a log messsage. It needs to be a bit long to make sure that we don't slow down too much while extracting the level","level":"crit"}`)
	result := benchExtractJSON(b, JSONBlob)
	b.Logf("Blob size: %d. Last result: %v", len(JSONBlob), result)
}
