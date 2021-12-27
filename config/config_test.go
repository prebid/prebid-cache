package config

import (
	"github.com/PubMatic-OpenWrap/prebid-cache/constant"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
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

func TestPrometheusTimeoutDuration(t *testing.T) {
	prometheusConfig := &PrometheusMetrics{
		TimeoutMillisRaw: 5,
	}

	expectedTimeout := time.Duration(5 * 1000 * 1000)
	actualTimeout := prometheusConfig.Timeout()
	assert.Equal(t, expectedTimeout, actualTimeout)
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
				Hosts: []string{},
			},
			Cassandra: Cassandra{
				DefaultTTL: 2400,
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
				Hosts:      []string{"aerospike2.prebid.com", "aerospike3.prebid.com"},
				Port:       3000,
				Namespace:  "whatever",
				User:       "foo",
				Password:   "bar",
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

func Test_getHostName(t *testing.T) {

	var (
		node string
		pod  string
	)

	saveEnvVarsForServerName := func() {
		node, _ = os.LookupEnv(constant.ENV_VAR_NODE_NAME)
		pod, _ = os.LookupEnv(constant.ENV_VAR_POD_NAME)
	}

	resetEnvVarsForServerName := func() {
		os.Setenv(constant.ENV_VAR_NODE_NAME, node)
		os.Setenv(constant.ENV_VAR_POD_NAME, pod)
	}

	type args struct {
		nodeName string
		podName  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "default_value",
			args: args{},
			want: constant.DEFAULT_NODENAME + ":" + constant.DEFAULT_PODNAME,
		},
		{
			name: "valid_name",
			args: args{
				nodeName: "sfo2hyp084.sfo2.pubmatic.com",
				podName:  "creativecache-0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
			},
			want: "sfo2hyp084:0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
		},
		{
			name: "special_characters",
			args: args{
				nodeName: "sfo2hyp084.sfo2.pubmatic.com!!!@#$-_^%x090",
				podName:  "creativecache-0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
			},
			want: "sfo2hyp084:0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveEnvVarsForServerName()

			if len(tt.args.nodeName) > 0 {
				os.Setenv(constant.ENV_VAR_NODE_NAME, tt.args.nodeName)
			}

			if len(tt.args.podName) > 0 {
				os.Setenv(constant.ENV_VAR_POD_NAME, tt.args.podName)
			}

			if got := getHostName(); got != tt.want {
				t.Errorf("getHostName() = %v, want %v", got, tt.want)
			}
			resetEnvVarsForServerName()
		})
	}
}