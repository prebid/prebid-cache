package config

import (
	"context"
	"testing"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/prebid/prebid-cache/utils"

	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestApplyCompression(t *testing.T) {
	testCases := []struct {
		desc                string
		inConfig            config.Compression
		expectedBackendType backends.Backend
	}{
		{
			desc: "Compression type none, expect the default fakeBackend",
			inConfig: config.Compression{
				Type: config.CompressionNone,
			},
			expectedBackendType: &fakeBackend{},
		},
		{
			desc: "Compression type snappy, expect the the backend to be a snappyCompressor backend",
			inConfig: config.Compression{
				Type: config.CompressionSnappy,
			},
			expectedBackendType: compression.SnappyCompress(&fakeBackend{}),
		},
	}

	for _, tc := range testCases {
		// set test
		sampleBackend := &fakeBackend{}

		// run
		actualBackend := applyCompression(tc.inConfig, sampleBackend)

		// assertions
		assert.IsType(t, tc.expectedBackendType, actualBackend, tc.desc)
	}
}

func TestApplyUnknownCompression(t *testing.T) {
	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := logrusTest.NewGlobal()
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	// Input and expected values
	inConfig := config.Compression{Type: "unknown"}
	expectedLogMessage := "Unknown compression type: unknown"
	expectedLogLevel := logrus.FatalLevel

	// run and assert it panics
	panicTestFunction := func() {
		applyCompression(inConfig, &fakeBackend{})
	}
	assert.Panics(t, panicTestFunction, "Unknown compression type should have made applyCompression to panic")

	// assertions
	assert.Equal(t, expectedLogMessage, hook.Entries[0].Message, "Expected log message not found")
	assert.Equal(t, expectedLogLevel, hook.Entries[0].Level, "Unexpected log level")
}

func TestNewMemoryOrMemcacheBackend(t *testing.T) {
	testCases := []struct {
		desc            string
		inConfig        config.Backend
		expectedBackend backends.Backend
	}{
		{
			desc:            "Memory",
			inConfig:        config.Backend{Type: config.BackendMemory},
			expectedBackend: backends.NewMemoryBackend(),
		},
		{
			desc:            "Memcache",
			inConfig:        config.Backend{Type: config.BackendMemcache},
			expectedBackend: &backends.MemcacheBackend{},
		},
	}

	for _, tc := range testCases {
		mockMetrics := metricstest.CreateMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{
				&mockMetrics,
			},
		}

		// run
		actualBackend := newBaseBackend(tc.inConfig, m)

		// assertions
		assert.IsType(t, tc.expectedBackend, actualBackend, tc.desc)
	}

}

