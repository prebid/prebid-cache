package backends

import (
	"context"
	"errors"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestMemcacheGet(t *testing.T) {
	mcBackend := &MemcacheBackend{}

	type testInput struct {
		memcacheClient MemcacheDataStore
		key            string
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
			"Memcache.Get() throws a redis.Nil error",
			testInput{
				NewErrorProneMemcache(memcache.ErrCacheMiss),
				"someKeyThatWontBeFound",
			},
			testExpectedValues{
				value: "",
				err:   utils.KeyNotFoundError{},
			},
		},
		{
			"Memcache.Get() throws an error different from Cassandra ErrNotFound error",
			testInput{
				NewErrorProneMemcache(errors.New("some other get error")),
				"someKey",
			},
			testExpectedValues{
				value: "",
				err:   errors.New("some other get error"),
			},
		},
		{
			"Memcache.Get() doesn't throw an error",
			testInput{
				NewGoodMemcache("defaultKey", "aValue"),
				"defaultKey",
			},
			testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
	}

	for _, tt := range testCases {
		mcBackend.client = tt.in.memcacheClient

		// Run test
		actualValue, actualErr := mcBackend.Get(context.TODO(), tt.in.key)

		// Assertions
		assert.Equal(t, tt.expected.value, actualValue, tt.desc)
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)
	}
}

func TestMemcachePut(t *testing.T) {
	mcBackend := &MemcacheBackend{}

	type testInput struct {
		memcacheClient MemcacheDataStore
		key            string
		valueToStore   string
		ttl            int
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
			"Memcache.Put() throws error",
			testInput{
				NewErrorProneMemcache(memcache.ErrCacheMiss),
				"someKey",
				"someValue",
				10,
			},
			testExpectedValues{
				"",
				memcache.ErrCacheMiss,
			},
		},
		{
			"Memcache.Put() gets called with zero ttlSeconds, value gets successfully set anyways",
			testInput{
				NewGoodMemcache("defaultKey", "aValue"),
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
			"Memcache.Put() successful, no need to set defaultTTL because ttl is greater than zero",
			testInput{
				NewGoodMemcache("defaultKey", "aValue"),
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
		mcBackend.client = tt.in.memcacheClient

		// Run test
		actualErr := mcBackend.Put(context.TODO(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assert Put error
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)

		// Assert value
		if tt.expected.err == nil {
			storedValue, getErr := mcBackend.Get(context.TODO(), tt.in.key)

			assert.NoError(t, getErr, tt.desc)
			assert.Equal(t, tt.expected.value, storedValue, tt.desc)
		}
	}
}
