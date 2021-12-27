package config

import (
	"os"
	"strings"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/constant"
	"github.com/PubMatic-OpenWrap/prebid-cache/stats"
	"github.com/spf13/viper"
)

func NewConfig(filename string) Configuration {
	v := viper.New()
	setConfigDefaults(v)
	setEnvVarsLookup(v)
	setConfigFilePath(v, filename)

	// Read configuration file
	err := v.ReadInConfig()
	if err != nil {
		// Make sure the configuration file was not defective
		if _, fileNotFound := err.(viper.ConfigFileNotFoundError); fileNotFound {
			// Config file not found. Just log at info level and start Prebid Cache with default values
			logger.Info("Configuration file not detected. Initializing with default values and environment variable overrides.")
		} else {
			// Config file was found but was defective, Either `UnsupportedConfigError` or `ConfigParseError` was thrown
			logger.Fatal("Configuration file could not be read: %v", err)
		}
	}

	cfg := Configuration{}
	if err := v.Unmarshal(&cfg); err != nil {
		logger.Fatal("Failed to unmarshal config: %v", err)
	}

	cfg.Server.ServerName = getHostName()

	//Initialize Stats Server
	stats.InitStat(cfg.Stats.StatsHost, cfg.Stats.StatsPort, cfg.Server.ServerName, cfg.Stats.StatsDCName,
		cfg.Stats.PortTCP, cfg.Stats.PublishInterval, cfg.Stats.PublishThreshold, cfg.Stats.Retries, cfg.Stats.DialTimeout, cfg.Stats.KeepAliveDuration, cfg.Stats.MaxIdleConnections, cfg.Stats.MaxIdleConnectionsPerHost, cfg.Stats.UseTCP)

	var logConf logger.LogConf
	logConf.LogLevel = cfg.OWLog.LogLevel
	logConf.LogPath = cfg.OWLog.LogPath
	logConf.LogRotationTime = cfg.OWLog.LogRotationTime
	logConf.MaxLogFiles = cfg.OWLog.MaxLogFiles
	logConf.MaxLogSize = cfg.OWLog.MaxLogSize
	//Initialize logger
	logger.InitGlog(logConf)

	return cfg
}

func setConfigDefaults(v *viper.Viper) {
	v.SetDefault("port", 2424)
	v.SetDefault("admin_port", 2525)
	v.SetDefault("index_response", "This application stores short-term data for use in Prebid.")
	v.SetDefault("log.level", "info")
	v.SetDefault("backend.type", "memory")
	v.SetDefault("backend.aerospike.host", "")
	v.SetDefault("backend.aerospike.hosts", []string{})
	v.SetDefault("backend.aerospike.port", 0)
	v.SetDefault("backend.aerospike.namespace", "")
	v.SetDefault("backend.aerospike.user", "")
	v.SetDefault("backend.aerospike.password", "")
	v.SetDefault("backend.aerospike.default_ttl_seconds", 0)
	v.SetDefault("backend.cassandra.hosts", "")
	v.SetDefault("backend.cassandra.keyspace", "")
	v.SetDefault("backend.cassandra.default_ttl_seconds", 2400)
	v.SetDefault("backend.memcache.hosts", []string{})
	v.SetDefault("backend.redis.host", "")
	v.SetDefault("backend.redis.port", 0)
	v.SetDefault("backend.redis.password", "")
	v.SetDefault("backend.redis.db", 0)
	v.SetDefault("backend.redis.expiration", 0)
	v.SetDefault("backend.redis.tls.enabled", false)
	v.SetDefault("backend.redis.tls.insecure_skip_verify", false)
	v.SetDefault("compression.type", "snappy")
	v.SetDefault("metrics.influx.host", "")
	v.SetDefault("metrics.influx.database", "")
	v.SetDefault("metrics.influx.username", "")
	v.SetDefault("metrics.influx.password", "")
	v.SetDefault("metrics.influx.enabled", false)
	v.SetDefault("metrics.prometheus.port", 0)
	v.SetDefault("metrics.prometheus.namespace", "")
	v.SetDefault("metrics.prometheus.subsystem", "")
	v.SetDefault("metrics.prometheus.timeout_ms", 0)
	v.SetDefault("metrics.prometheus.enabled", false)
	v.SetDefault("rate_limiter.enabled", true)
	v.SetDefault("rate_limiter.num_requests", 100)
	v.SetDefault("request_limits.allow_setting_keys", false)
	v.SetDefault("request_limits.max_size_bytes", 10*1024)
	v.SetDefault("request_limits.max_num_values", 10)
	v.SetDefault("request_limits.max_ttl_seconds", 3600)
	v.SetDefault("routes.allow_public_write", true)
}