func TestNewBaseBackend(t *testing.T) {
	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := logrusTest.NewGlobal()
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	type logEntry struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		desc                string
		inConfig            config.Backend
		inExpectPanic       bool
		expectedBackendType backends.Backend
		expectedLogEntries  []logEntry
	}{
		{
			desc:          "unknown",
			inConfig:      config.Backend{Type: "unknown"},
			inExpectPanic: true,
			expectedLogEntries: []logEntry{
				{msg: "Unknown backend type: unknown", lvl: logrus.FatalLevel},
			},
		},
		{
			desc:          "Cassandra",
			inConfig:      config.Backend{Type: config.BackendCassandra},
			inExpectPanic: true,
			expectedLogEntries: []logEntry{
				{msg: "Error creating Cassandra backend: ", lvl: logrus.FatalLevel},
			},
		},
		{
			desc:     "Aerospike",
			inConfig: config.Backend{Type: config.BackendAerospike},
			expectedLogEntries: []logEntry{
				{msg: "Error creating Aerospike backend: ", lvl: logrus.FatalLevel},
			},
		},
		{
			desc:          "Redis",
			inConfig:      config.Backend{Type: config.BackendRedis},
			inExpectPanic: true,
			expectedLogEntries: []logEntry{
				{msg: "Error creating Redis backend: ", lvl: logrus.FatalLevel},
			},
		},
		{
			desc:          "Ignite",
			inConfig:      config.Backend{Type: config.BackendIgnite},
			inExpectPanic: true,
			expectedLogEntries: []logEntry{
				{
					msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
					lvl: logrus.FatalLevel,
				},
			},
		},
	}

	for _, tc := range testCases {
		mockMetrics := metricstest.CreateMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{&mockMetrics},
		}

		// run
		panicTestFunction := func() {
			newBaseBackend(tc.inConfig, m)
		}

		if tc.inExpectPanic {
			if !assert.Panics(t, panicTestFunction, "%s backend initialized in this test should error and panic.", tc.desc) {
				continue
			}
		} else {
			if !assert.NotPanics(t, panicTestFunction, "%s backend initialized in this test should not panic.", tc.desc) {
				continue
			}
		}

		// assertions
		assert.Len(t, hook.Entries, len(tc.expectedLogEntries), tc.desc)
		if len(tc.expectedLogEntries) > 0 {
			for i := 0; i < len(tc.expectedLogEntries); i++ {
				assert.Contains(t, hook.Entries[i].Message, tc.expectedLogEntries[i].msg, tc.desc)
				assert.Equal(t, tc.expectedLogEntries[i].lvl, hook.Entries[i].Level, tc.desc)
			}
		}
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestGetMaxTTLSeconds(t *testing.T) {
	const SIXTY_SECONDS = 60
	type testCases struct {
		desc                  string
		inConfig              config.Configuration
		expectedMaxTTLSeconds int
	}
	tests := []struct {
		groupDesc string
		unitTests []testCases
	}{
		{
			groupDesc: "Cassandra backend",
			unitTests: []testCases{
				{
					desc: "cfg.RequestLimits.MaxTTLSeconds > utils.CASSANDRA_DEFAULT_TTL_SECONDS",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendCassandra,
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: utils.REQUEST_MAX_TTL_SECONDS,
						},
					},
					expectedMaxTTLSeconds: utils.CASSANDRA_DEFAULT_TTL_SECONDS,
				},
				{
					desc: "cfg.RequestLimits.MaxTTLSeconds <= utils.CASSANDRA_DEFAULT_TTL_SECONDS",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendCassandra,
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: 10,
						},
					},
					expectedMaxTTLSeconds: 10,
				},
			},
		},
		{
			groupDesc: "Aerospike backend",
			unitTests: []testCases{
				{
					desc: "cfg.Backend.Aerospike.DefaultTTL <= 0",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendAerospike,
							Aerospike: config.Aerospike{
								DefaultTTLSecs: 0,
							},
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: 10,
						},
					},
					expectedMaxTTLSeconds: 10,
				},
				{
					desc: "cfg.Backend.Aerospike.DefaultTTL > 0 and maxTTLSeconds < cfg.Backend.Aerospike.DefaultTTL ",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendAerospike,
							Aerospike: config.Aerospike{
								DefaultTTLSecs: 100,
							},
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: 10,
						},
					},
					expectedMaxTTLSeconds: 10,
				},
				{
					desc: "cfg.Backend.Aerospike.DefaultTTL > 0 and maxTTLSeconds > cfg.Backend.Aerospike.DefaultTTL ",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendAerospike,
							Aerospike: config.Aerospike{
								DefaultTTLSecs: 1,
							},
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: 10,
						},
					},
					expectedMaxTTLSeconds: 1,
				},
			},
		},
		{
			groupDesc: "Redis backend",
			unitTests: []testCases{
				{
					desc: "cfg.Backend.Redis.Expiration <= 0",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendRedis,
							Redis: config.Redis{
								ExpirationMinutes: 0,
							},
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: 10,
						},
					},
					expectedMaxTTLSeconds: 10,
				},
				{
					desc: "cfg.Backend.Redis.Expiration > 0 and maxTTLSeconds < cfg.Backend.Redis.Expiration*60",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendRedis,
							Redis: config.Redis{
								ExpirationMinutes: 1,
							},
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: 10,
						},
					},
					expectedMaxTTLSeconds: 10,
				},
				{
					desc: "cfg.Backend.Redis.Expiration > 0 and maxTTLSeconds > cfg.Backend.Redis.Expiration",
					inConfig: config.Configuration{
						Backend: config.Backend{
							Type: config.BackendRedis,
							Redis: config.Redis{
								ExpirationMinutes: 1,
							},
						},
						RequestLimits: config.RequestLimits{
							MaxTTLSeconds: utils.REQUEST_MAX_TTL_SECONDS,
						},
					},
					expectedMaxTTLSeconds: SIXTY_SECONDS,
				},
			},
		},
	}

	for _, tgroup := range tests {
		for _, tc := range tgroup.unitTests {
			assert.Equal(t, tc.expectedMaxTTLSeconds, getMaxTTLSeconds(tc.inConfig), tc.desc)
		}
	}
}

type fakeBackend struct{}

func (c *fakeBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return nil
}

func (c *fakeBackend) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}
