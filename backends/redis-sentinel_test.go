package backends

import (
	"context"
	"errors"
	"testing"

	"github.com/prebid/prebid-cache/utils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisSentinelClientGet(t *testing.T) {
	redisSentinelBackend := &RedisSentinelBackend{}

	type testInput struct {
		client RedisDB
		key    string
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
			desc: "RedisSentinelBackend.Get() throws a redis.Nil error",
			in: testInput{
				client: FakeRedisClient{
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
				client: FakeRedisClient{
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
				client: FakeRedisClient{
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
		redisSentinelBackend.client = tt.in.client

		// Run test
		actualValue, actualErr := redisSentinelBackend.Get(context.Background(), tt.in.key)

		// Assertions
		assert.Equal(t, tt.expected.value, actualValue, tt.desc)
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)
	}
}

func TestRedisSentinelClientPut(t *testing.T) {
	redisSentinelBackend := &RedisSentinelBackend{}

	type testInput struct {
		redisSentinelClient RedisDB
		key                 string
		valueToStore        string
		ttl                 int
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
				redisSentinelClient: FakeRedisClient{
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
				redisSentinelClient: FakeRedisClient{
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
				redisSentinelClient: FakeRedisClient{
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
				redisSentinelClient: FakeRedisClient{
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
		redisSentinelBackend.client = tt.in.redisSentinelClient

		// Run test
		actualErr := redisSentinelBackend.Put(context.Background(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assertions
		assert.Equal(t, tt.expected.redisClientErr, actualErr, tt.desc)

		// Put error
		assert.Equal(t, tt.expected.redisClientErr, actualErr, tt.desc)

		if actualErr == nil || actualErr == utils.NewPBCError(utils.RECORD_EXISTS) {
			// Either a value was inserted successfully or the record already existed.
			// Assert data in the backend
			storage, ok := tt.in.redisSentinelClient.(FakeRedisClient)
			assert.True(t, ok, tt.desc)
			assert.Equal(t, tt.expected.writtenValue, storage.StoredData[tt.in.key], tt.desc)
		}
	}
}