func setConfigFilePath(v *viper.Viper, filename string) {
	v.SetConfigName(filename)              // name of config file (without extension)
	v.AddConfigPath("/etc/prebid-cache/")  // path to look for the config file in
	v.AddConfigPath("$HOME/.prebid-cache") // call multiple times to add many search paths
	v.AddConfigPath(".")
}

func setEnvVarsLookup(v *viper.Viper) {
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("PBC")
	v.AutomaticEnv()
}

type Configuration struct {
	Port          int           `mapstructure:"port"`
	AdminPort     int           `mapstructure:"admin_port"`
	IndexResponse string        `mapstructure:"index_response"`
	Log           Log           `mapstructure:"log"`
	RateLimiting  RateLimiting  `mapstructure:"rate_limiter"`
	RequestLimits RequestLimits `mapstructure:"request_limits"`
	Backend       Backend       `mapstructure:"backend"`
	Compression   Compression   `mapstructure:"compression"`
	Metrics       Metrics       `mapstructure:"metrics"`
	Stats         Stats         `mapstructure:"stats"`
	Server        Server        `mapstructure:"server"`
	OWLog         OWLog         `mapstructure:"ow_log"`
	Routes        Routes        `mapstructure:"routes"`
}

// ValidateAndLog validates the config, terminating the program on any errors.
// It also logs the config values that it used.
func (cfg *Configuration) ValidateAndLog() {
	logger.Info("config.port: %d", cfg.Port)
	logger.Info("config.admin_port: %d", cfg.AdminPort)
	cfg.Log.validateAndLog()
	cfg.RateLimiting.validateAndLog()
	cfg.RequestLimits.validateAndLog()

	if err := cfg.Backend.validateAndLog(); err != nil {
		logger.Fatal("%s", err.Error())
	}

	cfg.Compression.validateAndLog()
	cfg.Metrics.validateAndLog()
	cfg.Stats.validateAndLog()
	cfg.Server.validateAndLog()
	cfg.OWLog.validateAndLog()
	cfg.Routes.validateAndLog()
}

type Log struct {
	Level LogLevel `mapstructure:"level"`
}

func (cfg *Log) validateAndLog() {
	logger.Info("config.log.level: %s", cfg.Level)
}

type LogLevel string

const (
	Trace   LogLevel = "trace"
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
	Fatal   LogLevel = "fatal"
	Panic   LogLevel = "panic"
)

type RateLimiting struct {
	Enabled              bool  `mapstructure:"enabled"`
	MaxRequestsPerSecond int64 `mapstructure:"num_requests"`
}

func (cfg *RateLimiting) validateAndLog() {
	logger.Info("config.rate_limiter.enabled: %t", cfg.Enabled)
	logger.Info("config.rate_limiter.num_requests: %d", cfg.MaxRequestsPerSecond)
}

type RequestLimits struct {
	MaxSize          int  `mapstructure:"max_size_bytes"`
	MaxNumValues     int  `mapstructure:"max_num_values"`
	MaxTTLSeconds    int  `mapstructure:"max_ttl_seconds"`
	AllowSettingKeys bool `mapstructure:"allow_setting_keys"`
}

func (cfg *RequestLimits) validateAndLog() {
	logger.Info("config.request_limits.allow_setting_keys: %v", cfg.AllowSettingKeys)

	if cfg.MaxTTLSeconds >= 0 {
		logger.Info("config.request_limits.max_ttl_seconds: %d", cfg.MaxTTLSeconds)
	} else {
		logger.Fatal("invalid config.request_limits.max_ttl_seconds: %d. Value cannot be negative.", cfg.MaxTTLSeconds)
	}

	if cfg.MaxSize >= 0 {
		logger.Info("config.request_limits.max_size_bytes: %d", cfg.MaxSize)
	} else {
		logger.Fatal("invalid config.request_limits.max_size_bytes: %d. Value cannot be negative.", cfg.MaxSize)
	}

	if cfg.MaxNumValues >= 0 {
		logger.Info("config.request_limits.max_num_values: %d", cfg.MaxNumValues)
	} else {
		logger.Fatal("invalid config.request_limits.max_num_values: %d. Value cannot be negative.", cfg.MaxNumValues)
	}
}

