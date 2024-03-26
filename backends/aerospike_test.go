package backends

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"
	as_types "github.com/aerospike/aerospike-client-go/v6/types"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAerospikeClientPolicy(t *testing.T) {
	testCases := []struct {
		desc     string
		inCfg    config.Aerospike
		expected *as.ClientPolicy
	}{
		{
			desc:     "Blank configuration",
			inCfg:    config.Aerospike{},
			expected: as.NewClientPolicy(),
		},
		{
			desc: "Config with credentials",
			inCfg: config.Aerospike{
				User:     "foobar",
				Password: "password",
			},
			expected: &as.ClientPolicy{
				User:                        "foobar",
				Password:                    "password",
				AuthMode:                    as.AuthModeInternal,
				Timeout:                     30 * time.Second,
				IdleTimeout:                 0 * time.Second,
				LoginTimeout:                10 * time.Second,
				ConnectionQueueSize:         100,
				OpeningConnectionThreshold:  0,
				FailIfNotConnected:          true,
				TendInterval:                time.Second,
				LimitConnectionsToQueueSize: true,
				IgnoreOtherSubnetAliases:    false,
				MaxErrorRate:                100,
				ErrorRateWindow:             1,
			},
		},
		{
			desc: "Config with ConnIdleTimeoutSecs",
			inCfg: config.Aerospike{
				ConnIdleTimeoutSecs: 3600,
			},
			expected: &as.ClientPolicy{
				AuthMode:                    as.AuthModeInternal,
				Timeout:                     30 * time.Second,
				IdleTimeout:                 3600 * time.Second,
				LoginTimeout:                10 * time.Second,
				ConnectionQueueSize:         100,
				OpeningConnectionThreshold:  0,
				FailIfNotConnected:          true,
				TendInterval:                time.Second,
				LimitConnectionsToQueueSize: true,
				IgnoreOtherSubnetAliases:    false,
				MaxErrorRate:                100,
				ErrorRateWindow:             1,
			},
		},
		{
			desc: "Config with ConnIdleTimeoutSecs",
			inCfg: config.Aerospike{
				ConnQueueSize: 31416,
			},
			expected: &as.ClientPolicy{
				AuthMode:                    as.AuthModeInternal,
				Timeout:                     30 * time.Second,
				IdleTimeout:                 0 * time.Second,
				LoginTimeout:                10 * time.Second,
				ConnectionQueueSize:         31416,
				OpeningConnectionThreshold:  0,
				FailIfNotConnected:          true,
				TendInterval:                time.Second,
				LimitConnectionsToQueueSize: true,
				IgnoreOtherSubnetAliases:    false,
				MaxErrorRate:                100,
				ErrorRateWindow:             1,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			asPolicy := generateAerospikeClientPolicy(tc.inCfg)
			assert.Equal(t, tc.expected, asPolicy)
		})
	}
}

