package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDefaults(t *testing.T) {
	v := viper.New()

	setConfigDefaults(v)

	cfg := Configuration{}
	err := v.Unmarshal(&cfg)
	assert.NoError(t, err, "Failed to unmarshal config: %v", err)

	assertIntsEqual(t, "port", cfg.Port, 2424)
	assertIntsEqual(t, "admin_port", cfg.AdminPort, 2525)
	assertStringsEqual(t, "index_response", cfg.IndexResponse, "This application stores short-term data for use in Prebid.")
	assertStringsEqual(t, "log.level", string(cfg.Log.Level), "info")
	assertStringsEqual(t, "backend.type", string(cfg.Backend.Type), "memory")
	assertStringsEqual(t, "backend.aerospike.host", cfg.Backend.Aerospike.Host, "")
	assertIntsEqual(t, "backend.aerospike.port", cfg.Backend.Aerospike.Port, 0)
	assertStringsEqual(t, "backend.aerospike.namespace", cfg.Backend.Aerospike.Namespace, "")
	assertIntsEqual(t, "backend.aerospike.default_ttl_seconds", cfg.Backend.Aerospike.DefaultTTL, 0)
	assertStringsEqual(t, "backend.azure.account", cfg.Backend.Azure.Account, "")
	assertStringsEqual(t, "backend.azure.key", cfg.Backend.Azure.Key, "")
	assertStringsEqual(t, "backend.cassandra.hosts", cfg.Backend.Cassandra.Hosts, "")
	assertStringsEqual(t, "backend.cassandra.keyspace", cfg.Backend.Cassandra.Keyspace, "")
	assert.Equal(t, []string{}, cfg.Backend.Memcache.Hosts, "backend.memcache.hosts should be a zero-lenght slice of strings")
	assertStringsEqual(t, "backend.redis.host", cfg.Backend.Redis.Host, "")
	assertIntsEqual(t, "backend.redis.port", cfg.Backend.Redis.Port, 0)
	assertStringsEqual(t, "backend.redis.password", cfg.Backend.Redis.Password, "")
	assertIntsEqual(t, "backend.redis.db", cfg.Backend.Redis.Db, 0)
	assertIntsEqual(t, "backend.redis.expiration", cfg.Backend.Redis.Expiration, 0)
	assertBoolsEqual(t, "backend.redis.tls.enabled", cfg.Backend.Redis.TLS.Enabled, false)
	assertBoolsEqual(t, "backend.redis.tls.insecure_skip_verify", cfg.Backend.Redis.TLS.InsecureSkipVerify, false)
	assertStringsEqual(t, "compression.type", string(cfg.Compression.Type), "snappy")
	assertStringsEqual(t, "metrics.type", string(cfg.Metrics.Type), "")
	assertStringsEqual(t, "metrics.influx.host", cfg.Metrics.Influx.Host, "")
	assertStringsEqual(t, "metrics.influx.database", cfg.Metrics.Influx.Database, "")
	assertStringsEqual(t, "metrics.influx.username", cfg.Metrics.Influx.Username, "")
	assertStringsEqual(t, "metrics.influx.password", cfg.Metrics.Influx.Password, "")
	assertBoolsEqual(t, "metrics.influx.enabled", cfg.Metrics.Influx.Enabled, false)
	assertIntsEqual(t, "metrics.prometheus.port", cfg.Metrics.Prometheus.Port, 0)
	assertStringsEqual(t, "metrics.prometheus.namespace", cfg.Metrics.Prometheus.Namespace, "")
	assertStringsEqual(t, "metrics.prometheus.subsystem", cfg.Metrics.Prometheus.Subsystem, "")
	assertIntsEqual(t, "metrics.prometheus.timeout_ms", cfg.Metrics.Prometheus.TimeoutMillisRaw, 0)
	assertBoolsEqual(t, "metrics.prometheus.enabled", cfg.Metrics.Prometheus.Enabled, false)
	assertBoolsEqual(t, "rate_limiter.enabled", cfg.RateLimiting.Enabled, true)
	assertInt64sEqual(t, "rate_limiter.num_requests", cfg.RateLimiting.MaxRequestsPerSecond, 100)
	assertIntsEqual(t, "request_limits.max_size_bytes", cfg.RequestLimits.MaxSize, 10*1024)
	assertIntsEqual(t, "request_limits.max_num_values", cfg.RequestLimits.MaxNumValues, 10)
	assertIntsEqual(t, "request_limits.max_ttl_seconds", cfg.RequestLimits.MaxTTLSeconds, 3600)
	assertBoolsEqual(t, "routes.allow_public_write", cfg.Routes.AllowPublicWrite, true)
}

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
	assertStringsEqual(t, "metrics.prometheus.namespace", cfg.Metrics.Prometheus.Namespace, "prebid")
	assertStringsEqual(t, "metrics.prometheus.subsystem", cfg.Metrics.Prometheus.Subsystem, "cache")
	assertIntsEqual(t, "metrics.prometheus.timeout_ms", cfg.Metrics.Prometheus.TimeoutMillisRaw, 100)
	assertBoolsEqual(t, "metrics.prometheus.enabled", cfg.Metrics.Prometheus.Enabled, true)
}