type Compression struct {
	Type CompressionType `mapstructure:"type"`
}

func (cfg *Compression) validateAndLog() {
	switch cfg.Type {
	case CompressionNone:
		fallthrough
	case CompressionSnappy:
		logger.Info("config.compression.type: %s", cfg.Type)
	default:
		logger.Fatal(`invalid config.compression.type: %s. It must be "none" or "snappy"`, cfg.Type)
	}
}

type CompressionType string

const (
	CompressionNone   CompressionType = "none"
	CompressionSnappy CompressionType = "snappy"
)

type Metrics struct {
	Type       MetricsType       `mapstructure:"type"`
	Influx     InfluxMetrics     `mapstructure:"influx"`
	Prometheus PrometheusMetrics `mapstructure:"prometheus"`
}

func (cfg *Metrics) validateAndLog() {

	if cfg.Type == MetricsInflux || cfg.Influx.Enabled {
		cfg.Influx.validateAndLog()
		cfg.Influx.Enabled = true
	}

	if cfg.Prometheus.Enabled {
		cfg.Prometheus.validateAndLog()
		cfg.Prometheus.Enabled = true
	}

	metricsEnabled := cfg.Influx.Enabled || cfg.Prometheus.Enabled
	if cfg.Type == MetricsNone || cfg.Type == "" {
		if !metricsEnabled {
			logger.Info("Prebid Cache will run without metrics")
		}
	} else if cfg.Type != MetricsInflux {
		// Was any other metrics system besides "InfluxDB" or "Prometheus" specified in `cfg.Type`?
		if metricsEnabled {
			// Prometheus, Influx or both, are enabled. Log a message explaining that `prebid-cache` will
			// continue with supported metrics and non-supported metrics will be disabled
			logger.Info("Prebid Cache will run without unsupported metrics \"%s\".", cfg.Type)
		} else {
			// The only metrics engine specified in the configuration file is a non-supported
			// metrics engine. We should log error and exit program
			logger.Fatal("Metrics \"%s\" are not supported, exiting program.", cfg.Type)
		}
	}
}

type MetricsType string

const (
	MetricsNone   MetricsType = "none"
	MetricsInflux MetricsType = "influx"
)

type InfluxMetrics struct {
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Enabled  bool   `mapstructure:"enabled"`
}

func (influxMetricsConfig *InfluxMetrics) validateAndLog() {
	if influxMetricsConfig.Host == "" {
		logger.Fatal(`Despite being enabled, influx metrics came with no host info: config.metrics.influx.host = "".`)
	}
	if influxMetricsConfig.Database == "" {
		logger.Fatal(`Despite being enabled, influx metrics came with no database info: config.metrics.influx.database = "".`)
	}
	logger.Info("config.metrics.influx.host: %s", influxMetricsConfig.Host)
	logger.Info("config.metrics.influx.database: %s", influxMetricsConfig.Database)
}

type PrometheusMetrics struct {
	Port             int    `mapstructure:"port"`
	Namespace        string `mapstructure:"namespace"`
	Subsystem        string `mapstructure:"subsystem"`
	TimeoutMillisRaw int    `mapstructure:"timeout_ms"`
	Enabled          bool   `mapstructure:"enabled"`
}

func (promMetricsConfig *PrometheusMetrics) validateAndLog() {
	if promMetricsConfig.Port == 0 {
		logger.Fatal(`Despite being enabled, prometheus metrics came with an empty port number: config.metrics.prometheus.port = 0`)
	}
	if promMetricsConfig.Namespace == "" {
		logger.Fatal(`Despite being enabled, prometheus metrics came with an empty name space: config.metrics.prometheus.namespace = %s.`, promMetricsConfig.Namespace)
	}
	if promMetricsConfig.Subsystem == "" {
		logger.Fatal(`Despite being enabled, prometheus metrics came with an empty subsystem value: config.metrics.prometheus.subsystem = \"\".`)
	}
	logger.Info("config.metrics.prometheus.namespace: %s", promMetricsConfig.Namespace)
	logger.Info("config.metrics.prometheus.subsystem: %s", promMetricsConfig.Subsystem)
	logger.Info("config.metrics.prometheus.port: %d", promMetricsConfig.Port)
}

