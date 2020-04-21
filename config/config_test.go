package config

import (
	_ "fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestSampleConfig(t *testing.T) {
	cfg := Configuration{}
	v := newViperFromSample(t)

	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	assertIntsEqual(t, "port", cfg.Port, 2424)
	assertIntsEqual(t, "admin_port", cfg.AdminPort, 2525)
	assertStringsEqual(t, "log.level", string(cfg.Log.Level), "info")
	assertBoolsEqual(t, "rate_limiter.enabled", cfg.RateLimiting.Enabled, true)
	assertInt64sEqual(t, "rate_limiter.num_requests", cfg.RateLimiting.MaxRequestsPerSecond, 100)
	assertIntsEqual(t, "request_limits.max_size_bytes", cfg.RequestLimits.MaxSize, 10240)
	assertIntsEqual(t, "request_limits.max_num_values", cfg.RequestLimits.MaxNumValues, 10)
	assertIntsEqual(t, "request_limits.max_ttl_seconds", cfg.RequestLimits.MaxTTLSeconds, 5000)
	assertStringsEqual(t, "backend.type", string(cfg.Backend.Type), "memory")
	assertIntsEqual(t, "backend.aerospike.default_ttl_seconds", cfg.Backend.Aerospike.DefaultTTL, 3600)
	assertStringsEqual(t, "backend.aerospike.host", cfg.Backend.Aerospike.Host, "aerospike.prebid.com")
	assertIntsEqual(t, "backend.aerospike.port", cfg.Backend.Aerospike.Port, 3000)
	assertStringsEqual(t, "backend.aerospike.namespace", cfg.Backend.Aerospike.Namespace, "whatever")
	assertStringsEqual(t, "backend.azure.account", cfg.Backend.Azure.Account, "azure-account-here")
	assertStringsEqual(t, "backend.azure.key", cfg.Backend.Azure.Key, "azure-key-here")
	assertStringsEqual(t, "backend.cassandra.hosts", cfg.Backend.Cassandra.Hosts, "127.0.0.1")
	assertStringsEqual(t, "backend.cassandra.keyspace", cfg.Backend.Cassandra.Keyspace, "prebid")
	assertStringsEqual(t, "backend.memcache.hosts", cfg.Backend.Memcache.Hosts[0], "10.0.0.1:11211")
	assertIntsEqual(t, "backend.redis.port", cfg.Backend.Redis.Port, 6379)
	assertIntsEqual(t, "backend.redis.db", cfg.Backend.Redis.Db, 1)
	assertStringsEqual(t, "backend.redis.host", cfg.Backend.Redis.Host, "127.0.0.1")
	assertStringsEqual(t, "backend.redis.password", cfg.Backend.Redis.Password, "")
	assertBoolsEqual(t, "backend.redis.tls.enabled", cfg.Backend.Redis.TLS.Enabled, false)
	assertBoolsEqual(t, "backend.redis.tls.insecure_skip_verify", cfg.Backend.Redis.TLS.InsecureSkipVerify, false)
	assertStringsEqual(t, "compression.type", string(cfg.Compression.Type), "snappy")
	assertStringsEqual(t, "metrics.type", string(cfg.Metrics.Type), "none")
	assertStringsEqual(t, "metrics.influx.host", cfg.Metrics.Influx.Host, "default-metrics-host")
	assertStringsEqual(t, "metrics.influx.database", cfg.Metrics.Influx.Database, "default-metrics-database")
	assertStringsEqual(t, "metrics.influx.username", cfg.Metrics.Influx.Username, "metrics-username")
	assertStringsEqual(t, "metrics.influx.password", cfg.Metrics.Influx.Password, "metrics-password")
	assertBoolsEqual(t, "metrics.influx.enabled", cfg.Metrics.Influx.Enabled, true)
	assertIntsEqual(t, "metrics.prometheus.port", cfg.Metrics.Prometheus.Port, 8080)
	assertStringsEqual(t, "metrics.prometheus.namespace", cfg.Metrics.Prometheus.Namespace, "default-prometheus-namespace")
	assertStringsEqual(t, "metrics.prometheus.subsystem", cfg.Metrics.Prometheus.Subsystem, "default-prometheus-subsystem")
	assertIntsEqual(t, "metrics.prometheus.timeout_ms", cfg.Metrics.Prometheus.TimeoutMillisRaw, 100)
	assertBoolsEqual(t, "metrics.prometheus.enabled", cfg.Metrics.Prometheus.Enabled, true)
}

