package backends

import (
	"context"
	"fmt"
	"testing"

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

func TestNewAerospikeBackend(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		desc               string
		inCfg              config.Aerospike
		expectPanic        bool
		expectedLogEntries []logEntry
	}{
		{
			desc: "Unable to connect hosts fakeTestUrl panic and log fatal error when passed additional hosts",
			inCfg: config.Aerospike{
				Hosts: []string{"foo.com", "bat.com"},
				Port:  8888,
			},
			expectPanic: true,
			expectedLogEntries: []logEntry{
				{
					msg: "Error creating Aerospike backend: ResultCode: TIMEOUT, Iteration: 0, InDoubt: false, Node: <nil>: command execution timed out on client: See `Policy.Timeout`",
					lvl: logrus.FatalLevel,
				},
			},
		},
		{
			desc: "Unable to connect host and hosts panic and log fatal error when passed additional hosts",
			inCfg: config.Aerospike{
				Host:  "fakeTestUrl.foo",
				Hosts: []string{"foo.com", "bat.com"},
				Port:  8888,
			},
			expectPanic: true,
			expectedLogEntries: []logEntry{
				{
					msg: "config.backend.aerospike.host is being deprecated in favor of config.backend.aerospike.hosts",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "Error creating Aerospike backend: ResultCode: TIMEOUT, Iteration: 0, InDoubt: false, Node: <nil>: command execution timed out on client: See `Policy.Timeout`",
					lvl: logrus.FatalLevel,
				},
			},
		},
		{
			desc: "Unable to connect host panic and log fatal error",
			inCfg: config.Aerospike{
				Host: "fakeTestUrl.foo",
				Port: 8888,
			},
			expectPanic: true,
			expectedLogEntries: []logEntry{
				{
					msg: "config.backend.aerospike.host is being deprecated in favor of config.backend.aerospike.hosts",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "Error creating Aerospike backend: ResultCode: TIMEOUT, Iteration: 0, InDoubt: false, Node: <nil>: command execution timed out on client: See `Policy.Timeout`",
					lvl: logrus.FatalLevel,
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
		// Run test
		assert.Panics(t, func() { NewAerospikeBackend(test.inCfg, nil) }, "Aerospike library's NewClientWithPolicyAndHost() should have thrown an error and didn't, hence the panic didn't happen")
		if assert.Len(t, hook.Entries, len(test.expectedLogEntries), test.desc) {
			for i := 0; i < len(test.expectedLogEntries); i++ {
				assert.Equal(t, test.expectedLogEntries[i].msg, hook.Entries[i].Message, test.desc)
				assert.Equal(t, test.expectedLogEntries[i].lvl, hook.Entries[i].Level, test.desc)
			}
		}

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
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
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_KEY_GEN_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "ResultCode: NOT_AUTHENTICATED, Iteration: 0, InDoubt: false, Node: <nil>: ",
		},
		{
			desc:              "AerospikeBackend.Get() throws error when 'client.Get(..)' gets called",
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_GET_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "Key not found",
		},
		{
			desc:              "AerospikeBackend.Get() throws error when 'client.Get(..)' returns a nil record",
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_NIL_RECORD_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "Nil record",
		},
		{
			desc:              "AerospikeBackend.Get() throws error no BIN_VALUE bucket is found",
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_NO_BUCKET_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "No 'value' bucket found",
		},
		{
			desc:              "AerospikeBackend.Get() returns a record that does not store a string",
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_NON_STRING_VALUE_ERROR"},
			expectedValue:     "",
			expectedErrorMsg:  "Unexpected non-string value found",
		},
		{
			desc: "AerospikeBackend.Get() does not throw error",
			inAerospikeClient: &goodAerospikeClient{
				records: map[string]*as.Record{
					"defaultKey": {
						Bins: as.BinMap{binValue: "Default value"},
					},
				},
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
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_KEY_GEN_ERROR"},
			inKey:             "testKey",
			inValueToStore:    "not default value",
			expectedStoredVal: "",
			expectedErrorMsg:  "ResultCode: NOT_AUTHENTICATED, Iteration: 0, InDoubt: false, Node: <nil>: ",
		},
		{
			desc:              "AerospikeBackend.Put() throws error when 'client.Put(..)' gets called",
			inAerospikeClient: &errorProneAerospikeClient{errorThrowingFunction: "TEST_PUT_ERROR"},
			inKey:             "testKey",
			inValueToStore:    "not default value",
			expectedStoredVal: "",
			expectedErrorMsg:  "Record exists with provided key.",
		},
		{
			desc: "AerospikeBackend.Put() does not throw error",
			inAerospikeClient: &goodAerospikeClient{
				records: map[string]*as.Record{
					"defaultKey": {
						Bins: as.BinMap{binValue: "Default value"},
					},
				},
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

// Aerospike client that always throws an error
type errorProneAerospikeClient struct {
	errorThrowingFunction string
}

func (c *errorProneAerospikeClient) NewUUIDKey(namespace string, key string) (*as.Key, error) {
	if c.errorThrowingFunction == "TEST_KEY_GEN_ERROR" {
		return nil, &as.AerospikeError{ResultCode: as_types.NOT_AUTHENTICATED}
	}
	return nil, nil
}

func (c *errorProneAerospikeClient) Get(key *as.Key) (*as.Record, error) {
	if c.errorThrowingFunction == "TEST_GET_ERROR" {
		return nil, &as.AerospikeError{ResultCode: as_types.KEY_NOT_FOUND_ERROR}
	} else if c.errorThrowingFunction == "TEST_NO_BUCKET_ERROR" {
		return &as.Record{Bins: as.BinMap{"AnyKey": "any_value"}}, nil
	} else if c.errorThrowingFunction == "TEST_NON_STRING_VALUE_ERROR" {
		return &as.Record{Bins: as.BinMap{binValue: 0.0}}, nil
	}
	return nil, nil
}

func (c *errorProneAerospikeClient) Put(policy *as.WritePolicy, key *as.Key, binMap as.BinMap) error {
	if c.errorThrowingFunction == "TEST_PUT_ERROR" {
		return &as.AerospikeError{ResultCode: as_types.KEY_EXISTS_ERROR}
	}
	return nil
}

// Aerospike client that does not throw errors
type goodAerospikeClient struct {
	records map[string]*as.Record
}

func (c *goodAerospikeClient) Get(aeKey *as.Key) (*as.Record, error) {
	if aeKey != nil && aeKey.Value() != nil {
		key := aeKey.Value().String()

		if rec, found := c.records[key]; found {
			return rec, nil
		}
	}
	return nil, &as.AerospikeError{ResultCode: as_types.KEY_NOT_FOUND_ERROR}
}

func (c *goodAerospikeClient) Put(policy *as.WritePolicy, aeKey *as.Key, binMap as.BinMap) error {
	if aeKey != nil && aeKey.Value() != nil {
		key := aeKey.Value().String()
		c.records[key] = &as.Record{
			Bins: binMap,
		}
		return nil
	}
	return &as.AerospikeError{ResultCode: as_types.KEY_MISMATCH}
}

func (c *goodAerospikeClient) NewUUIDKey(namespace string, key string) (*as.Key, error) {
	return as.NewKey(namespace, setName, key)
}
