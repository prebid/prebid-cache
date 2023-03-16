package backends

import (
	"context"
	"errors"
	"testing"

	"github.com/google/gomemcache/memcache"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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
			"Memcache.Get() throws a memcache.ErrCacheMiss error",
			testInput{
				&ErrorProneMemcache{ServerError: memcache.ErrCacheMiss},
				"someKeyThatWontBeFound",
			},
			testExpectedValues{
				value: "",
				err:   utils.NewPBCError(utils.KEY_NOT_FOUND),
			},
		},
		{
			"Memcache.Get() throws an error different from Cassandra ErrNotFound error",
			testInput{
				&ErrorProneMemcache{ServerError: errors.New("some other get error")},
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
				&GoodMemcache{StoredData: map[string]string{"defaultKey": "aValue"}},
				"defaultKey",
			},
			testExpectedValues{
				value: "aValue",
				err:   nil,
			},
		},
	}

	for _, tt := range testCases {
		mcBackend.memcache = tt.in.memcacheClient

		// Run test
		actualValue, actualErr := mcBackend.Get(context.Background(), tt.in.key)

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
			"Memcache.Put() throws non-ErrNotStored error",
			testInput{
				&ErrorProneMemcache{ServerError: memcache.ErrServerError},
				"someKey",
				"someValue",
				10,
			},
			testExpectedValues{
				"",
				memcache.ErrServerError,
			},
		},
		{
			"Memcache.Put() throws ErrNotStored error",
			testInput{
				&ErrorProneMemcache{ServerError: memcache.ErrNotStored},
				"someKey",
				"someValue",
				10,
			},
			testExpectedValues{
				"",
				utils.NewPBCError(utils.RECORD_EXISTS),
			},
		},
		{
			"Memcache.Put() successful",
			testInput{
				&GoodMemcache{StoredData: map[string]string{"defaultKey": "aValue"}},
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
		mcBackend.memcache = tt.in.memcacheClient

		// Run test
		actualErr := mcBackend.Put(context.Background(), tt.in.key, tt.in.valueToStore, tt.in.ttl)

		// Assert Put error
		assert.Equal(t, tt.expected.err, actualErr, tt.desc)

		// Assert value
		if tt.expected.err == nil {
			storedValue, getErr := mcBackend.Get(context.Background(), tt.in.key)

			assert.NoError(t, getErr, tt.desc)
			assert.Equal(t, tt.expected.value, storedValue, tt.desc)
		}
	}
}

func TestNewMemcacheBackend(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		desc               string
		inCfg              config.Memcache
		expectPanic        bool
		expectedLogEntries []logEntry
	}{
		{
			desc: "PollIntervalSeconds is less than 1, memcache.NewDiscoveryClient will throw an error, expect Panic",
			inCfg: config.Memcache{
				ConfigHost:          "somehost.com:0000",
				PollIntervalSeconds: 0,
			},
			expectPanic: true,
			expectedLogEntries: []logEntry{
				{
					msg: "Discovery polling duration is invalid",
					lvl: logrus.FatalLevel,
				},
			},
		},
		{
			desc: "PollIntervalSeconds is greater than 1, memcache.NewDiscoveryClient will return a valid memcache client",
			inCfg: config.Memcache{
				ConfigHost:          "somehost.com:0000",
				PollIntervalSeconds: 2,
			},
			expectedLogEntries: []logEntry{},
		},
		{
			desc: "ConfigHost is an empty string, memcache client gets created calling memcache.New(cfg.Hosts...)",
			inCfg: config.Memcache{
				Hosts: []string{"somehost.com:0000"},
			},
			expectedLogEntries: []logEntry{},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, test := range testCases {
		if test.expectPanic {
			assert.Panics(t, func() { NewMemcacheBackend(test.inCfg) }, "memcache.NewDiscoveryClient() should have thrown an error and didn't, hence the panic didn't happen")
		} else {
			NewMemcacheBackend(test.inCfg)
		}

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