func TestEnvConfig(t *testing.T) {
	defer forceEnv(t, "PBC_PORT", "2000")()
	defer forceEnv(t, "PBC_COMPRESSION_TYPE", "none")()

	v := viper.New()
	setConfigDefaults(v)
	setEnvVars(v)
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(sampleConfig)); err != nil {
		t.Errorf("Failed to read sample file: %v", err)
	}

	cfg := Configuration{}
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	assertIntsEqual(t, "port", cfg.Port, 2000)
	assertStringsEqual(t, "compression.type", string(cfg.Compression.Type), "none")
}

func TestLogValidateAndLog(t *testing.T) {
	hook := test.NewGlobal()

	configLogObject := Log{
		Level: Debug,
	}

	configLogObject.validateAndLog()

	if !assert.Equal(t, 1, len(hook.Entries), "No entries were logged to logrus.") {
		return
	}
	if !assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level) {
		return
	}
	if !assert.Equal(t, "config.log.level: debug", hook.LastEntry().Message) {
		return
	}

	hook.Reset()
	assert.Nil(t, hook.LastEntry())
}

func TestCheckMetricsEnabled(t *testing.T) {
	/* | cfg.Influx.Enabled | cfg.Prometheus.Enabled | cfg.Type ||
		---|--------------------|------------------------|----------||-------------
		  1|      false         |       false            | "none"   ||  go ahead no metrics
		  2|      false         |       true             | "none"   ||  prometheus metrics
		  3|      true          |       false            | "none"   ||  influx metrics
		  4|      true          |       true             | "none"   ||  both prometheus and influx

		  1|      false         |       false            | "influx" ||  influx metrics
		  2|      false         |       true             | "influx" ||  prometheus metrics
		  3|      true          |       false            | "influx" ||  influx metrics
		  4|      true          |       true             | "influx" ||  both prometheus and influx

		  1|      false         |       false            | "Trendalyze" ||  exit error
		  2|      false         |       true             | "Trendalyze" ||  prometheus metrics
		  3|      true          |       false            | "Trendalyze" ||  influx metrics
		  4|      true          |       true             | "Trendalyze" ||  both prometheus and influx

	// metricType = "none" area
		description: "[1] metricType = \"none\"; both prometheus and influx flags off. Continue with no metrics, no log to assert",
			influxEnabled:       false,
			prometheusEnabled:   false,
			metricType:          "none",
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
			expectError:         false,
		description: "[2] metricType = \"none\"; prometheus flag on",
			influxEnabled:       false,
			prometheusEnabled:   true,
			metricType:          "none",
			expectInfluxEnabled: false,
			expectPromEnabled:   true,
			expectError:         false,
		description: "[3] metricType = \"none\"; influx flag on",
			influxEnabled:       true,
			prometheusEnabled:   false,
			metricType:          "none",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		description: "[4] both prometheus and influx enabled flags on",
			influxEnabled:       true,
			prometheusEnabled:   true,
			metricType:          "none",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
	// metricType = "influx" area
		description: "[5] metricType = \"influx\"; both prometheus and influx flags off",
			influxEnabled:       false,
			prometheusEnabled:   false,
			metricType:          "influx",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		description: "[6] metricType = \"influx\"; prometheus flags on",
			influxEnabled:       false,
			prometheusEnabled:   true,
			metricType:          "influx",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
		description: "[7] metricType = \"influx\"; inlfux flags on",
			influxEnabled:       true,
			prometheusEnabled:   false,
			metricType:          "influx",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		description: "[8] metricType = \"influx\"; prometheus and inlfux flags on",
			influxEnabled:       true,
			prometheusEnabled:   true,
			metricType:          "influx",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
	// other metrics system in the `cfg.Type` field
		description: "[9] metricType = \"trendalyze\"; both prometheus and influx flags off. Exit error",
			influxEnabled:       false,
			prometheusEnabled:   false,
			metricType:          "trendalyze",
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
			expectError:         true,
		description: "[10] metricType = \"trendalyze\"; prometheus flags on.",
			influxEnabled:       false,
			prometheusEnabled:   true,
			metricType:          "trendalyze",
			expectInfluxEnabled: false,
			expectPromEnabled:   true,
			expectError:         false,
		description: "[11] metricType = \"trendalyze\"; influx flags on."
			influxEnabled:       true,
			prometheusEnabled:   false,
			metricType:          "trendalyze",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		description: "[12] metricType = \"trendalyze\"; prometheus and inlfux flags on"
			influxEnabled:       true,
			prometheusEnabled:   true,
			metricType:          "trendalyze",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
	*/
	type aTest struct {
		description string
		// In
		influxEnabled     bool
		prometheusEnabled bool
		metricType        MetricsType
		// Out
		expectInfluxEnabled bool
		expectPromEnabled   bool
		expectError         bool
	}
	testCases := []aTest{
		{
			description:         "[1] metricType = \"none\"; both prometheus and influx flags off. Continue with no metrics, no log to assert",
			influxEnabled:       false,
			prometheusEnabled:   false,
			metricType:          "none",
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
			expectError:         false,
		},
		{
			description:         "[2] metricType = \"none\"; prometheus flag on",
			influxEnabled:       false,
			prometheusEnabled:   true,
			metricType:          "none",
			expectInfluxEnabled: false,
			expectPromEnabled:   true,
			expectError:         false,
		},
		{
			description:         "[3] metricType = \"none\"; InfluxDB flag on",
			influxEnabled:       true,
			prometheusEnabled:   false,
			metricType:          "none",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		},
		{
			description:         "[4] metricType = \"none\"; Both prometheus and influx flags on",
			influxEnabled:       true,
			prometheusEnabled:   true,
			metricType:          "none",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
		},
		// -------
		{
			description:         "[5] metricType = \"influx\"; both prometheus and influx flags off",
			influxEnabled:       false,
			prometheusEnabled:   false,
			metricType:          "influx",
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
			expectError:         false,
		},
		{
			description:         "[6] metricType = \"influx\"; prometheus flags on",
			influxEnabled:       false,
			prometheusEnabled:   true,
			metricType:          "influx",
			expectInfluxEnabled: false,
			expectPromEnabled:   true,
			expectError:         false,
		},
		{
			description:         "[7] metricType = \"influx\"; inlfux flags on",
			influxEnabled:       true,
			prometheusEnabled:   false,
			metricType:          "influx",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		},
		{
			description:         "[8] metricType = \"influx\"; prometheus and inlfux flags on",
			influxEnabled:       true,
			prometheusEnabled:   true,
			metricType:          "influx",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
		},
		// -------
		{
			description:         "[9] metricType = \"trendalyze\"; both prometheus and influx flags off. Exit error",
			influxEnabled:       false,
			prometheusEnabled:   false,
			metricType:          "trendalyze",
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
			expectError:         true,
		},
		{
			description:         "[10] metricType = \"trendalyze\"; prometheus flags on.",
			influxEnabled:       false,
			prometheusEnabled:   true,
			metricType:          "trendalyze",
			expectInfluxEnabled: false,
			expectPromEnabled:   true,
			expectError:         false,
		},
		{
			description:         "[11] metricType = \"trendalyze\"; influx flags on.",
			influxEnabled:       true,
			prometheusEnabled:   false,
			metricType:          "trendalyze",
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
			expectError:         false,
		},
		{
			description:         "[12] metricType = \"trendalyze\"; prometheus and inlfux flags on",
			influxEnabled:       true,
			prometheusEnabled:   true,
			metricType:          "trendalyze",
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
			expectError:         false,
		},
	}
	cfg := &Metrics{
		Influx: InfluxMetrics{
			Host:     "http://fakeurl.com",
			Database: "database-value",
		},
		Prometheus: PrometheusMetrics{
			Port:      8080,
			Namespace: "prebid",
			Subsystem: "cache",
		},
	}
	for _, test := range testCases {
		cfg.Type = test.metricType
		cfg.Influx.Enabled = test.influxEnabled
		cfg.Prometheus.Enabled = test.prometheusEnabled

		actualErr := cfg.validateAndLog()

		assertBoolsEqual(t, "metrics.influx.enabled", cfg.Influx.Enabled, test.expectInfluxEnabled)
		assertBoolsEqual(t, "metrics.prometheus.enabled", cfg.Prometheus.Enabled, test.expectPromEnabled)

		if test.expectError {
			assert.Error(t, actualErr, "We should get a no-metrics-enabled error", test.description)
		} else {
			assert.NoError(t, actualErr, "We shouldn't have gotten a no-metrics-enabled error. Description: %s \n", test.description)
		}
	}
}

