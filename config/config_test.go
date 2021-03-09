package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	hook := test.NewGlobal()

	// Define object to run `validateAndLog()` on
	configLogObject := Log{
		Level: Debug,
		UUID:  false,
	}

	// Run test
	configLogObject.validateAndLog()

	if assert.Len(t, hook.Entries, 2, "Logged incorrect number of entries") {
		for i := 0; i < len(hook.Entries); i++ {
			assert.Equal(t, logrus.InfoLevel, hook.Entries[i].Level, "Expected info level log")
		}
		assert.Equal(t, "config.log.level: debug", hook.Entries[0].Message, "Wrong log message")
		assert.Equal(t, "config.log.uuid: false", hook.Entries[1].Message, "Wrong log message")
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
		compressionCfg  *Compression
		expectedLogInfo []logComponents
	}{
		{
			description:    "Blank compression type, expect fatal level log entry",
			compressionCfg: &Compression{Type: CompressionType("")},
			expectedLogInfo: []logComponents{
				{msg: `invalid config.compression.type: . It must be "none" or "snappy"`, lvl: logrus.FatalLevel},
			},
		},
		{
			description:    "Valid compression type 'none', expect info level log entry",
			compressionCfg: &Compression{Type: CompressionNone},
			expectedLogInfo: []logComponents{
				{msg: "config.compression.type: none", lvl: logrus.InfoLevel},
			},
		},
		{
			description:    "Valid compression type 'snappy', expect info level log entry",
			compressionCfg: &Compression{Type: CompressionSnappy},
			expectedLogInfo: []logComponents{
				{msg: "config.compression.type: snappy", lvl: logrus.InfoLevel},
			},
		},
		{
			description:    "Unsupported compression, expect fatal level log entry",
			compressionCfg: &Compression{Type: CompressionType("UnknownCompressionType")},
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
		tc.compressionCfg.validateAndLog()

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
	hook := test.NewGlobal()

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
	hook := test.NewGlobal()
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
		{msg: fmt.Sprintf("config.port: %d", expectedConfig.Port), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.admin_port: %d", expectedConfig.AdminPort), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.log.level: %s", expectedConfig.Log.Level), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.log.uuid: %t", expectedConfig.Log.UUID), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.rate_limiter.enabled: %t", expectedConfig.RateLimiting.Enabled), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.rate_limiter.num_requests: %d", expectedConfig.RateLimiting.MaxRequestsPerSecond), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.request_limits.allow_setting_keys: %v", expectedConfig.RequestLimits.AllowSettingKeys), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.request_limits.max_ttl_seconds: %d", expectedConfig.RequestLimits.MaxTTLSeconds), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.request_limits.max_size_bytes: %d", expectedConfig.RequestLimits.MaxSize), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.request_limits.max_num_values: %d", expectedConfig.RequestLimits.MaxNumValues), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.backend.type: %s", expectedConfig.Backend.Type), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("config.compression.type: %s", expectedConfig.Compression.Type), lvl: logrus.InfoLevel},
		{msg: fmt.Sprintf("Prebid Cache will run without metrics"), lvl: logrus.InfoLevel},
	}

	// Run test
	expectedConfig.ValidateAndLog()

	// Assertions
	if assert.Len(t, hook.Entries, len(expectedLogInfo)) {
		for i := 0; i < len(expectedLogInfo); i++ {
			assert.True(t, strings.HasPrefix(hook.Entries[i].Message, expectedLogInfo[i].msg), "Wrong message")
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
	hook := test.NewGlobal()

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
		},
		Compression: Compression{
			Type: CompressionType("snappy"),
		},
		RateLimiting: RateLimiting{
			Enabled:              true,
			MaxRequestsPerSecond: 100,
		},
		RequestLimits: RequestLimits{
			MaxSize:       10240,
			MaxNumValues:  10,
			MaxTTLSeconds: 3600,
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
		},
		Backend: Backend{
			Type: BackendMemory,
			Aerospike: Aerospike{
				DefaultTTL: 3600,
				Host:       "aerospike.prebid.com",
				Port:       3000,
				Namespace:  "whatever",
			},
			Azure: Azure{
				Account: "azure-account-here",
				Key:     "azure-key-here",
			},
			Cassandra: Cassandra{
				Hosts:    "127.0.0.1",
				Keyspace: "prebid",
			},
			Memcache: Memcache{
				Hosts: []string{"10.0.0.1:11211", "127.0.0.1"},
			},
			Redis: Redis{
				Host:       "127.0.0.1",
				Port:       6379,
				Password:   "redis-password",
				Db:         1,
				Expiration: 1,
				TLS: RedisTLS{
					Enabled:            false,
					InsecureSkipVerify: false,
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
