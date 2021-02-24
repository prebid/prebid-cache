package backends

import (
	"context"
	"fmt"
	"strings"
	"testing"

	as "github.com/aerospike/aerospike-client-go"
	as_types "github.com/aerospike/aerospike-client-go/types"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
)

// Mock Aerospike client that always throws an error
type errorProneAerospikeClient struct {
	errorThrowingFunction string
}

func NewErrorProneAerospikeClient(funcName string) *errorProneAerospikeClient {
	return &errorProneAerospikeClient{
		errorThrowingFunction: funcName,
	}
}

func (c *errorProneAerospikeClient) NewUuidKey(namespace string, key string) (*as.Key, error) {
	if c.errorThrowingFunction == "TEST_KEY_GEN_ERROR" {
		return nil, as_types.NewAerospikeError(as_types.NOT_AUTHENTICATED)
	}
	return nil, nil
}

func (c *errorProneAerospikeClient) Get(key *as.Key) (*as.Record, error) {
	if c.errorThrowingFunction == "TEST_GET_ERROR" {
		return nil, formatAerospikeError(as_types.NewAerospikeError(as_types.SERVER_NOT_AVAILABLE), "GET")
	} else if c.errorThrowingFunction == "TEST_NO_BUCKET_ERROR" {
		return &as.Record{Bins: as.BinMap{"AnyKey": "any_value"}}, nil
	} else if c.errorThrowingFunction == "TEST_NON_STRING_VALUE_ERROR" {
		return &as.Record{Bins: as.BinMap{binValue: 0.0}}, nil
	}
	return nil, nil
}

func (c *errorProneAerospikeClient) Put(key *as.Key, value string, ttlSeconds int) error {
	if c.errorThrowingFunction == "TEST_PUT_ERROR" {
		return formatAerospikeError(as_types.NewAerospikeError(as_types.KEY_EXISTS_ERROR), "PUT")
	}
	return nil
}

// Mock Aerospike client that does not throw errors
type goodAerospikeClient struct {
	record *as.Record
}

func NewGoodAerospikeClient() *goodAerospikeClient {
	return &goodAerospikeClient{
		record: &as.Record{
			Bins: make(as.BinMap, 1),
		},
	}
}

func (c *goodAerospikeClient) Get(key *as.Key) (*as.Record, error) {
	if _, found := c.record.Bins[binValue]; !found {
		c.record.Bins[binValue] = "Default value"
	}
	return c.record, nil
}

func (c *goodAerospikeClient) Put(key *as.Key, value string, ttlSeconds int) error {
	c.record.Bins[binValue] = value
	return nil
}

func (c *goodAerospikeClient) NewUuidKey(namespace string, key string) (*as.Key, error) {
	return nil, nil
}

