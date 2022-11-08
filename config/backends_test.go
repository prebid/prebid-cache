package config

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	testLogrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestAerospikeValidateAndLog(t *testing.T) {

	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	type testCase struct {
		desc          string
		inCfg         Aerospike
		hasError      bool
		expectedError error
		logEntries    []logComponents
	}
	testGroups := []struct {
		desc      string
		testCases []testCase
	}{
		{
			desc: "No errors expected",
			testCases: []testCase{
				{
					desc: "aerospike.host passed in",
					inCfg: Aerospike{
						Host: "foo.com",
						Port: 8888,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "aerospike.host passed in",
					inCfg: Aerospike{
						Host:      "foo.com",
						Port:      8888,
						Namespace: "prebid",
						User:      "prebid-user",
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: prebid", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: prebid-user", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "aerospike.hosts passed in",
					inCfg: Aerospike{
						Hosts: []string{"foo.com", "bat.com"},
						Port:  8888,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: ", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{"foo.com", "bat.com"}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "both aerospike.host aerospike.hosts passed in",
					inCfg: Aerospike{
						Host:  "foo.com",
						Hosts: []string{"foo.com", "bat.com"},
						Port:  8888,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{"foo.com", "bat.com"}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "both aerospike.host, aerospike.hosts and aerospike.default_ttl_seconds set",
					inCfg: Aerospike{
						Host:       "foo.com",
						Hosts:      []string{"foo.com", "bat.com"},
						Port:       8888,
						DefaultTTL: 3600,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{"foo.com", "bat.com"}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.default_ttl_seconds: 3600. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "both aerospike.host, aerospike.port and an aerospike.max_read_retries invalid value. Default to 2 retries",
					inCfg: Aerospike{
						Host:           "foo.com",
						Port:           8888,
						MaxReadRetries: 1,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "aerospike.max_read_retries valid value.",
					inCfg: Aerospike{
						Host:           "foo.com",
						Port:           8888,
						MaxReadRetries: 3,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries: 3.", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "aerospike.max_write_retries invalid value. Default to 2 retries",
					inCfg: Aerospike{
						Host:            "foo.com",
						Port:            8888,
						MaxReadRetries:  2,
						MaxWriteRetries: -1,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_write_retries value cannot be negative and will default to 0", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "aerospike.max_read_retries valid value.",
					inCfg: Aerospike{
						Host:            "foo.com",
						Port:            8888,
						MaxReadRetries:  2,
						MaxWriteRetries: 1,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_write_retries: 1.", lvl: logrus.InfoLevel},
					},
				},
				{
					desc: "aerospike.connection_idle_timeout_seconds value found in config",
					inCfg: Aerospike{
						Host:                  "foo.com",
						Port:                  8888,
						ConnectionIdleTimeout: 1,
					},
					hasError: false,
					logEntries: []logComponents{
						{msg: "config.backend.aerospike.host: foo.com", lvl: logrus.InfoLevel},
						{msg: fmt.Sprintf("config.backend.aerospike.hosts: %v", []string{}), lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.port: 8888", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.namespace: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.user: ", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.connection_idle_timeout_seconds: 1.", lvl: logrus.InfoLevel},
						{msg: "config.backend.aerospike.max_read_retries value will default to 2", lvl: logrus.InfoLevel},
					},
				},
			},
		},
		{
			desc: "Expect error",
			testCases: []testCase{
				{
					desc: "aerospike.host and aerospike.hosts missing",
					inCfg: Aerospike{
						Port: 8888,
					},
					hasError:      true,
					expectedError: fmt.Errorf("Cannot connect to empty Aerospike host(s)"),
				},
				{
					desc: "aerospike.port config missing",
					inCfg: Aerospike{
						Host: "foo.com",
					},
					hasError:      true,
					expectedError: fmt.Errorf("Cannot connect to Aerospike host at port 0"),
				},
				{
					desc: "aerospike.port config missing",
					inCfg: Aerospike{
						Host:  "foo.com",
						Hosts: []string{"bar.com"},
					},
					hasError:      true,
					expectedError: fmt.Errorf("Cannot connect to Aerospike host at port 0"),
				},
			},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for _, group := range testGroups {
		for _, test := range group.testCases {
			fatal = false

			//run test
			if test.hasError {
				assert.Equal(t, test.inCfg.validateAndLog(), test.expectedError, group.desc+" : "+test.desc)
			} else {
				assert.Nil(t, test.inCfg.validateAndLog(), group.desc+" : "+test.desc)
			}

			assert.False(t, fatal, group.desc+" : "+test.desc)

			if assert.Len(t, hook.Entries, len(test.logEntries), "Incorrect number of entries were logged to logrus in test %s", group.desc+" : "+test.desc) {
				for i := 0; i < len(test.logEntries); i++ {
					assert.Equal(t, test.logEntries[i].msg, hook.Entries[i].Message, group.desc+" : "+test.desc)
					assert.Equal(t, test.logEntries[i].lvl, hook.Entries[i].Level, group.desc+" : "+test.desc)
				}
			}

			//Reset log after every test and assert successful reset
			hook.Reset()
			assert.Nil(t, hook.LastEntry(), group.desc+" : "+test.desc)
		}
	}
}