func TestNewConfigFuncFileParam(t *testing.T) {
	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	// Some of the values set in the setConfigDefaults(v *viper.Viper) function
	defaultAdminPort := 2525
	defaultPort := 2424

	type logComponents struct {
		msg string
		lvl logrus.Level
	}
	type testOut struct {
		configFileName       string
		expectedLogInfo      []logComponents
		overridesDefaultPort bool
	}

	testCases := []struct {
		description      string
		inConfigFileName string
		out              testOut
	}{
		{
			description:      "Empty file name: expect INFO level log message and start server with default config values",
			inConfigFileName: "",
			out: testOut{
				expectedLogInfo: []logComponents{
					{
						msg: "No configuration file was specified, Prebid Cache will initialize with default values",
						lvl: logrus.InfoLevel,
					},
				},
				overridesDefaultPort: false,
			},
		},
		{
			description:      "Configuration file was specified but doesn't exist: stop execution and log Fatal message",
			inConfigFileName: "non_existent_file",
			out: testOut{
				expectedLogInfo: []logComponents{
					{
						msg: "Failed to load config file: Config File \"non_existent_file\" Not Found in \"[/etc/prebid-cache /Users/gcarreongutierrez/.prebid-cache /Users/gcarreongutierrez/go/src/github.com/prebid/prebid-cache/config]\"",
						lvl: logrus.FatalLevel,
					},
				},
			},
		},
		{
			description:      "File exists but could not be read because 'txt' extension is not supported: stop execution and log Fatal message",
			inConfigFileName: filepath.Join("configtest", "fake_txt_config_file"),
			out: testOut{
				expectedLogInfo: []logComponents{
					{
						msg: "Failed to load config file: Config File \"configtest/fake_txt_config_file\" Not Found in \"[/etc/prebid-cache /Users/gcarreongutierrez/.prebid-cache /Users/gcarreongutierrez/go/src/github.com/prebid/prebid-cache/config]\"",
						lvl: logrus.FatalLevel,
					},
				},
			},
		},
		{
			description:      "File exists and its 'json' markup is supported: configuration is parsed from file and overrides default port value",
			inConfigFileName: filepath.Join("configtest", "fake_json_config_file"),
			out:              testOut{overridesDefaultPort: true},
		},
		{
			description:      "file exists but its yaml markup is invalid: stop execution and log Fatal message",
			inConfigFileName: filepath.Join("configtest", "config_file_invalid"),
			out: testOut{
				expectedLogInfo: []logComponents{
					{
						msg: "Failed to load config file: While parsing config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `malform...` into map[string]interface {}",
						lvl: logrus.FatalLevel,
					},
				},
			},
		},
		{
			description:      "Valid yaml file exists, configuration from file gets read but does not override port default value",
			inConfigFileName: filepath.Join("configtest", "config_file"),
			out:              testOut{},
		},
		{
			description:      "Valid yaml configuration gets parsed from file and overrides port default value",
			inConfigFileName: filepath.Join("configtest", "config_file_overrides_defaults"),
			out:              testOut{overridesDefaultPort: true},
		},
	}

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		//run test
		cfg := NewConfig(tc.inConfigFileName)

		// Assert logrus expected entries
		if assert.Len(t, hook.Entries, len(tc.out.expectedLogInfo), "Incorrect number of entries were logged to logrus in test: %s. Expected: %d Actual: %d", tc.description, len(tc.out.expectedLogInfo), len(hook.Entries)) {
			for i := 0; i < len(tc.out.expectedLogInfo); i++ {
				assert.Equal(t, tc.out.expectedLogInfo[i].msg, hook.Entries[i].Message, "Wrong log message. Test: %s", tc.description)
				assert.Equal(t, tc.out.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Wrong info level. Test: %s", tc.description)
			}
		} else {
			return
		}

		// Assert configuration
		assert.Equal(t, defaultAdminPort, cfg.AdminPort, "AdminPort number in this test is supposed to be the 2525 default. Test: %s", tc.description)
		if tc.out.overridesDefaultPort {
			assert.NotEqual(t, defaultPort, cfg.Port, "Port number in this test is supposed to be different from the 2424 default. Test: %s", tc.description)
		} else {
			assert.Equal(t, 2424, cfg.Port, "Port number in this test is supposed to be 2424 default. Test: %s", tc.description)
		}

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
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

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	// Define object to run `validateAndLog()` on
	configLogObject := Log{
		Level: Debug,
	}

	// run test
	configLogObject.validateAndLog()

	// Assert logrus entries
	if !assert.Equal(t, 1, len(hook.Entries), "No entries were logged to logrus.") {
		return
	}
	if !assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level) {
		return
	}
	if !assert.Equal(t, "config.log.level: debug", hook.LastEntry().Message) {
		return
	}

	// Reset logrus
	hook.Reset()
	assert.Nil(t, hook.LastEntry())
}

