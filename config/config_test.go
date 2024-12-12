package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
	testLogrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	v := viper.New()

	setConfigDefaults(v)

	cfg := Configuration{}
	err := v.Unmarshal(&cfg)
	assert.NoError(t, err, "Failed to unmarshal config: %v", err)

	expectedConfig := getExpectedDefaultConfig()
	assert.Equal(t, expectedConfig, cfg, "Expected Configuration instance does not match.")
}

func TestEnvConfig(t *testing.T) {
	defer setEnvVar(t, "PBC_METRICS_INFLUX_HOST", "env-var-defined-metrics-host")()

	// Inside NewConfig() metrics.influx.host sets the default value to ""
	// "config/configtest/sample_full_config.yaml", sets it to  "metrics-host"
	cfg := NewConfig("sample_full_config")

	// assert env variable value supercedes them both
	assert.Equal(t, "env-var-defined-metrics-host", string(cfg.Metrics.Influx.Host), "metrics.influx.host did not equal expected")
}

func TestLogValidateAndLog(t *testing.T) {

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

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
		{
			msg: "config.metrics.influx.measurement: measurement-value",
			lvl: logrus.InfoLevel,
		},
		{
			msg: "config.metrics.influx.align_timestamps: false",
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
			Host:        "http://fakeurl.com",
			Database:    "database-value",
			Measurement: "measurement-value",
		},
		Prometheus: PrometheusMetrics{
			Port:      8080,
			Namespace: "prebid",
			Subsystem: "cache",
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for i, tc := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		// Set test flags in metrics object
		cfg.Type = tc.metricType
		cfg.Influx.Enabled = tc.influxEnabled
		cfg.Prometheus.Enabled = tc.prometheusEnabled

		//run test
		cfg.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(tc.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %d: len(tc.expectedLogInfo) = %d len(hook.Entries) = %d", i+1, len(tc.expectedLogInfo), len(hook.Entries)) {
			for j := 0; j < len(tc.expectedLogInfo); j++ {
				assert.Equal(t, tc.expectedLogInfo[j].msg, hook.Entries[j].Message, "Test case %d log message differs", i+1)
				assert.Equal(t, tc.expectedLogInfo[j].lvl, hook.Entries[j].Level, "Test case %d log level differs", i+1)
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, tc.expectedError, fatal, "Test case %d failed.", i+1)

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
	hook := testLogrus.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for i, tc := range testCases {
		// Reset Metrics object
		metricsCfg := Metrics{
			Type: tc.in.metricType,
			Influx: InfluxMetrics{
				Host:     "http://fakeurl.com",
				Database: "database-value",
				Enabled:  tc.in.influxEnabled,
			},
			Prometheus: PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "cache",
				Enabled:   tc.in.prometheusEnabled,
			},
		}

		//run test
		metricsCfg.validateAndLog()

		// Assert `Enabled` flags value
		assert.Equal(t, tc.out.expectedInfluxEnabled, metricsCfg.Influx.Enabled, "Test case %d failed. `cfg.Influx.Enabled` carries wrong value.", i+1)
		assert.Equal(t, tc.out.expectedprometheusEnabled, metricsCfg.Prometheus.Enabled, "Test case %d failed. `cfg.Prometheus.Enabled` carries wrong value.", i+1)

		//Reset log after every test
		hook.Reset()
	}
}

func TestInfluxValidateAndLog(t *testing.T) {

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

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
			description: "All Required Fields Missing",
			influxConfig: &InfluxMetrics{
				Host:        "",
				Database:    "",
				Measurement: "",
			},
			expectError: true,
			expectedLogInfo: []logComponents{
				{lvl: logrus.FatalLevel, msg: `Despite being enabled, influx metrics came with no host info: config.metrics.influx.host = "".`},
				{lvl: logrus.FatalLevel, msg: `Despite being enabled, influx metrics came with no database info: config.metrics.influx.database = "".`},
				{lvl: logrus.FatalLevel, msg: `Despite being enabled, influx metrics came with no measurement info: config.metrics.influx.measurement = "".`},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.host: "},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.database: "},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.measurement: "},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.align_timestamps: false"},
			},
		},
		{
			description: "Host Missing",
			influxConfig: &InfluxMetrics{
				Host:        "",
				Database:    "database-value",
				Measurement: "measurement-value",
			},
			expectError: true,
			expectedLogInfo: []logComponents{
				{lvl: logrus.FatalLevel, msg: `Despite being enabled, influx metrics came with no host info: config.metrics.influx.host = "".`},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.host: "},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.database: database-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.measurement: measurement-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.align_timestamps: false"},
			},
		},
		{
			description: "Database Missing",
			influxConfig: &InfluxMetrics{
				Host:        "http://fakeurl.com",
				Database:    "",
				Measurement: "measurement-value",
			},
			expectError: true,
			expectedLogInfo: []logComponents{
				{lvl: logrus.FatalLevel, msg: `Despite being enabled, influx metrics came with no database info: config.metrics.influx.database = "".`},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.host: http://fakeurl.com"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.database: "},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.measurement: measurement-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.align_timestamps: false"},
			},
		},
		{
			description: "Measurement Missing",
			influxConfig: &InfluxMetrics{
				Host:        "http://fakeurl.com",
				Database:    "database-value",
				Measurement: "",
			},
			expectError: true,
			expectedLogInfo: []logComponents{
				{lvl: logrus.FatalLevel, msg: `Despite being enabled, influx metrics came with no measurement info: config.metrics.influx.measurement = "".`},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.host: http://fakeurl.com"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.database: database-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.measurement: "},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.align_timestamps: false"},
			},
		},
		{
			description: "All Required Fields Provided",
			influxConfig: &InfluxMetrics{
				Host:            "http://fakeurl.com",
				Database:        "database-value",
				Measurement:     "measurement-value",
				AlignTimestamps: true,
			},
			expectError: false,
			expectedLogInfo: []logComponents{
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.host: http://fakeurl.com"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.database: database-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.measurement: measurement-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.align_timestamps: true"},
			},
		},
		{
			description: "Align Timestamps",
			influxConfig: &InfluxMetrics{
				Host:            "http://fakeurl.com",
				Database:        "database-value",
				Measurement:     "measurement-value",
				AlignTimestamps: true,
			},
			expectError: false,
			expectedLogInfo: []logComponents{
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.host: http://fakeurl.com"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.database: database-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.measurement: measurement-value"},
				{lvl: logrus.InfoLevel, msg: "config.metrics.influx.align_timestamps: true"},
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
		tc.influxConfig.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(tc.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %d: len(tc.expectedLogInfo) = %d len(hook.Entries) = %d", j, len(tc.expectedLogInfo), len(hook.Entries)) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message, "Test case %d failed", j)
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Test case %d failed", j)
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, tc.expectError, fatal)

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
			description: "Port invalid, both Namespace and Subsystem were set. Expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      0,
				Namespace: "prebid",
				Subsystem: "cache",
			},
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
			description: "Port valid, Namespace empty, Subsystem set. Don't expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "",
				Subsystem: "cache",
			},
			expectError: false,
			expectedLogInfo: []logComponents{
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
			description: "Port valid, Namespace set, Subsystem empty. Don't expect error",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "",
			},
			expectError: false,
			expectedLogInfo: []logComponents{
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
			description: "Port valid, both Namespace and Subsystem set. Expect elements in log",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "prebid",
				Subsystem: "cache",
			},
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
		{
			description: "Port valid, Namespace and Subsystem empty. Expect log messages with blank Namespace and Subsystem",
			prometheusConfig: &PrometheusMetrics{
				Port:      8080,
				Namespace: "",
				Subsystem: "",
			},
			expectError: false,
			expectedLogInfo: []logComponents{
				{
					msg: "config.metrics.prometheus.namespace: ",
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
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for _, tc := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		//run test
		tc.prometheusConfig.validateAndLog()

		// Assert logrus expected entries
		if assert.Equal(t, len(tc.expectedLogInfo), len(hook.Entries), "Incorrect number of entries were logged to logrus in test %s.", tc.description) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message)
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, "Expected Info entry in log. Test %s.", tc.description)
			}
		} else {
			return
		}

		// Assert log.Fatalf() was called or not
		assert.Equal(t, tc.expectError, fatal)

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestRequestLimitsValidateAndLog(t *testing.T) {

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		description        string
		inRequestLimitsCfg *RequestLimits
		expectedLogInfo    []logComponents
		expectFatal        bool
	}{
		{
			description:        "Blank RequestLimits",
			inRequestLimitsCfg: &RequestLimits{},
			expectedLogInfo: []logComponents{
				{msg: `config.request_limits.allow_setting_keys: false`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_ttl_seconds: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_size_bytes: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_num_values: 0`, lvl: logrus.InfoLevel},
			},
			expectFatal: false,
		},
		{
			description:        "allow_setting_keys flag set to true",
			inRequestLimitsCfg: &RequestLimits{AllowSettingKeys: true},
			expectedLogInfo: []logComponents{
				{msg: `config.request_limits.allow_setting_keys: true`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_ttl_seconds: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_size_bytes: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_num_values: 0`, lvl: logrus.InfoLevel},
			},
			expectFatal: false,
		},
		{
			description:        "Negative max_ttl_seconds, expect fatal level log and early exit",
			inRequestLimitsCfg: &RequestLimits{MaxTTLSeconds: -1},
			expectedLogInfo: []logComponents{
				{msg: `config.request_limits.allow_setting_keys: false`, lvl: logrus.InfoLevel},
				{msg: `invalid config.request_limits.max_ttl_seconds: -1. Value cannot be negative.`, lvl: logrus.FatalLevel},
			},
			expectFatal: true,
		},
		{
			description:        "Negative max_size_bytes, expect fatal level log and early exit",
			inRequestLimitsCfg: &RequestLimits{MaxSize: -1},
			expectedLogInfo: []logComponents{
				{msg: `config.request_limits.allow_setting_keys: false`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_ttl_seconds: 0`, lvl: logrus.InfoLevel},
				{msg: `invalid config.request_limits.max_size_bytes: -1. Value cannot be negative.`, lvl: logrus.FatalLevel},
			},
			expectFatal: true,
		},
		{
			description:        "Negative max_num_values, expect fatal level log and early exit",
			inRequestLimitsCfg: &RequestLimits{MaxNumValues: -1},
			expectedLogInfo: []logComponents{
				{msg: `config.request_limits.allow_setting_keys: false`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_ttl_seconds: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_size_bytes: 0`, lvl: logrus.InfoLevel},
				{msg: `invalid config.request_limits.max_num_values: -1. Value cannot be negative.`, lvl: logrus.FatalLevel},
			},
			expectFatal: true,
		},
		{
			description:        "Negative max_header_size_bytes, expect fatal level log and early exit",
			inRequestLimitsCfg: &RequestLimits{MaxHeaderSize: -1},
			expectedLogInfo: []logComponents{
				{msg: `config.request_limits.allow_setting_keys: false`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_ttl_seconds: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_size_bytes: 0`, lvl: logrus.InfoLevel},
				{msg: `config.request_limits.max_num_values: 0`, lvl: logrus.InfoLevel},
				{msg: `invalid config.request_limits.max_header_size_bytes: -1. Value cannot be negative.`, lvl: logrus.FatalLevel},
			},
			expectFatal: true,
		},
	}

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for _, tc := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		// Run test
		tc.inRequestLimitsCfg.validateAndLog()

		// Assert logrus expected entries
		logEntryCount := 0
		for i := 0; i < len(tc.expectedLogInfo); i++ {
			assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message, tc.description+":message")
			assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, tc.description+":log level")

			logEntryCount++
			if tc.expectedLogInfo[i].lvl == logrus.FatalLevel {
				break
			}
		}
		if tc.expectedLogInfo[logEntryCount-1].lvl == logrus.FatalLevel && !fatal {
			t.Errorf("Log level fatal was expected. %s", tc.description)
		}
		assert.Len(t, tc.expectedLogInfo, logEntryCount, tc.description)

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestRequestLogging(t *testing.T) {
	hook := testLogrus.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		name                string
		inRequestLoggingCfg *RequestLogging
		expectedLogInfo     []logComponents
	}{
		{
			name: "invalid_negative", // must be greater or equal to zero. Expect fatal log
			inRequestLoggingCfg: &RequestLogging{
				RefererSamplingRate: -0.1,
			},
			expectedLogInfo: []logComponents{
				{msg: `invalid config.request_logging.referer_sampling_rate: value must be positive and not greater than 1.0. Got -0.1`, lvl: logrus.FatalLevel},
			},
		},
		{
			name: "invalid_high", // must be less than or equal to 1. expect fatal log.
			inRequestLoggingCfg: &RequestLogging{
				RefererSamplingRate: 1.1,
			},
			expectedLogInfo: []logComponents{
				{msg: `invalid config.request_logging.referer_sampling_rate: value must be positive and not greater than 1.0. Got 1.1`, lvl: logrus.FatalLevel},
			},
		},
		{
			name: "valid_one", // sampling rate of 1.0 is between the acceptable threshold. Expect info log"
			inRequestLoggingCfg: &RequestLogging{
				RefererSamplingRate: 1.0,
			},
			expectedLogInfo: []logComponents{
				{msg: `config.request_logging.referer_sampling_rate: 1`, lvl: logrus.InfoLevel},
			},
		},
		{
			name: "valid_zero", // sampling rate of 0.0 is between the acceptable threshold. Expect info log.
			inRequestLoggingCfg: &RequestLogging{
				RefererSamplingRate: 0.0,
			},
			expectedLogInfo: []logComponents{
				{msg: `config.request_logging.referer_sampling_rate: 0`, lvl: logrus.InfoLevel},
			},
		},
		{
			name: "valid",
			inRequestLoggingCfg: &RequestLogging{
				RefererSamplingRate: 0.1111,
			},
			expectedLogInfo: []logComponents{
				{msg: `config.request_logging.referer_sampling_rate: 0.1111`, lvl: logrus.InfoLevel},
			},
		},
	}

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.inRequestLoggingCfg.validateAndLog()

			// assertions
			require.Len(t, hook.Entries, len(tc.expectedLogInfo), tc.name+":log_entries")
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message, tc.name+":message")
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, tc.name+":log_level")
			}

			//Reset log after every test and assert successful reset
			hook.Reset()
			assert.Nil(t, hook.LastEntry())

		})
	}
}

func TestCompressionValidateAndLog(t *testing.T) {

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		description      string
		inCompressionCfg *Compression
		inBackendType    BackendType
		expectedLogInfo  []logComponents
	}{
		{
			description:      "Blank compression type, expect fatal level log entry",
			inCompressionCfg: &Compression{Type: CompressionType("")},
			inBackendType:    BackendMemory,
			expectedLogInfo: []logComponents{
				{msg: `invalid config.compression.type: . It must be "none" or "snappy"`, lvl: logrus.FatalLevel},
			},
		},
		{
			description:      "Valid compression type 'none', expect info level log entry",
			inCompressionCfg: &Compression{Type: CompressionNone},
			inBackendType:    BackendMemory,
			expectedLogInfo: []logComponents{
				{msg: "config.compression.type: none", lvl: logrus.InfoLevel},
			},
		},
		{
			description:      "Valid compression type 'snappy', expect info level log entry",
			inCompressionCfg: &Compression{Type: CompressionSnappy},
			inBackendType:    BackendMemory,
			expectedLogInfo: []logComponents{
				{msg: "config.compression.type: snappy", lvl: logrus.InfoLevel},
			},
		},
		{
			description:      "Unsupported compression, expect fatal level log entry",
			inCompressionCfg: &Compression{Type: CompressionType("UnknownCompressionType")},
			inBackendType:    BackendMemory,
			expectedLogInfo: []logComponents{
				{msg: `invalid config.compression.type: UnknownCompressionType. It must be "none" or "snappy"`, lvl: logrus.FatalLevel},
			},
		},
	}

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		// Run test
		tc.inCompressionCfg.validateAndLog()

		// Assert logrus expected entries
		if assert.Len(t, hook.Entries, len(tc.expectedLogInfo), tc.description) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message, tc.description+":message")
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, tc.description+":log level")
			}
		}

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestNewConfigFromFile(t *testing.T) {
	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}
	testCases := []struct {
		description      string
		inConfigFileName string
		expectedLogInfo  []logComponents
		expectedConfig   Configuration
	}{
		{
			description:      "Empty file name: expect INFO level log message and start server with default config values",
			inConfigFileName: "",
			expectedLogInfo: []logComponents{
				{
					msg: "Configuration file not detected. Initializing with default values and environment variable overrides.",
					lvl: logrus.InfoLevel,
				},
			},
			expectedConfig: getExpectedDefaultConfig(),
		},
		{
			description:      "Configuration file was specified but doesn't exist: expect INFO level log message and start server with default config values",
			inConfigFileName: "non_existent_file",
			expectedLogInfo: []logComponents{
				{
					msg: "Configuration file not detected. Initializing with default values and environment variable overrides.",
					lvl: logrus.InfoLevel,
				},
			},
			expectedConfig: getExpectedDefaultConfig(),
		},
		{
			description:      "file exists but its yaml markup is invalid: stop execution and log Fatal message",
			inConfigFileName: filepath.Join("configtest", "config_invalid"),
			expectedLogInfo: []logComponents{
				{
					msg: "Configuration file could not be read:",
					lvl: logrus.FatalLevel,
				},
			},
			expectedConfig: getExpectedDefaultConfig(),
		},
		{
			description:      "Valid yaml configuration populates all configuration fields properly",
			inConfigFileName: filepath.Join("configtest", "sample_full_config"),
			expectedConfig:   getExpectedFullConfigForTestFile(),
		},
	}

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		//run test
		actualCfg := NewConfig(tc.inConfigFileName)

		// Assert logrus expected entries
		if assert.Len(t, hook.Entries, len(tc.expectedLogInfo), tc.description) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.True(t, strings.HasPrefix(hook.Entries[i].Message, tc.expectedLogInfo[i].msg), tc.description+":message")
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, tc.description+":log level")
			}
		}

		assert.Equal(t, tc.expectedConfig, actualCfg, "Expected Configuration instance does not match. Test desc:%s", tc.description)

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

func TestConfigurationValidateAndLog(t *testing.T) {
	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()
	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	// Instantiate test objects
	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	expectedConfig := getExpectedDefaultConfig()

	expectedLogInfo := []logComponents{
		{msg: "config.port: 2424", lvl: logrus.InfoLevel},
		{msg: "config.admin_port: 2525", lvl: logrus.InfoLevel},
		{msg: "config.log.level: info", lvl: logrus.InfoLevel},
		{msg: "config.rate_limiter.enabled: true", lvl: logrus.InfoLevel},
		{msg: "config.rate_limiter.num_requests: 100", lvl: logrus.InfoLevel},
		{msg: "config.request_limits.allow_setting_keys: false", lvl: logrus.InfoLevel},
		{msg: "config.request_limits.max_ttl_seconds: 3600", lvl: logrus.InfoLevel},
		{msg: "config.request_limits.max_size_bytes: 10240", lvl: logrus.InfoLevel},
		{msg: "config.request_limits.max_num_values: 10", lvl: logrus.InfoLevel},
		{msg: "config.request_limits.max_header_size_bytes: 1048576", lvl: logrus.InfoLevel},
		{msg: "config.request_logging.referer_sampling_rate: 0", lvl: logrus.InfoLevel},
		{msg: "config.backend.type: memory", lvl: logrus.InfoLevel},
		{msg: "config.compression.type: snappy", lvl: logrus.InfoLevel},
		{msg: "Prebid Cache will run without metrics", lvl: logrus.InfoLevel},
	}

	// Run test
	expectedConfig.ValidateAndLog()

	// Assertions
	if assert.Len(t, hook.Entries, len(expectedLogInfo)) {
		for i := 0; i < len(expectedLogInfo); i++ {
			assert.Equal(t, expectedLogInfo[i].msg, hook.Entries[i].Message, "Wrong message")
			assert.Equal(t, expectedLogInfo[i].lvl, hook.Entries[i].Level, "Wrong log level")
		}
	}

	//Reset log
	hook.Reset()
	assert.Nil(t, hook.LastEntry())
}

func TestPrometheusTimeoutDuration(t *testing.T) {
	prometheusConfig := &PrometheusMetrics{
		TimeoutMillisRaw: 5,
	}

	expectedTimeout := time.Duration(5 * 1000 * 1000)
	actualTimeout := prometheusConfig.Timeout()
	assert.Equal(t, expectedTimeout, actualTimeout)
}

func TestRoutesValidateAndLog(t *testing.T) {
	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	type logComponents struct {
		msg string
		lvl logrus.Level
	}

	testCases := []struct {
		description     string
		inRoutesConfig  *Routes
		expectedLogInfo []logComponents
	}{
		{
			description:    "Public write is not allowed, log info level message",
			inRoutesConfig: &Routes{AllowPublicWrite: false},
			expectedLogInfo: []logComponents{
				{msg: "Main server will only accept GET requests", lvl: logrus.InfoLevel},
			},
		},
		{
			description:     "Public write allowed. Default GET and POST methods are allowed, no need to log anything",
			inRoutesConfig:  &Routes{AllowPublicWrite: true},
			expectedLogInfo: []logComponents{},
		},
	}

	//substitute logger exit function so execution doesn't get interrupted
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		// Run test
		tc.inRoutesConfig.validateAndLog()

		// Assert logrus expected entries
		if assert.Len(t, hook.Entries, len(tc.expectedLogInfo), tc.description) {
			for i := 0; i < len(tc.expectedLogInfo); i++ {
				assert.Equal(t, tc.expectedLogInfo[i].msg, hook.Entries[i].Message, tc.description+":message")
				assert.Equal(t, tc.expectedLogInfo[i].lvl, hook.Entries[i].Level, tc.description+":log level")
			}
		}

		//Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
	}
}

// setEnvVar sets an environment variable to a certain value, and returns a function which resets it to its original value.
func setEnvVar(t *testing.T, key string, val string) func() {
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

func getExpectedDefaultConfig() Configuration {
	return Configuration{
		Port:          2424,
		AdminPort:     2525,
		IndexResponse: "This application stores short-term data for use in Prebid.",
		Log: Log{
			Level: Info,
		},
		Backend: Backend{
			Type: BackendMemory,
			Memcache: Memcache{
				Hosts: []string{},
			},
			Aerospike: Aerospike{
				Hosts:          []string{},
				MaxReadRetries: 2,
			},
			Cassandra: Cassandra{
				DefaultTTL: utils.CASSANDRA_DEFAULT_TTL_SECONDS,
			},
			Redis: Redis{
				ExpirationMinutes: utils.REDIS_DEFAULT_EXPIRATION_MINUTES,
			},
			Ignite: Ignite{
				Headers: map[string]string{},
			},
		},
		Compression: Compression{
			Type: CompressionType("snappy"),
		},
		RateLimiting: RateLimiting{
			Enabled:              true,
			MaxRequestsPerSecond: 100,
		},
		RequestLogging: RequestLogging{
			RefererSamplingRate: 0.00,
		},
		RequestLimits: RequestLimits{
			MaxSize:       10240,
			MaxNumValues:  10,
			MaxTTLSeconds: 3600,
			MaxHeaderSize: 1048576,
		},
		Routes: Routes{
			AllowPublicWrite: true,
		},
	}
}

// Returns a Configuration object that matches the values found in the `sample_full_config.yaml`
func getExpectedFullConfigForTestFile() Configuration {
	return Configuration{
		Port:          9000,
		AdminPort:     2525,
		IndexResponse: "Any index response",
		Log: Log{
			Level: Info,
		},
		RateLimiting: RateLimiting{
			Enabled:              false,
			MaxRequestsPerSecond: 150,
		},
		RequestLimits: RequestLimits{
			MaxSize:          10240,
			MaxNumValues:     10,
			MaxTTLSeconds:    5000,
			AllowSettingKeys: true,
			MaxHeaderSize:    16384, //16KiB
		},
		Backend: Backend{
			Type: BackendMemory,
			Aerospike: Aerospike{
				DefaultTTLSecs:      3600,
				Host:                "aerospike.prebid.com",
				Hosts:               []string{"aerospike2.prebid.com", "aerospike3.prebid.com"},
				Port:                3000,
				Namespace:           "whatever",
				User:                "foo",
				Password:            "bar",
				MaxReadRetries:      2,
				ConnIdleTimeoutSecs: 2,
			},
			Cassandra: Cassandra{
				Hosts:      "127.0.0.1",
				Keyspace:   "prebid",
				DefaultTTL: 60,
			},
			Memcache: Memcache{
				Hosts: []string{"10.0.0.1:11211", "127.0.0.1"},
			},
			Redis: Redis{
				Host:              "127.0.0.1",
				Port:              6379,
				Password:          "redis-password",
				Db:                1,
				ExpirationMinutes: 1,
				TLS: RedisTLS{
					Enabled:            false,
					InsecureSkipVerify: false,
				},
			},
			Ignite: Ignite{
				Scheme: "http",
				Host:   "127.0.0.1",
				Port:   8080,
				Headers: map[string]string{
					"Content-Length": "0",
				},
				Cache: IgniteCache{
					Name:          "whatever",
					CreateOnStart: false,
				},
			},
		},
		Compression: Compression{
			Type: CompressionType("snappy"),
		},
		Metrics: Metrics{
			Type: MetricsType("none"),
			Influx: InfluxMetrics{
				Host:     "metrics-host",
				Database: "metrics-database",
				Username: "metrics-username",
				Password: "metrics-password",
				Enabled:  true,
			},
			Prometheus: PrometheusMetrics{
				Port:             8080,
				Namespace:        "prebid",
				Subsystem:        "cache",
				TimeoutMillisRaw: 100,
				Enabled:          true,
			},
		},
		Routes: Routes{
			AllowPublicWrite: true,
		},
	}
}
