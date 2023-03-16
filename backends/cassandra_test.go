package backends

import (
	"context"
	"errors"
	"testing"

	"github.com/gocql/gocql"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestCassandraClientGet(t *testing.T) {
	cassandraBackend := &CassandraBackend{}

	type testInput struct {
		cassandraClient CassandraDB
		key             string
	}

	type testExpectedValues struct {
		value string
		err   error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testExpectedValues
	}{
		{
			"CassandraBackend.Get() throws a Cassandra ErrNotFound error",
			testInput{
				&ErrorProneCassandraClient{ServerError: gocql.ErrNotFound},
				"someKeyThatWontBeFound",
			},
			testExpectedValues{
				value: "",
				err:   utils.NewPBCError(utils.KEY_NOT_FOUND),
			},
		},
		{
			"CassandraBackend.Get() throws an error different from Cassandra ErrNotFound error",
			testInput{
				&ErrorProneCassandraClient{ServerError: errors.New("some other get error")},
				"someKey",
			},
			testExpectedValues{
				value: "",
				err:   errors.New("some other get error"),
			},
		},
		{
			"CassandraBackend.Get() doesn't throw an error",
			testInput{
				&GoodCassandraClient{
					StoredData: map[string]string{"defaultKey": "aValue"},
				},
				"defaultKey",
			},
			testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
	}

	for _, tt := range testCases {
		cassandraBackend.client = tt.in.cassandraClient

		// Run test
		actualValue, actualErr := cassandraBackend.Get(context.Background(), tt.in.key)

		// Assertions
		assert.Equal(t, tt.expected.value, actualValue, tt.desc)
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)
	}
}

func TestCassandraClientPut(t *testing.T) {
	cassandraBackend := &CassandraBackend{
		defaultTTL: 50,
	}

	type testInput struct {
		cassandraClient CassandraDB
		key             string
		valueToStore    string
		ttl             int
	}

	type testExpectedValues struct {
		value string
		err   error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testExpectedValues
	}{
		{
			"CassandraBackend.Put() didn't store the value under the corresponding key. Because the 'applied' return value was false, expect a RECORD_EXISTS error",
			testInput{
				cassandraClient: &ErrorProneCassandraClient{Applied: false},
				key:             "someKey",
				valueToStore:    "someValue",
				ttl:             10,
			},
			testExpectedValues{
				value: "",
				err:   utils.NewPBCError(utils.RECORD_EXISTS),
			},
		},
		{
			"CassandraBackend.Put() returns the 'applied' boolean value as 'true' in addition to a Cassandra server error. Not even sure if this scenario is feasible in practice",
			testInput{
				cassandraClient: &ErrorProneCassandraClient{Applied: true, ServerError: gocql.ErrNoConnections},
				key:             "someKey",
				valueToStore:    "someValue",
				ttl:             10,
			},
			testExpectedValues{
				value: "",
				err:   errors.New("gocql: no hosts available in the pool"),
			},
		},
		{
			"CassandraBackend.Put() gets called with zero ttlSeconds, value gets successfully set anyways",
			testInput{
				cassandraClient: &GoodCassandraClient{StoredData: map[string]string{"defaultKey": "aValue"}},
				key:             "defaultKey",
				valueToStore:    "aValue",
				ttl:             0,
			},
			testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
		{
			"CassandraBackend.Put() successful, no need to set defaultTTL because ttl is greater than zero",
			testInput{
				cassandraClient: &GoodCassandraClient{StoredData: map[string]string{"defaultKey": "aValue"}},
				key:             "defaultKey",
				valueToStore:    "aValue",
				ttl:             1,
			},
			testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
	}

	for _, tt := range testCases {
		cassandraBackend.client = tt.in.cassandraClient

		// Run test
		actualErr := cassandraBackend.Put(context.Background(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assert Put error
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)

		// Assert value
		if tt.expected.err == nil {
			storedValue, getErr := cassandraBackend.Get(context.Background(), tt.in.key)

			assert.NoError(t, getErr, tt.desc)
			assert.Equal(t, tt.expected.value, storedValue, tt.desc)
		}
	}
}