func TestInfluxValidateAndLog(t *testing.T) {
	hook := test.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}
	type aTest struct {
		description string
		// In
		influxConfig *InfluxMetrics
		//out
		expectError     bool
		expectedLogInfo []logComponents
	}
	testCases := []aTest{
		{
			description: "[1] both InfluxDB host and database blank, expect error",
			influxConfig: &InfluxMetrics{
				Host:     "",
				Database: "",
			},
			//out
			expectError:     true,
			expectedLogInfo: nil,
		},
		{
			description: "[2] InfluxDB host blank, expect error",
			influxConfig: &InfluxMetrics{
				Host:     "",
				Database: "database-value",
			},
			//out
			expectError:     true,
			expectedLogInfo: nil,
		},
		{
			description: "[3] InfluxDB database blank, expect error",
			influxConfig: &InfluxMetrics{
				Host:     "http://fakeurl.com",
				Database: "",
			},
			//out
			expectError:     true,
			expectedLogInfo: nil,
		},
		{
			description: "[4] Valid InfluxDB host and database, expect log.Info",
			influxConfig: &InfluxMetrics{
				Host:     "http://fakeurl.com",
				Database: "database-value",
			},
			//out
			expectError: false,
			expectedLogInfo: []logComponents{
				{
					msg: "config.metrics.influx.host: http://fakeurl.com",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.influx.database: database-value",
					lvl: logrus.InfoLevel,
				},
			},
		},
	}
	for j, test := range testCases {
		//run test
		err := test.influxConfig.validateAndLog()

		//If error, assert
		if test.expectError {
			assert.Error(t, err, "Error expected in test number %d", j)
		} else {
			//If not error, assert
			assert.NoError(t, err, "No Error expected in test number %d", j)

			//Further assertions
			if !assert.Equal(t, len(test.expectedLogInfo), len(hook.Entries), "No entries were logged to logrus.") {
				return
			}
			for i := 0; i < len(hook.Entries); i++ {
				assert.Equal(t, test.expectedLogInfo[i].msg, hook.Entries[i].Message)
				assert.Equal(t, test.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Expected Info entry in log")
			}
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
		}
	}
}

