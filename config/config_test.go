package config

import (
	"os"
	"strings"
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-cache/constant"

	"github.com/spf13/viper"
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
	assertStringsEqual(t, "backend.type", string(cfg.Backend.Type), "memory")
	assertStringsEqual(t, "backend.aerospike.host", cfg.Backend.Aerospike.Host, "aerospike.prebid.com")
	assertIntsEqual(t, "backend.aerospike.port", cfg.Backend.Aerospike.Port, 3000)
	assertStringsEqual(t, "backend.aerospike.namespace", cfg.Backend.Aerospike.Namespace, "whatever")
	assertStringsEqual(t, "backend.azure.account", cfg.Backend.Azure.Account, "azure-account-here")
	assertStringsEqual(t, "backend.azure.key", cfg.Backend.Azure.Key, "azure-key-here")
	assertStringsEqual(t, "backend.cassandra.hosts", cfg.Backend.Cassandra.Hosts, "127.0.0.1")
	assertStringsEqual(t, "backend.cassandra.keyspace", cfg.Backend.Cassandra.Keyspace, "prebid")
	assertStringsEqual(t, "backend.memcache.hosts", cfg.Backend.Memcache.Hosts[0], "10.0.0.1:11211")
	assertStringsEqual(t, "compression.type", string(cfg.Compression.Type), "snappy")
	assertStringsEqual(t, "metrics.type", string(cfg.Metrics.Type), "none")
	assertStringsEqual(t, "metrics.influx.host", cfg.Metrics.Influx.Host, "default-metrics-host")
	assertStringsEqual(t, "metrics.influx.database", cfg.Metrics.Influx.Database, "default-metrics-database")
	assertStringsEqual(t, "metrics.influx.username", cfg.Metrics.Influx.Username, "metrics-username")
	assertStringsEqual(t, "metrics.influx.password", cfg.Metrics.Influx.Password, "metrics-password")
	assertStringsEqual(t, "stats.host", cfg.Stats.StatsHost, "stats-host")
	assertStringsEqual(t, "stats.port", cfg.Stats.StatsPort, "stats-port")
	assertStringsEqual(t, "stats.dc_name", cfg.Stats.StatsDCName, "stats-dc-name")
	assertStringsEqual(t, "server.port", cfg.Server.ServerPort, "server-port")
	assertStringsEqual(t, "server.name", cfg.Server.ServerName, "server-name")
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
backend:
  type: "memory"
  aerospike:
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
compression:
  type: "snappy"
metrics:
  type: "none"
  influx:
    host: "default-metrics-host"
    database: "default-metrics-database"
    username: "metrics-username"
    password: "metrics-password"
stats:
  host: "stats-host"
  port: "stats-port"
  dc_name: "stats-dc-name"
server:
  port: "server-port"
  name: "server-name"
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
