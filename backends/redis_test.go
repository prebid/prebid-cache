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
			"RedisBackend.Get() throws a redis.Nil error",
			testInput{
				&errorProneRedisClient{success: false, errorToThrow: redis.Nil},
				"someKeyThatWontBeFound",
			},
			testExpectedValues{
				value: "",
				err:   utils.NewPBCError(utils.KEY_NOT_FOUND),
			},
		},
		{
			"RedisBackend.Get() throws an error different from redis.Nil",
			testInput{
				&errorProneRedisClient{success: false, errorToThrow: errors.New("some other get error")},
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
				&goodRedisClient{key: "defaultKey", value: "aValue"},
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
				&errorProneRedisClient{success: false, errorToThrow: redis.Nil},
				"repeatedKey",
				"overwriteValue",
				10,
			},
			testExpectedValues{
				"",
				utils.NewPBCError(utils.RECORD_EXISTS),
			},
		},
		{
			"RedisBackend.Put() throws an error different from error redis.Nil, which gets returned when key does not exist.",
			testInput{
				&errorProneRedisClient{success: true, errorToThrow: errors.New("some other Redis error")},
				"someKey",
				"someValue",
				10,
			},
			testExpectedValues{
				"",
				errors.New("some other Redis error"),
			},
		},
		{
			"RedisBackend.Put() gets called with zero ttlSeconds, value gets successfully set anyways",
			testInput{
				&goodRedisClient{key: "defaultKey", value: "aValue"},
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
				&goodRedisClient{key: "defaultKey", value: "aValue"},
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
		// Assign redis backend cient
		redisBackend.client = tt.in.redisClient

		// Run test
		actualErr := redisBackend.Put(context.Background(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assert Put error
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)

		// Assert value
		if tt.expected.err == nil {
			storedValue, getErr := redisBackend.Get(context.Background(), tt.in.key)

			assert.NoError(t, getErr, tt.desc)
			assert.Equal(t, tt.expected.value, storedValue, tt.desc)
		}
	}
}

// errorProneRedisClient always throws an error
type errorProneRedisClient struct {
	success      bool
	errorToThrow error
}

func (ec *errorProneRedisClient) Get(ctx context.Context, key string) (string, error) {
	return "", ec.errorToThrow
}

func (ec *errorProneRedisClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	return ec.success, ec.errorToThrow
}

// goodRedisClient does not throw errors
type goodRedisClient struct {
	key   string
	value string
}

func (gc *goodRedisClient) Get(ctx context.Context, key string) (string, error) {
	if key == gc.key {
		return gc.value, nil
	}
	return "", utils.NewPBCError(utils.KEY_NOT_FOUND)
}

func (gc *goodRedisClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	if gc.key != key {
		gc.key = key
	}
	gc.value = value

	return true, nil
}