func TestCheckMetricsEnabled(t *testing.T) {

	// Structure to hold the expected log entry values
	type logComponents struct {
		msg string
		lvl logrus.Level
	}
	// Expected log entries when we succeed in activating metrics
	// Prometheus success
	prometheusSuccess := []logComponents{
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
	}

	// Influx success
	influxSuccess := []logComponents{
		{
			msg: "config.metrics.influx.host: http://fakeurl.com",
			lvl: logrus.InfoLevel,
		},
		{
			msg: "config.metrics.influx.database: database-value",
			lvl: logrus.InfoLevel,
		},
	}

	// test cases
	type aTest struct {
		description string
		// In
		influxEnabled     bool
		prometheusEnabled bool
		metricType        MetricsType
		// Out
		expectedError   bool
		expectedLogInfo []logComponents
	}
	testCases := []aTest{
		{
			description:       "[1] metricType = \"none\"; both prometheus and influx flags off. Continue with no metrics, no log to assert",
			influxEnabled:     false,
			prometheusEnabled: false,
			metricType:        "none",
			expectedError:     false,
			expectedLogInfo: []logComponents{
				{
					msg: "Prebid Cache will run without metrics",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description:       "[2] metricType = \"none\"; prometheus flag on",
			influxEnabled:     false,
			prometheusEnabled: true,
			metricType:        "none",
			expectedError:     false,
			expectedLogInfo:   prometheusSuccess,
		},
		{
			description:       "[3] metricType = \"none\"; InfluxDB flag on",
			influxEnabled:     true,
			prometheusEnabled: false,
			metricType:        "none",
			expectedError:     false,
			expectedLogInfo:   influxSuccess,
		},
		{
			description:       "[4] metricType = \"none\"; Both prometheus and influx flags on",
			influxEnabled:     true,
			prometheusEnabled: true,
			metricType:        "none",
			expectedError:     false,
			expectedLogInfo:   append(influxSuccess, prometheusSuccess...),
		},
		{
			description:       "[5] metricType = \"influx\"; both prometheus and influx flags off, expect influx success",
			influxEnabled:     false,
			prometheusEnabled: false,
			metricType:        "influx",
			expectedError:     false,
			expectedLogInfo:   influxSuccess,
		},
		{
			description:       "[6] metricType = \"influx\"; prometheus flags on, expect both metrics",
			influxEnabled:     false,
			prometheusEnabled: true,
			metricType:        "influx",
			expectedError:     false,
			expectedLogInfo:   append(influxSuccess, prometheusSuccess...),
		},
		{
			description:       "[7] metricType = \"influx\"; inlfux flags on",
			influxEnabled:     true,
			prometheusEnabled: false,
			metricType:        "influx",
			expectedError:     false,
			expectedLogInfo:   influxSuccess,
		},
		{
			description:       "[8] metricType = \"influx\"; prometheus and inlfux flags on",
			influxEnabled:     true,
			prometheusEnabled: true,
			metricType:        "influx",
			expectedError:     false,
			expectedLogInfo:   append(influxSuccess, prometheusSuccess...),
		},
		{
			description:       "[9] metricType = \"unknown\"; both prometheus and influx flags off. Exit error",
			influxEnabled:     false,
			prometheusEnabled: false,
			metricType:        "unknown",
			expectedError:     true,
			expectedLogInfo: []logComponents{
				{
					msg: "Metrics \"unknown\" are not supported, exiting program.",
					lvl: logrus.FatalLevel,
				},
			},
		},
		{
			description:       "[10] metricType = \"unknown\"; prometheus flags on.",
			influxEnabled:     false,
			prometheusEnabled: true,
			metricType:        "unknown",
			expectedError:     false,
			expectedLogInfo: append(
				prometheusSuccess,
				logComponents{
					msg: "Prebid Cache will run without unsupported metrics \"unknown\".",
					lvl: logrus.InfoLevel,
				},
			),
		},
		{
			description:       "[11] metricType = \"unknown\"; influx flags on.",
			influxEnabled:     true,
			prometheusEnabled: false,
			metricType:        "unknown",
			expectedError:     false,
			expectedLogInfo: append(
				influxSuccess,
				logComponents{
					msg: "Prebid Cache will run without unsupported metrics \"unknown\".",
					lvl: logrus.InfoLevel,
				},
			),
		},
		{
			description:       "[12] metricType = \"unknown\"; prometheus and inlfux flags on",
			influxEnabled:     true,
			prometheusEnabled: true,
			metricType:        "unknown",
			expectedError:     false,
			expectedLogInfo: append(
				influxSuccess,
				prometheusSuccess[0],
				prometheusSuccess[1],
				prometheusSuccess[2],
				logComponents{
					msg: "Prebid Cache will run without unsupported metrics \"unknown\".",
					lvl: logrus.InfoLevel,
				},
			),
		},
		{
			description:       "[13] metricType = \"\"; both prometheus and influx flags off. Continue with no metrics, no log to assert",
			influxEnabled:     false,
			prometheusEnabled: false,
			metricType:        "",
			expectedError:     false,
			expectedLogInfo: []logComponents{
				{
					msg: "Prebid Cache will run without metrics",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description:       "[14] metricType = \"\"; prometheus flag on",
			influxEnabled:     false,
			prometheusEnabled: true,
			metricType:        "",
			expectedError:     false,
			expectedLogInfo:   prometheusSuccess,
		},
		{
			description:       "[15] metricType = \"\"; InfluxDB flag on",
			influxEnabled:     true,
			prometheusEnabled: false,
			metricType:        "",
			expectedError:     false,
			expectedLogInfo:   influxSuccess,
		},
		{
			description:       "[16] metricType = \"\"; Both prometheus and influx flags on",
			influxEnabled:     true,
			prometheusEnabled: true,
			metricType:        "",
			expectedError:     false,
			expectedLogInfo:   append(influxSuccess, prometheusSuccess...),
		},
	}

	//Standard elements of the config.Metrics object are set so test cases only modify what's relevant to them
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

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for i, test := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		// Set test flags in metrics object
		cfg.Type = test.metricType
		cfg.Influx.Enabled = test.influxEnabled
		cfg.Prometheus.Enabled = test.prometheusEnabled

		//run test
		cfg.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(test.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %d: len(test.expectedLogInfo) = %d len(hook.Entries) = %d", i+1, len(test.expectedLogInfo), len(hook.Entries)) {
			for j := 0; j < len(test.expectedLogInfo); j++ {
				assert.Equal(t, test.expectedLogInfo[j].msg, hook.Entries[j].Message, "Test case %d log message differs", i+1)
				assert.Equal(t, test.expectedLogInfo[j].lvl, hook.Entries[j].Level, "Test case %d log level differs", i+1)
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, test.expectedError, fatal, "Test case %d failed.", i+1)

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestEnabledFlagGetsModified(t *testing.T) {

	type testIn struct {
		metricType        MetricsType
		influxEnabled     bool
		prometheusEnabled bool
	}
	type testOut struct {
		expectedInfluxEnabled     bool
		expectedprometheusEnabled bool
	}

	// test cases
	type aTest struct {
		description string
		in          testIn
		out         testOut
	}
	testCases := []aTest{
		{
			description: "[1] metricType = \"none\"; No flags enabled. ",
			in:          testIn{"none", false, false},
			out:         testOut{false, false},
		},
		{
			description: "[2] metricType = \"none\"; Influx flag enabled.",
			in:          testIn{"none", true, false},
			out:         testOut{true, false},
		},
		{
			description: "[3] metricType = \"none\"; Prometheus flag enabled. ",
			in:          testIn{"none", false, true},
			out:         testOut{false, true},
		},
		{
			description: "[4] metricType = \"none\"; Both flags enabled. ",
			in:          testIn{"none", true, true},
			out:         testOut{true, true},
		},
		{
			description: "[5] metricType = \"influx\"; No flags enabled.",
			in:          testIn{"influx", false, false},
			out:         testOut{true, false},
		},
		{
			description: "[6] metricType = \"influx\"; Influx flag enabled.",
			in:          testIn{"influx", true, false},
			out:         testOut{true, false},
		},
		{
			description: "[7] metricType = \"influx\"; Prometheus flag enabled.",
			in:          testIn{"influx", false, true},
			out:         testOut{true, true},
		},
		{
			description: "[8] metricType = \"influx\"; Both flags enabled.",
			in:          testIn{"influx", true, true},
			out:         testOut{true, true},
		},
		{
			description: "[9] metricType = \"unknown\"; No flags enabled. ",
			in:          testIn{"unknown", false, false},
			out:         testOut{false, false},
		},
		{
			description: "[10] metricType = \"unknown\"; Influx flag enabled.",
			in:          testIn{"unknown", true, false},
			out:         testOut{true, false},
		},
		{
			description: "[11] metricType = \"unknown\"; Prometheus flag enabled. ",
			in:          testIn{"unknown", false, true},
			out:         testOut{false, true},
		},
		{
			description: "[12] metricType = \"unknown\"; Both flags enabled. ",
			in:          testIn{"unknown", true, true},
			out:         testOut{true, true},
		},
		{
			description: "[13] metricType = \"\"; No flags enabled. ",
			in:          testIn{"", false, false},
			out:         testOut{false, false},
		},
		{
			description: "[14] metricType = \"\"; Influx flag enabled.",
			in:          testIn{"", true, false},
			out:         testOut{true, false},
		},
		{
			description: "[15] metricType = \"\"; Prometheus flag enabled. ",
			in:          testIn{"", false, true},
			out:         testOut{false, true},
		},
		{
			description: "[16] metricType = \"\"; Both flags enabled. ",
			in:          testIn{"", true, true},
			out:         testOut{true, true},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for i, test := range testCases {
		// Reset Metrics object
		metricsCfg := Metrics{
			Type: test.in.metricType,
			Influx: InfluxMetrics{
				Host:     "http://fakeurl.com",
				Database: "database-value",
				Enabled:  test.in.influxEnabled,
			},
			Prometheus: PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "cache",
				Enabled:   test.in.prometheusEnabled,
			},
		}

		//run test
		metricsCfg.validateAndLog()

		// Assert `Enabled` flags value
		assert.Equal(t, test.out.expectedInfluxEnabled, metricsCfg.Influx.Enabled, "Test case %d failed. `cfg.Influx.Enabled` carries wrong value.", i+1)
		assert.Equal(t, test.out.expectedprometheusEnabled, metricsCfg.Prometheus.Enabled, "Test case %d failed. `cfg.Prometheus.Enabled` carries wrong value.", i+1)

		//Reset log after every test
		hook.Reset()
	}
}

func TestInfluxValidateAndLog(t *testing.T) {

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
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
			description: "[0] both InfluxDB host and database blank, expect error",
			influxConfig: &InfluxMetrics{
				Host:     "",
				Database: "",
			},
			//out
			expectError: true,
			expectedLogInfo: []logComponents{
				{
					msg: `Despite being enabled, influx metrics came with no host info: config.metrics.influx.host = "".`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: `Despite being enabled, influx metrics came with no database info: config.metrics.influx.database = "".`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: "config.metrics.influx.host: ",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.influx.database: ",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description: "[1] InfluxDB host blank, expect error",
			influxConfig: &InfluxMetrics{
				Host:     "",
				Database: "database-value",
			},
			//out
			expectError: true,
			expectedLogInfo: []logComponents{
				{
					msg: `Despite being enabled, influx metrics came with no host info: config.metrics.influx.host = "".`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: "config.metrics.influx.host: ",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.influx.database: database-value",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description: "[2] InfluxDB database blank, expect error",
			influxConfig: &InfluxMetrics{
				Host:     "http://fakeurl.com",
				Database: "",
			},
			//out
			expectError: true,
			expectedLogInfo: []logComponents{
				{
					msg: `Despite being enabled, influx metrics came with no database info: config.metrics.influx.database = "".`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: "config.metrics.influx.host: http://fakeurl.com",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.influx.database: ",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description: "[3] Valid InfluxDB host and database, expect log.Info",
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

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for j, test := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		//run test
		test.influxConfig.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(test.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %d: len(test.expectedLogInfo) = %d len(hook.Entries) = %d", j, len(test.expectedLogInfo), len(hook.Entries)) {
			for i := 0; i < len(test.expectedLogInfo); i++ {
				assert.Equal(t, test.expectedLogInfo[i].msg, hook.Entries[i].Message, "Test case %d failed", j)
				assert.Equal(t, test.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Test case %d failed", j)
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, test.expectError, fatal)

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestPrometheusValidateAndLog(t *testing.T) {

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
			expectError: true,
			expectedLogInfo: []logComponents{
				{
					msg: `Despite being enabled, prometheus metrics came with an empty port number: config.metrics.prometheus.port = 0`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: "config.metrics.prometheus.namespace: prebid",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.prometheus.subsystem: cache",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.prometheus.port: 0",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description: "[2] Port valid, Namespace invalid, Subsystem valid. Expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "",
				Subsystem: "cache",
			},
			//out
			expectError: true,
			expectedLogInfo: []logComponents{
				{
					msg: `Despite being enabled, prometheus metrics came with an empty name space: config.metrics.prometheus.namespace = .`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: "config.metrics.prometheus.namespace: ",
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
		{
			description: "[3] Port valid, Namespace valid, Subsystem invalid. Expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "",
			},
			//out
			expectError: true,
			expectedLogInfo: []logComponents{
				{
					msg: `Despite being enabled, prometheus metrics came with an empty subsystem value: config.metrics.prometheus.subsystem = \"\".`,
					lvl: logrus.FatalLevel,
				},
				{
					msg: "config.metrics.prometheus.namespace: prebid",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.prometheus.subsystem: ",
					lvl: logrus.InfoLevel,
				},
				{
					msg: "config.metrics.prometheus.port: 8080",
					lvl: logrus.InfoLevel,
				},
			},
		},
		{
			description: "[4] Port valid, Namespace valid, Subsystem valid. Expect elements in log",
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

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for j, test := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		//run test
		test.prometheusConfig.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(test.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %d: len(test.expectedLogInfo) = %d len(hook.Entries) = %d", j, len(test.expectedLogInfo), len(hook.Entries)) {
			for i := 0; i < len(test.expectedLogInfo); i++ {
				assert.Equal(t, test.expectedLogInfo[i].msg, hook.Entries[i].Message)
				assert.Equal(t, test.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Expected Info entry in log")
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, test.expectError, fatal)

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestCompressionValidateAndLog(t *testing.T) {

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
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

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for j, tc := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		//run test
		tc.compConf.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(tc.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %d: len(tc.expectedLogInfo) = %d len(hook.Entries) = %d", j, len(tc.expectedLogInfo), len(hook.Entries)) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message)
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Expected Info entry in log")
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, tc.expectFatal, fatal)

		//Reset log after every test and assert successful reset
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
    namespace: "prebid"
    subsystem: "cache"
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