func TestNewAerospikeBackend(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		desc            string
		inCfg           config.Aerospike
		expectPanic     bool
		expectedLogInfo logEntry
	}{
		{
			desc: "Empty host logs fatal error",
			inCfg: config.Aerospike{
				Host: "",
				Port: 8080,
			},
			expectPanic: false,
			expectedLogInfo: logEntry{
				msg: "Cannot connect to empty Aerospike host",
				lvl: logrus.FatalLevel,
			},
		},
		{
			desc: "Invalid port logs fatal error",
			inCfg: config.Aerospike{
				Host: "http://fakeTestUrl.org",
				Port: -1,
			},
			expectPanic: false,
			expectedLogInfo: logEntry{
				msg: "Cannot connect to Aerospike host at port -1",
				lvl: logrus.FatalLevel,
			},
		},
		{
			desc: "Unable to connect fakeTestUrlto panic and log fatal error",
			inCfg: config.Aerospike{
				Host: "fakeTestUrl.foo",
				Port: 8888,
			},
			expectPanic: true,
			expectedLogInfo: logEntry{
				msg: "Connected to Aerospike at fakeTestUrl.foo:8888",
				lvl: logrus.FatalLevel,
			},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for _, test := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		// Run test
		if test.expectPanic {
			assert.Panics(t, func() { NewAerospikeBackend(test.inCfg, nil) }, "Aerospike library's NewClient() fhould have thrown an error and didn't, hence the panic didn't happen")
			assert.Equal(t, test.expectedLogInfo.lvl == logrus.FatalLevel, fatal, "Test case log level should be 'Fatal' and it wasn't")
		}

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestFormatAerospikeError(t *testing.T) {
	testCases := []struct {
		desc        string
		inErr       error
		expectedErr error
	}{
		{
			desc:        "Not even an error",
			inErr:       nil,
			expectedErr: nil,
		},
		{
			desc:        "Not an Aerospike error",
			inErr:       fmt.Errorf("Aerospike client.Get returned nil record"),
			expectedErr: fmt.Errorf("Aerospike client.Get returned nil record"),
		},
		{
			desc:        "Aerospike error",
			inErr:       as_types.NewAerospikeError(as_types.SERVER_NOT_AVAILABLE),
			expectedErr: fmt.Errorf("Aerospike TEST_CASE: Server is not accepting requests."),
		},
	}
	for _, test := range testCases {
		actualErr := formatAerospikeError(test.inErr, "TEST_CASE")
		if test.expectedErr == nil {
			assert.Nil(t, actualErr, "Nil error was expected")
		} else {
			assert.Equal(t, strings.Compare(test.expectedErr.Error(), actualErr.Error()), 0, test.desc)
		}
	}
}

func TestClientGet(t *testing.T) {
	aerospikeBackend := &AerospikeBackend{
		cfg:     config.Aerospike{Host: "www.anyHost.com", Port: 0},
		metrics: metricstest.CreateMockMetrics(),
	}

	testCases := []struct {
		desc              string
		inAerospikeClient AerospikeDB
		expectedValue     string
		expectedErrorMsg  string
	}{
		{
			desc:              "AerospikeBackend.Get() throws error when trying to generate new key",
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_KEY_GEN_ERROR"),
			expectedValue:     "",
			expectedErrorMsg:  "Aerospike GET: Not authenticated",
		},
		{
			desc:              "AerospikeBackend.Get() throws error when 'client.Get(..)' gets called",
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_GET_ERROR"),
			expectedValue:     "",
			expectedErrorMsg:  "Aerospike GET: Server is not accepting requests.",
		},
		{
			desc:              "AerospikeBackend.Get() throws error when 'client.Get(..)' returns a nil record",
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_NIL_RECORD_ERROR"),
			expectedValue:     "",
			expectedErrorMsg:  "Aerospike GET: Nil record",
		},
		{
			desc:              "AerospikeBackend.Get() throws error no BIN_VALUE bucket is found",
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_NO_BUCKET_ERROR"),
			expectedValue:     "",
			expectedErrorMsg:  "Aerospike GET: No 'value' bucket found",
		},
		{
			desc:              "AerospikeBackend.Get() returns a record that does not store a string",
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_NON_STRING_VALUE_ERROR"),
			expectedValue:     "",
			expectedErrorMsg:  "Aerospike GET: Unexpected non-string value found",
		},
		{
			desc:              "AerospikeBackend.Get() does not throw error",
			inAerospikeClient: NewGoodAerospikeClient(),
			expectedValue:     "Default value",
			expectedErrorMsg:  "",
		},
	}

	//Run tests
	for i, tt := range testCases {
		// Assign aerospike backend cient
		aerospikeBackend.client = tt.inAerospikeClient

		// Run test
		actualValue, actualErr := aerospikeBackend.Get(context.TODO(), "testKey")

		// Assert value
		assert.Equal(t, tt.expectedValue, actualValue, "Test case %d. Wrong value fetched", i)

		// Assert error
		if tt.expectedErrorMsg == "" {
			assert.Nil(t, actualErr, "Test case %d Nil error was expected", i)
		} else {
			assert.Equal(t, tt.expectedErrorMsg, actualErr.Error(), "Test case %d. Wrong error message", i)
		}
	}
}

func TestClientPut(t *testing.T) {
	aerospikeBackend := &AerospikeBackend{
		cfg:     config.Aerospike{Host: "www.anyHost.com", Port: 0},
		metrics: metricstest.CreateMockMetrics(),
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
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_KEY_GEN_ERROR"),
			inKey:             "testKey",
			inValueToStore:    "not default value",
			expectedStoredVal: "",
			expectedErrorMsg:  "Aerospike PUT: Not authenticated",
		},
		{
			desc:              "AerospikeBackend.Put() throws error when 'client.Put(..)' gets called",
			inAerospikeClient: NewErrorProneAerospikeClient("TEST_PUT_ERROR"),
			inKey:             "testKey",
			inValueToStore:    "not default value",
			expectedStoredVal: "",
			expectedErrorMsg:  "Aerospike PUT: Key already exists",
		},
		{
			desc:              "AerospikeBackend.Put() does not throw error",
			inAerospikeClient: NewGoodAerospikeClient(),
			inKey:             "testKey",
			inValueToStore:    "any value",
			expectedStoredVal: "any value",
			expectedErrorMsg:  "",
		},
	}

	for i, tt := range testCases {
		// Assign aerospike backend cient
		aerospikeBackend.client = tt.inAerospikeClient

		// Run test
		actualErr := aerospikeBackend.Put(context.TODO(), tt.inKey, tt.inValueToStore, 0)

		// Assert Put error
		if tt.expectedErrorMsg != "" {
			assert.Equal(t, tt.expectedErrorMsg, actualErr.Error(), "Test case %d. Wrong error message", i)
		} else {
			assert.Nil(t, actualErr, "Test case %d Nil error was expected", i)

			// Assert Put() sucessfully logged "not default value" under "testKey":
			storedValue, getErr := aerospikeBackend.Get(context.TODO(), "testKey")

			assert.Nil(t, getErr, "Get() was not expected to throw an error")
			assert.Equal(t, tt.inValueToStore, storedValue, "Put() stored wrong value")
		}
	}
}