func TestPrometheusValidateAndLog(t *testing.T) {
	hook := test.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}
	type aTest struct {
		description string
		// In
		prometheusConfig *PrometheusMetrics
		//out
		expectError     bool
		expectedLogInfo []logComponents
	}
	testCases := []aTest{
		{
			description: "[1] Port invalid, Namespace valid, Subsystem valid. Expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      0,
				Namespace: "prebid",
				Subsystem: "cache",
			},
			//out
			expectError:     true,
			expectedLogInfo: nil,
		},
		{
			description: "[2] Port valid, Namespace invalid, Subsystem valid. Expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "",
				Subsystem: "cache",
			},
			//out
			expectError:     true,
			expectedLogInfo: nil,
		},
		{
			description: "[3] Port valid, Namespace valid, Subsystem invalid. Expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "",
			},
			//out
			expectError:     true,
			expectedLogInfo: nil,
		},
		{
			description: "[3] Port valid, Namespace valid, Subsystem valid. Expect elements in log",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "cache",
			},
			//out
			expectError: false,
			expectedLogInfo: []logComponents{
				{
					msg: "config.metrics.prometheus.namespace: prebid",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.prometheus.subsystem: cache",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.prometheus.port: 8080",
					lvl: logrus.InfoLevel,
				},
			},
		},
	}
	for j, test := range testCases {
		//run test
		err := test.prometheusConfig.validateAndLog()

		//If error, assert
		if test.expectError {
			assert.Error(t, err, "Error expected in test number %d", j)
		} else {
			//If not error, assert
			assert.NoError(t, err, "No Error expected in test number %d", j)

			//Further assertions
			if !assert.Equal(t, len(test.expectedLogInfo), len(hook.Entries), "No entries were logged to logrus.") {
				return
			}
			for i := 0; i < len(hook.Entries); i++ {
				assert.Equal(t, test.expectedLogInfo[i].msg, hook.Entries[i].Message)
				assert.Equal(t, test.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Expected Info entry in log")
			}
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
		}
	}
}

