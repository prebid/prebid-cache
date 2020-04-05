package config

import (
	"os"
	"strings"
	"testing"

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

func TestCheckMetricsEnabled(t *testing.T) {
	var otherMetricType MetricsType = "something-else"
	type aTest struct {
		description string
		// In
		metricType        MetricsType
		influxEnabled     bool
		prometheusEnabled bool
		// Out
		expectError         bool
		expectInfluxEnabled bool
		expectPromEnabled   bool
	}
	testCases := []aTest{
		{ //1
			description:         "[1] No metrics enabled nor specified, expect error",
			metricType:          otherMetricType,
			influxEnabled:       false,
			prometheusEnabled:   false,
			expectError:         true,
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
		},
		{ //2
			description:         "[2] Unknown metricsType, Promethus enabled",
			metricType:          otherMetricType,
			influxEnabled:       false,
			prometheusEnabled:   true,
			expectError:         false,
			expectInfluxEnabled: false,
			expectPromEnabled:   true,
		},
		{ //3
			description:         "[3] Unknown metricsType, Influx enabled",
			metricType:          otherMetricType,
			influxEnabled:       true,
			prometheusEnabled:   false,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
		},
		{ //4
			description:         "[4] Unknown metricsType, both Prometheus and Influx enabled",
			metricType:          otherMetricType,
			influxEnabled:       true,
			prometheusEnabled:   true,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
		},
		{ //5
			description:         "[5] \"Influx\" metricsType, no metrics enabled",
			metricType:          MetricsInflux,
			influxEnabled:       false,
			prometheusEnabled:   false,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
		},
		{ //6
			description:         "[6] \"Influx\" metricsType, Promethus enabled",
			metricType:          MetricsInflux,
			influxEnabled:       false,
			prometheusEnabled:   true,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
		},
		{ //7
			description:         "[7] \"Influx\" metricsType, Influx enabled",
			metricType:          MetricsInflux,
			influxEnabled:       true,
			prometheusEnabled:   false,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
		},
		{ //8
			description:         "[8] \"Influx\" metricsType, both Prometheus and Influx enabled",
			metricType:          MetricsInflux,
			influxEnabled:       true,
			prometheusEnabled:   true,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
		},
		{ //9
			description:         "[9] \"none\" metricsType, no metrics enabled",
			metricType:          MetricsNone,
			influxEnabled:       false,
			prometheusEnabled:   false,
			expectError:         false,
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
		},
		{ //10
			description:         "[10] \"none\" metricsType, no metrics enabled",
			metricType:          MetricsNone,
			influxEnabled:       false,
			prometheusEnabled:   false,
			expectError:         false,
			expectInfluxEnabled: false,
			expectPromEnabled:   false,
		},
		{ //11
			description:         "11",
			metricType:          MetricsNone,
			influxEnabled:       true,
			prometheusEnabled:   false,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   false,
		},
		{ //12
			description:         "12",
			metricType:          MetricsNone,
			influxEnabled:       true,
			prometheusEnabled:   true,
			expectError:         false,
			expectInfluxEnabled: true,
			expectPromEnabled:   true,
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

		actualErr := cfg.checkMetricsEnabled()

		assertBoolsEqual(t, "metrics.influx.enabled", cfg.Influx.Enabled, test.expectInfluxEnabled)
		assertBoolsEqual(t, "metrics.prometheus.enabled", cfg.Prometheus.Enabled, test.expectPromEnabled)

		if test.expectError {
			assert.Error(t, actualErr, "We should get a no-metrics-enabled error", test.description)
		} else {
			assert.NoError(t, actualErr, "We shouldn't have gotten a no-metrics-enabled error. Description: %s \n", test.description)
		}
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
