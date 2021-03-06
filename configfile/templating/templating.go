// Package templating is used to allow the configuration files to have
// some dynamic configuration to them.
// It was shamelessly taken from
// https://github.com/tnozicka/goenvtemplator/blob/master/template_test.go
// However I didn't need most of the other stuff in the package
package templating

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"text/template"
)

type OptionalString struct {
	ptr *string
}

var funcMap = template.FuncMap{
	"env":      Env,
	"default":  Default,
	"required": Required,
	"zerolen":  ZeroLen,
}

func (s OptionalString) String() string {
	if s.ptr == nil {
		return ""
	}
	return *s.ptr
}

func Env(key string) OptionalString {
	value, ok := os.LookupEnv(key)
	if !ok {
		return OptionalString{nil}
	}
	return OptionalString{&value}
}

func Default(args ...interface{}) (string, error) {
	for _, arg := range args {
		if arg == nil {
			continue
		}
		switch v := arg.(type) {
		case string:
			return v, nil
		case *string:
			if v != nil {
				return *v, nil
			}
		case OptionalString:
			if v.ptr != nil {
				return *v.ptr, nil
			}
		default:
			return "", fmt.Errorf("Default: unsupported type '%T'", v)
		}
	}

	return "", errors.New("Default: all arguments are nil")
}

func Required(arg interface{}) (string, error) {
	if arg == nil {
		return "", errors.New("Required argument is missing")
	}

	switch value := arg.(type) {
	case string:
		return value, nil
	case *string:
		if value != nil {
			return *value, nil
		}
	case OptionalString:
		if value.ptr != nil {
			return *value.ptr, nil
		}
	default:
		return "", fmt.Errorf("Requires: unsupported type '%T'", value)
	}
	return "", nil
}

// GenerateTemplate will action all the functions on the configuration file
func GenerateTemplate(source []byte) ([]byte, error) {
	tplt, err := template.New("configfile").Funcs(funcMap).Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("failed to create template. Error: %s", err)
	}

	var buffer bytes.Buffer
	if err = tplt.Execute(&buffer, nil); err != nil {
		return nil, fmt.Errorf("failed to transform template. Error: %s", err)
	}
	return buffer.Bytes(), nil
}

// ZeroLen is used to test id a value passed in is of zero length. This is expected
// to be a string or OptionalString. Used primarily to determine if a value exists.
// Think of -z in Bash. The true or false value is returned depending on the outcome.
func ZeroLen(arg, trueValue, falseValue interface{}) (interface{}, error) {
	if arg == nil {
		return trueValue, nil
	}

	var testString string

	switch v := arg.(type) {
	case string:
		testString = v
	case *string:
		if v != nil {
			testString = *v
		} else {
			return trueValue, nil
		}
	case OptionalString:
		if v.ptr != nil {
			testString = *v.ptr
		}
	default:
		return falseValue, fmt.Errorf("Default: unsupported type '%T'", v)
	}

	if len(testString) == 0 {
		return trueValue, nil
	}
	return falseValue, nil
}
