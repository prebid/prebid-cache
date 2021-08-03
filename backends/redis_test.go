package backends

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redis"
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
			"RedisBackend.Get() throws a redis.Nil error",
			testInput{
				NewErrorProneRedisClient(false, redis.Nil),
				"someKeyThatWontBeFound",
			},
			testExpectedValues{
				value: "",
				err:   utils.KeyNotFoundError{},
			},
		},
		{
			"RedisBackend.Get() throws an error different from Cassandra ErrNotFound error",
			testInput{
				NewErrorProneRedisClient(false, errors.New("some other get error")),
				"someKey",
			},
			testExpectedValues{
				value: "",
				err:   errors.New("some other get error"),
			},
		},
		{
			"RedisBackend.Get() doesn't throw an error",
			testInput{
				NewGoodRedisClient("defaultKey", "aValue"),
				"defaultKey",
			},
			testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
	}

	for _, tt := range testCases {
		redisBackend.client = tt.in.redisClient

		// Run test
		actualValue, actualErr := redisBackend.Get(context.TODO(), tt.in.key)

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
		value string
		err   error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testExpectedValues
	}{
		{
			"RedisBackend.Put() tries to overwrite already existing key",
			testInput{
				NewErrorProneRedisClient(false, redis.Nil),
				"repeatedKey",
				"overwriteValue",
				10,
			},
			testExpectedValues{
				"",
				utils.RecordExistsError{},
			},
		},
		{
			"RedisBackend.Put() throws an error different from error redis.Nil, which gets returned when key does not exist.",
			testInput{
				NewErrorProneRedisClient(true, errors.New("Some other redis error.")),
				"someKey",
				"someValue",
				10,
			},
			testExpectedValues{
				"",
				errors.New("Some other redis error."),
			},
		},
		{
			"RedisBackend.Put() gets called with zero ttlSeconds, value gets successfully set anyways",
			testInput{
				NewGoodRedisClient("defaultKey", "aValue"),
				"defaultKey",
				"aValue",
				0,
			},
			testExpectedValues{
				"aValue",
				nil,
			},
		},
		{
			"RedisBackend.Put() successful, no need to set defaultTTL because ttl is greater than zero",
			testInput{
				NewGoodRedisClient("defaultKey", "aValue"),
				"defaultKey",
				"aValue",
				1,
			},
			testExpectedValues{
				"aValue",
				nil,
			},
		},
	}

	for _, tt := range testCases {
		// Assign aerospike backend cient
		redisBackend.client = tt.in.redisClient

		// Run test
		actualErr := redisBackend.Put(context.TODO(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assert Put error
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)

		// Assert value
		if tt.expected.err == nil {
			storedValue, getErr := redisBackend.Get(context.TODO(), tt.in.key)

			assert.NoError(t, getErr, tt.desc)
			assert.Equal(t, tt.expected.value, storedValue, tt.desc)
		}
	}
}