func (promMetricsConfig *PrometheusMetrics) Timeout() time.Duration {
	return time.Duration(promMetricsConfig.TimeoutMillisRaw) * time.Millisecond
}

type Routes struct {
	AllowPublicWrite bool `mapstructure:"allow_public_write"`
}

func (cfg *Routes) validateAndLog() {
	if !cfg.AllowPublicWrite {
		logger.Info("Main server will only accept GET requests")
	}
}

type Stats struct {
	StatsHost   string `mapstructure:"host"`
	StatsPort   string `mapstructure:"port"`
	StatsDCName string `mapstructure:"dc_name"`

	PortTCP                   string `mapstructure:"port_tcp"`
	PublishInterval           int    `mapstructure:"publish_interval"`
	PublishThreshold          int    `mapstructure:"publish_threshold"`
	Retries                   int    `mapstructure:"retries"`
	DialTimeout               int    `mapstructure:"dial_timeout"`
	KeepAliveDuration         int    `mapstructure:"keep_alive_duration"`
	MaxIdleConnections        int    `mapstructure:"max_idle_connections"`
	MaxIdleConnectionsPerHost int    `mapstructure:"max_idle_connections_per_host"`

	UseTCP bool `mapstructure:"use_tcp"`
}

func (cfg *Stats) validateAndLog() {
	logger.Info("config.stats.host: %s", cfg.StatsHost)
	logger.Info("config.stats.port: %s", cfg.StatsPort)
	logger.Info("config.stats.dc_name: %s", cfg.StatsDCName)

	logger.Info("config.stats.port_tcp: %s", cfg.PortTCP)
	logger.Info("config.stats.publisher_interval: %d", cfg.PublishInterval)
	logger.Info("config.stats.publisher_threshold: %d", cfg.PublishThreshold)
	logger.Info("config.stats.retries: %d", cfg.Retries)
	logger.Info("config.stats.dial_timeout: %d", cfg.DialTimeout)
	logger.Info("config.stats.keep_alive_duration: %d", cfg.KeepAliveDuration)
	logger.Info("config.stats.max_idle_connections: %d", cfg.MaxIdleConnections)
	logger.Info("config.stats.max_idle_connections_per_host: %d", cfg.MaxIdleConnectionsPerHost)

	logger.Info("config.stats.use_tcp: %t", cfg.UseTCP)
}

type Server struct {
	ServerPort string `mapstructure:"port"`
	ServerName string `mapstructure:"name"`
}

func (cfg *Server) validateAndLog() {
	logger.Info("config.server.port: %s", cfg.ServerPort)
	logger.Info("config.server.name: %s", cfg.ServerName)
}

type OWLog struct {
	LogLevel        logger.LogLevel `mapstructure:"level"`
	LogPath         string          `mapstructure:"path"`
	LogRotationTime time.Duration   `mapstructure:"rotation_time"`
	MaxLogFiles     int             `mapstructure:"max_log_files"`
	MaxLogSize      uint64          `mapstructure:"max_log_size"`
}

func (cfg *OWLog) validateAndLog() {
	logger.Info("config.ow_log.level: %v", cfg.LogLevel)
	logger.Info("config.ow_log.path: %s", cfg.LogPath)
	logger.Info("config.ow_log.rotation_time: %v", cfg.LogRotationTime)
	logger.Info("config.ow_log.max_log_files: %v", cfg.MaxLogFiles)
	logger.Info("config.ow_log.max_log_size: %v", cfg.MaxLogSize)
}

//getHostName Generates server name from node and pod name in K8S  environment
func getHostName() string {
	var (
		nodeName string
		podName  string
	)

	if nodeName, _ = os.LookupEnv(constant.ENV_VAR_NODE_NAME); nodeName == "" {
		nodeName = constant.DEFAULT_NODENAME
		logger.Info("Node name not set. Using default name: '%s'", nodeName)
	} else {
		nodeName = strings.Split(nodeName, ".")[0]
	}

	if podName, _ = os.LookupEnv(constant.ENV_VAR_POD_NAME); podName == "" {
		podName = constant.DEFAULT_PODNAME
		logger.Info("Pod name not set. Using default name: '%s'", podName)
	} else {
		podName = strings.TrimPrefix(podName, "creativecache-")
	}

	serverName := nodeName + ":" + podName
	logger.Info("Server name: '%s'", serverName)

	return serverName
}
