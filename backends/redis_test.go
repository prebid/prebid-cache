package backends

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestRedisClientGet(t *testing.T) {
	redisBackend := &RedisBackend{}

	type testInput struct {
		redisClient RedisDB
		key         string
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
			desc: "RedisBackend.Get() throws a redis.Nil error",
			in: testInput{
				redisClient: FakeRedisClient{
					Success:     false,
					ServerError: redis.Nil,
				},
				key: "someKeyThatWontBeFound",
			},
			expected: testExpectedValues{
				value: "",
				err:   utils.NewPBCError(utils.KEY_NOT_FOUND),
			},
		},
		{
			desc: "RedisBackend.Get() throws an error different from redis.Nil",
			in: testInput{
				redisClient: FakeRedisClient{
					Success:     false,
					ServerError: errors.New("some other get error"),
				},
				key: "someKey",
			},
			expected: testExpectedValues{
				value: "",
				err:   errors.New("some other get error"),
			},
		},
		{
			desc: "RedisBackend.Get() doesn't throw an error",
			in: testInput{
				redisClient: FakeRedisClient{
					Success:    true,
					StoredData: map[string]string{"defaultKey": "aValue"},
				},
				key: "defaultKey",
			},
			expected: testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
	}

	for _, tt := range testCases {
		redisBackend.client = tt.in.redisClient

		// Run test
		actualValue, actualErr := redisBackend.Get(context.Background(), tt.in.key)

		// Assertions
		assert.Equal(t, tt.expected.value, actualValue, tt.desc)
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)
	}
}

func TestRedisClientPut(t *testing.T) {
	redisBackend := &RedisBackend{}

	type testInput struct {
		redisClient  RedisDB
		key          string
		valueToStore string
		ttl          int
	}

	type testExpectedValues struct {
		writtenValue   string
		redisClientErr error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testExpectedValues
	}{
		{
			desc: "Try to overwrite already existing key. From redis client documentation, SetNX returns 'false' because no operation is performed",
			in: testInput{
				redisClient: FakeRedisClient{
					Success:     false,
					StoredData:  map[string]string{"key": "original value"},
					ServerError: redis.Nil,
				},
				key:          "key",
				valueToStore: "overwrite value",
				ttl:          10,
			},
			expected: testExpectedValues{
				redisClientErr: utils.NewPBCError(utils.RECORD_EXISTS),
				writtenValue:   "original value",
			},
		},
		{
			desc: "When key does not exist, redis.Nil is returned. Other errors should be interpreted as a server side error. Expect error.",
			in: testInput{
				redisClient: FakeRedisClient{
					Success:     true,
					StoredData:  map[string]string{},
					ServerError: errors.New("A Redis client side error"),
				},
				key:          "someKey",
				valueToStore: "someValue",
				ttl:          10,
			},
			expected: testExpectedValues{
				redisClientErr: errors.New("A Redis client side error"),
			},
		},
		{
			desc: "In Redis, a zero ttl value means no expiration. Expect value to be successfully set",
			in: testInput{
				redisClient: FakeRedisClient{
					StoredData:  map[string]string{},
					Success:     true,
					ServerError: redis.Nil,
				},
				key:          "defaultKey",
				valueToStore: "aValue",
				ttl:          0,
			},
			expected: testExpectedValues{
				writtenValue: "aValue",
			},
		},
		{
			desc: "RedisBackend.Put() successful, no need to set defaultTTL because ttl is greater than zero",
			in: testInput{
				redisClient: FakeRedisClient{
					StoredData:  map[string]string{},
					Success:     true,
					ServerError: redis.Nil,
				},
				key:          "defaultKey",
				valueToStore: "aValue",
				ttl:          1,
			},
			expected: testExpectedValues{
				writtenValue: "aValue",
			},
		},
	}

	for _, tt := range testCases {
		// Assign redis backend client
		redisBackend.client = tt.in.redisClient

		// Run test
		actualErr := redisBackend.Put(context.Background(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assertions
		assert.Equal(t, tt.expected.redisClientErr, actualErr, tt.desc)

		// Put error
		assert.Equal(t, tt.expected.redisClientErr, actualErr, tt.desc)

		if actualErr == nil || actualErr == utils.NewPBCError(utils.RECORD_EXISTS) {
			// Either a value was inserted successfully or the record already existed.
			// Assert data in the backend
			storage, ok := tt.in.redisClient.(FakeRedisClient)
			assert.True(t, ok, tt.desc)
			assert.Equal(t, storage.StoredData[tt.in.key], tt.expected.writtenValue, tt.desc)
		}
	}
}