func TestCompressionValidateAndLog(t *testing.T) {
	hook := test.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		description     string
		compConf        *Compression
		expectFatal     bool
		expectedLogInfo []logComponents
	}{
		{
			description: "[1] Valid compression type expect to log.Infof",
			compConf:    &Compression{Type: CompressionSnappy},
			expectFatal: false,
			expectedLogInfo: []logComponents{
				{
					msg: "config.compression.type: snappy",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description: "[2] Invalid compression type expect to log.Fatal",
			compConf:    &Compression{Type: CompressionType("invalid")},
			expectFatal: true,
			expectedLogInfo: []logComponents{
				{
					msg: `invalid config.compression.type: invalid. It must be "none" or "snappy"`,
					lvl: logrus.FatalLevel,
				},
			},
		},
	}

	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for j, tc := range testCases {
		fatal = false
		tc.compConf.validateAndLog()

		if assert.Equal(t, len(tc.expectedLogInfo), len(hook.Entries), "No entries were logged to logrus in test %d: len(tc.expectedLogInfo) = %d len(hook.Entries) = %d", j, len(tc.expectedLogInfo), len(hook.Entries)) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message)
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Expected Info entry in log")
			}
		} else {
			return
		}

		assert.Equal(t, tc.expectFatal, fatal)

		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func newViperFromSample(t *testing.T) *viper.Viper {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(sampleConfig)); err != nil {
		t.Errorf("Failed to read sample file: %v", err)
	}
	return v
}

const sampleConfig = `
port: 2424
admin_port: 2525
log:
  level: "info"
rate_limiter:
  enabled: true
  num_requests: 100
request_limits:
  max_size_bytes: 10240
  max_num_values: 10
  max_ttl_seconds: 5000
backend:
  type: "memory"
  aerospike:
    default_ttl_seconds: 3600
    host: "aerospike.prebid.com"
    port: 3000
    namespace: "whatever"
  memcache:
    hosts: "10.0.0.1:11211"
  cassandra:
    hosts: "127.0.0.1"
    keyspace: "prebid"
  azure:
    account: "azure-account-here"
    key: "azure-key-here"
  redis:
    host: "127.0.0.1"
    port: 6379
    password: ""
    db: 1
    tls:
      enabled: false
      insecure_skip_verify: false
compression:
  type: "snappy"
metrics:
  type: "none"
  influx:
    host: "default-metrics-host"
    database: "default-metrics-database"
    username: "metrics-username"
    password: "metrics-password"
    enabled: true
  prometheus:
    port: 8080
    namespace: "default-prometheus-namespace"
    subsystem: "default-prometheus-subsystem"
    timeout_ms: 100
    enabled: true
`

func assertBoolsEqual(t *testing.T, path string, actual bool, expected bool) {
	t.Helper()
	if actual != expected {
		t.Errorf("%s value %t did not equal expected %t", path, actual, expected)
	}
}

func assertIntsEqual(t *testing.T, path string, actual int, expected int) {
	t.Helper()
	if actual != expected {
		t.Errorf("%s value %d did not equal expected %d", path, actual, expected)
	}
}

func assertInt64sEqual(t *testing.T, path string, actual int64, expected int) {
	t.Helper()
	if actual != int64(expected) {
		t.Errorf("%s value %d did not equal expected %d", path, actual, expected)
	}
}

func assertStringsEqual(t *testing.T, path string, actual string, expected string) {
	t.Helper()
	if actual != expected {
		t.Errorf(`%s value "%s" did not equal expected "%s"`, path, actual, expected)
	}
}

// forceEnv sets an environment variable to a certain value, and returns a function which resets it to its original value.
func forceEnv(t *testing.T, key string, val string) func() {
	orig, set := os.LookupEnv(key)
	err := os.Setenv(key, val)
	if err != nil {
		t.Fatalf("Error setting evnvironment %s", key)
	}
	if set {
		return func() {
			if os.Setenv(key, orig) != nil {
				t.Fatalf("Error unsetting evnvironment %s", key)
			}
		}
	} else {
		return func() {
			if os.Unsetenv(key) != nil {
				t.Fatalf("Error unsetting evnvironment %s", key)
			}
		}
	}
}