func TestGenerateHostsList(t *testing.T) {
	type testOutput struct {
		hosts []*as.Host
		err   error
	}
	type logEntry struct {
		msg string
		lvl logrus.Level
	}
	testCases := []struct {
		desc               string
		inCfg              config.Aerospike
		expectedOut        testOutput
		expectedLogEntries []logEntry
	}{
		{
			desc:  "no_port",
			inCfg: config.Aerospike{},
			expectedOut: testOutput{
				err: errors.New("Cannot connect to Aerospike host at port 0"),
			},
		},
		{
			desc:  "port_no_host_nor_hosts",
			inCfg: config.Aerospike{Port: 8888},
			expectedOut: testOutput{
				err: errors.New("Cannot connect to empty Aerospike host(s)"),
			},
		},
		{
			desc: "port_and_hosts_no_host",
			inCfg: config.Aerospike{
				Port:  8888,
				Hosts: []string{"foo.com", "bar.com"},
			},
			expectedOut: testOutput{
				hosts: []*as.Host{
					as.NewHost("foo.com", 8888),
					as.NewHost("bar.com", 8888),
				},
			},
		},
		{
			desc: "port_and_host",
			inCfg: config.Aerospike{
				Host: "foo.com",
				Port: 8888,
			},
			expectedOut: testOutput{
				hosts: []*as.Host{as.NewHost("foo.com", 8888)},
			},
			expectedLogEntries: []logEntry{
				{
					msg: "config.backend.aerospike.host is being deprecated in favor of config.backend.aerospike.hosts",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			desc: "Port_host_and_hosts",
			inCfg: config.Aerospike{
				Port:  8888,
				Host:  "foo.com",
				Hosts: []string{"foo.com", "bar.com"},
			},
			expectedOut: testOutput{
				hosts: []*as.Host{
					as.NewHost("foo.com", 8888),
					as.NewHost("foo.com", 8888),
					as.NewHost("bar.com", 8888),
				},
			},
			expectedLogEntries: []logEntry{
				{
					msg: "config.backend.aerospike.host is being deprecated in favor of config.backend.aerospike.hosts",
					lvl: logrus.InfoLevel,
				},
			},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			asHosts, err := generateHostsList(tc.inCfg)
			assert.Equal(t, tc.expectedOut.err, err)
			if assert.Len(t, asHosts, len(tc.expectedOut.hosts)) {
				assert.ElementsMatch(t, tc.expectedOut.hosts, asHosts)
			}
			if assert.Len(t, hook.Entries, len(tc.expectedLogEntries)) {
				for i := 0; i < len(tc.expectedLogEntries); i++ {
					assert.Equal(t, tc.expectedLogEntries[i].lvl, hook.Entries[i].Level)
					assert.Equal(t, tc.expectedLogEntries[i].msg, hook.Entries[i].Message)
				}
			}
			//Reset log after every test and assert successful reset
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
		})
	}
}

func TestNewAerospikeBackend(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}

	errorProneNewClientFunc := func(*as.ClientPolicy, ...*as.Host) (*as.Client, as.Error) {
		return nil, &as.AerospikeError{}
	}
	successfulNewClientFunc := func(*as.ClientPolicy, ...*as.Host) (*as.Client, as.Error) {
		return nil, nil
	}

	testCases := []struct {
		desc               string
		inCfg              config.Aerospike
		newClientFunc      NewAerospikeClientFunc
		expectedLogEntries []logEntry
		expectedPanic      bool
	}{
		{
			desc:  "no_port_error",
			inCfg: config.Aerospike{},
			expectedLogEntries: []logEntry{
				{
					msg: "Error creating Aerospike backend: Cannot connect to Aerospike host at port 0",
					lvl: logrus.FatalLevel,
				},
			},
		},
		{
			desc:  "no_host_nor_hosts_error",
			inCfg: config.Aerospike{Port: 8888},
			expectedLogEntries: []logEntry{
				{
					msg: "Error creating Aerospike backend: Cannot connect to empty Aerospike host(s)",
					lvl: logrus.FatalLevel,
				},
			},
		},
		{
			desc: "newAerospikeClient_error",
			inCfg: config.Aerospike{
				Hosts: []string{"fakeUrl"},
				Port:  8888,
			},
			newClientFunc: errorProneNewClientFunc,
			expectedLogEntries: []logEntry{
				{
					msg: "Error creating Aerospike backend: ResultCode: OK, Iteration: 0, InDoubt: false, Node: <nil>: ",
					lvl: logrus.FatalLevel,
				},
			},
			expectedPanic: true,
		},
		{
			desc: "success_with_deprecated_host",
			inCfg: config.Aerospike{
				Host: "fakeUrl",
				Port: 8888,
			},
			newClientFunc: successfulNewClientFunc,
			expectedLogEntries: []logEntry{
				{
					msg: "config.backend.aerospike.host is being deprecated in favor of config.backend.aerospike.hosts",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "Connected to Aerospike host(s) [fakeUrl] on port 8888",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			desc: "success_with_hosts_list",
			inCfg: config.Aerospike{
				Hosts: []string{"fakeUrl"},
				Port:  8888,
			},
			newClientFunc: successfulNewClientFunc,
			expectedLogEntries: []logEntry{
				{
					msg: "Connected to Aerospike host(s) [fakeUrl ] on port 8888",
					lvl: logrus.InfoLevel,
				},
			},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			if test.expectedPanic {
				if !assert.Panics(t, func() { newAerospikeBackend(test.newClientFunc, test.inCfg, nil) }, "Aerospike library's NewClientWithPolicyAndHost() should have thrown an error and didn't, hence the panic didn't happen") {
					return
				}
			} else {
				if !assert.NotPanics(t, func() { newAerospikeBackend(test.newClientFunc, test.inCfg, nil) }, "Aerospike library's NewClientWithPolicyAndHost() should have thrown an error and didn't, hence the panic didn't happen") {
					return
				}
			}

			if assert.Len(t, hook.Entries, len(test.expectedLogEntries), test.desc) {
				for i := 0; i < len(test.expectedLogEntries); i++ {
					assert.Equal(t, test.expectedLogEntries[i].lvl, hook.Entries[i].Level, test.desc)
					assert.Equal(t, test.expectedLogEntries[i].msg, hook.Entries[i].Message, test.desc)
				}
			}

			//Reset log after every test and assert successful reset
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
		})

	}
}

func TestClassifyAerospikeError(t *testing.T) {
	testCases := []struct {
		desc        string
		inErr       error
		expectedErr error
	}{
		{
			desc:        "Nil error",
			inErr:       nil,
			expectedErr: nil,
		},
		{
			desc:        "Generic non-nil error, expect same error in output",
			inErr:       fmt.Errorf("client.Get returned nil record"),
			expectedErr: fmt.Errorf("client.Get returned nil record"),
		},
		{
			desc:        "Aerospike error is neither KEY_NOT_FOUND_ERROR nor KEY_EXISTS_ERROR, expect same error as output",
			inErr:       &as.AerospikeError{ResultCode: as_types.SERVER_NOT_AVAILABLE},
			expectedErr: &as.AerospikeError{ResultCode: as_types.SERVER_NOT_AVAILABLE},
		},
		{
			desc:        "Aerospike KEY_NOT_FOUND_ERROR error, expect Prebid Cache's KEY_NOT_FOUND error",
			inErr:       &as.AerospikeError{ResultCode: as_types.KEY_NOT_FOUND_ERROR},
			expectedErr: utils.NewPBCError(utils.KEY_NOT_FOUND),
		},
		{
			desc:        "Aerospike KEY_EXISTS_ERROR error, expect Prebid Cache's RECORD_EXISTS error",
			inErr:       &as.AerospikeError{ResultCode: as_types.KEY_EXISTS_ERROR},
			expectedErr: utils.NewPBCError(utils.RECORD_EXISTS),
		},
	}
	for _, test := range testCases {
		actualErr := classifyAerospikeError(test.inErr)
		if test.expectedErr == nil {
			assert.Nil(t, actualErr, test.desc)
		} else {
			assert.Equal(t, test.expectedErr.Error(), actualErr.Error(), test.desc)
		}
	}
}

func TestAerospikeClientGet(t *testing.T) {
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	aerospikeBackend := &AerospikeBackend{
		metrics: m,
	}

	testCases := []struct {
		desc              string
		inAerospikeClient AerospikeDB
		expectedValue     string
		expectedErrorMsg  string
	}{
		{
			desc:              "AerospikeBackend.Get() throws error when trying to generate new key",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_KEY_GEN_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "ResultCode: NOT_AUTHENTICATED, Iteration: 0, InDoubt: false, Node: <nil>: ",
		},
		{
			desc:              "AerospikeBackend.Get() throws error when 'client.Get(..)' gets called",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_GET_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "Key not found",
		},
		{
			desc:              "AerospikeBackend.Get() throws error when 'client.Get(..)' returns a nil record",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_NIL_RECORD_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "Nil record",
		},
		{
			desc:              "AerospikeBackend.Get() throws error no BIN_VALUE bucket is found",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_NO_BUCKET_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "No 'value' bucket found",
		},
		{
			desc:              "AerospikeBackend.Get() returns a record that does not store a string",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_NON_STRING_VALUE_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "Unexpected non-string value found",
		},
		{
			desc: "AerospikeBackend.Get() does not throw error",
			inAerospikeClient: &GoodAerospikeClient{
				StoredData: map[string]string{"defaultKey": "Default value"},
			},
			expectedValue:    "Default value",
			expectedErrorMsg: "",
		},
	}

	for _, tt := range testCases {
		// Assign aerospike backend cient
		aerospikeBackend.client = tt.inAerospikeClient

		// Run test
		actualValue, actualErr := aerospikeBackend.Get(context.Background(), "defaultKey")

		// Assertions
		assert.Equal(t, tt.expectedValue, actualValue, tt.desc)

		if tt.expectedErrorMsg == "" {
			assert.Nil(t, actualErr, tt.desc)
		} else {
			assert.Equal(t, tt.expectedErrorMsg, actualErr.Error(), tt.desc)
		}
	}
}

func TestClientPut(t *testing.T) {
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	aerospikeBackend := &AerospikeBackend{
		metrics: m,
	}

	testCases := []struct {
		desc              string
		inAerospikeClient AerospikeDB
		inKey             string
		inValueToStore    string
		expectedStoredVal string
		expectedErrorMsg  string
	}{
		{
			desc:              "AerospikeBackend.Put() throws error when trying to generate new key",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_KEY_GEN_ERROR"},
			inKey:             "testKey",
			inValueToStore:    "not default value",
			expectedStoredVal: "",
			expectedErrorMsg:  "ResultCode: NOT_AUTHENTICATED, Iteration: 0, InDoubt: false, Node: <nil>: ",
		},
		{
			desc:              "AerospikeBackend.Put() throws error when 'client.Put(..)' gets called",
			inAerospikeClient: &ErrorProneAerospikeClient{ServerError: "TEST_PUT_ERROR"},
			inKey:             "testKey",
			inValueToStore:    "not default value",
			expectedStoredVal: "",
			expectedErrorMsg:  "Record exists with provided key.",
		},
		{
			desc: "AerospikeBackend.Put() does not throw error",
			inAerospikeClient: &GoodAerospikeClient{
				StoredData: map[string]string{"defaultKey": "Default value"},
			},
			inKey:             "testKey",
			inValueToStore:    "any value",
			expectedStoredVal: "any value",
			expectedErrorMsg:  "",
		},
	}

	for _, tt := range testCases {
		// Assign aerospike backend cient
		aerospikeBackend.client = tt.inAerospikeClient

		// Run test
		actualErr := aerospikeBackend.Put(context.Background(), tt.inKey, tt.inValueToStore, 0)

		// Assert Put error
		if tt.expectedErrorMsg != "" {
			assert.Equal(t, tt.expectedErrorMsg, actualErr.Error(), tt.desc)
		} else {
			assert.Nil(t, actualErr, tt.desc)

			// Assert Put() sucessfully logged "not default value" under "testKey":
			storedValue, getErr := aerospikeBackend.Get(context.Background(), tt.inKey)

			assert.Nil(t, getErr, tt.desc)
			assert.Equal(t, tt.inValueToStore, storedValue, tt.desc)
		}
	}
}
