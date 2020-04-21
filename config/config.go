package config

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewConfig() Configuration {
	v := viper.New()
	setConfigDefaults(v)
	setConfigFile(v)
	setEnvVars(v)

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg := Configuration{}
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	return cfg
}

func setConfigDefaults(v *viper.Viper) {
	v.SetDefault("rate_limiter.enabled", true)
	v.SetDefault("rate_limiter.num_requests", 100)
	v.SetDefault("request_limits.allow_setting_keys", false)
	v.SetDefault("request_limits.max_size_bytes", 10*1024)
	v.SetDefault("request_limits.max_num_values", 10)
	v.SetDefault("request_limits.max_ttl_seconds", 3600)
}

func setConfigFile(v *viper.Viper) {
	v.SetConfigName("config")              // name of config file (without extension)
	v.AddConfigPath("/etc/prebid-cache/")  // path to look for the config file in
	v.AddConfigPath("$HOME/.prebid-cache") // call multiple times to add many search paths
	v.AddConfigPath(".")
}

func setEnvVars(v *viper.Viper) {
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("PBC")
	v.AutomaticEnv()
}

type Configuration struct {
	Port          int           `mapstructure:"port"`
	AdminPort     int           `mapstructure:"admin_port"`
	Log           Log           `mapstructure:"log"`
	RateLimiting  RateLimiting  `mapstructure:"rate_limiter"`
	RequestLimits RequestLimits `mapstructure:"request_limits"`
	Backend       Backend       `mapstructure:"backend"`
	Compression   Compression   `mapstructure:"compression"`
	Metrics       Metrics       `mapstructure:"metrics"`
}

// ValidateAndLog validates the config, terminating the program on any errors.
// It also logs the config values that it used.
func (cfg *Configuration) ValidateAndLog() error {
	log.Infof("config.port: %d", cfg.Port)
	log.Infof("config.admin_port: %d", cfg.AdminPort)
	cfg.Log.validateAndLog()
	cfg.RateLimiting.validateAndLog()
	cfg.RequestLimits.validateAndLog()
	if err := cfg.Backend.validateAndLog(); err != nil {
		return err
	}
	cfg.Compression.validateAndLog()
	if err := cfg.Metrics.validateAndLog(); err != nil {
		return err
	}
	return nil
}

type Log struct {
	Level LogLevel `mapstructure:"level"`
}

func (cfg *Log) validateAndLog() {
	log.Infof("config.log.level: %s", cfg.Level)
}

type LogLevel string

const (
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
	log.Infof("config.rate_limiter.enabled: %t", cfg.Enabled)
	log.Infof("config.rate_limiter.num_requests: %d", cfg.MaxRequestsPerSecond)
}

type RequestLimits struct {
	MaxSize          int  `mapstructure:"max_size_bytes"`
	MaxNumValues     int  `mapstructure:"max_num_values"`
	MaxTTLSeconds    int  `mapstructure:"max_ttl_seconds"`
	AllowSettingKeys bool `mapstructure:"allow_setting_keys"`
}

func (cfg *RequestLimits) validateAndLog() {
	log.Infof("config.request_limits.allow_setting_keys: %v", cfg.AllowSettingKeys)
	log.Infof("config.request_limits.max_ttl_seconds: %d", cfg.MaxTTLSeconds)
	log.Infof("config.request_limits.max_size_bytes: %d", cfg.MaxSize)
	log.Infof("config.request_limits.max_num_values: %d", cfg.MaxNumValues)
}

type Compression struct {
	Type CompressionType `mapstructure:"type"`
}

func (cfg *Compression) validateAndLog() {
	switch cfg.Type {
	case CompressionNone:
		fallthrough
	case CompressionSnappy:
		log.Infof("config.compression.type: %s", cfg.Type)
	default:
		//err = fmt.Errorf(`invalid config.compression.type: %s. It must be "none" or "snappy"`, cfg.Type)
		log.Fatalf(`invalid config.compression.type: %s. It must be "none" or "snappy"`, cfg.Type)
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

func (cfg *Metrics) validateAndLog() error {
	metricsEnabled := false

	if cfg.Type == MetricsInflux || cfg.Influx.Enabled {
		if err := cfg.Influx.validateAndLog(); err != nil {
			return err
		}
		metricsEnabled = true
	}

	if cfg.Prometheus.Enabled {
		if err := cfg.Prometheus.validateAndLog(); err != nil {
			return err
		}
		metricsEnabled = true
	}

	// Was any other metrics system besides "InfluxDB" or "Prometheus" specified in `cfg.Type`?
	if cfg.Type != MetricsNone && cfg.Type != MetricsInflux && cfg.Type != "" {
		if metricsEnabled {
			// Prometheus, Influx or both, are enabled. Log a message explaining that `prebid-cache` will
			// continue with supported metrics and non-supported metrics will be disabled
			log.Infof("Prebid Cache will continue without the use of unsupported metrics \"%s\".", cfg.Type)
		} else {
			// THe only metrics engine specified in the configuration file is a non-supported
			// metrics engine, log error and exit program
			return fmt.Errorf("Metrics \"%s\" are not supported, exiting program.", cfg.Type)
		}
	}
	return nil
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

func (influxMetricsConfig *InfluxMetrics) validateAndLog() error {
	if influxMetricsConfig.Host == "" {
		return fmt.Errorf(`Despite being enabled, influx metrics came with no host info: config.metrics.influx.host = "".`)
	}
	if influxMetricsConfig.Database == "" {
		return fmt.Errorf(`Despite being enabled, influx metrics came with no database info: config.metrics.influx.database = "".`)
	}
	log.Infof("config.metrics.influx.host: %s", influxMetricsConfig.Host)
	log.Infof("config.metrics.influx.database: %s", influxMetricsConfig.Database)

	return nil
}

type PrometheusMetrics struct {
	Port             int    `mapstructure:"port"`
	Namespace        string `mapstructure:"namespace"`
	Subsystem        string `mapstructure:"subsystem"`
	TimeoutMillisRaw int    `mapstructure:"timeout_ms"`
	Enabled          bool   `mapstructure:"enabled"`
}

func (promMetricsConfig *PrometheusMetrics) validateAndLog() error {
	if promMetricsConfig.Port == 0 {
		return fmt.Errorf(`Despite being enabled, prometheus metrics came with an empty port number: config.metrics.prometheus.port = 0`)
	}
	if promMetricsConfig.Namespace == "" {
		return fmt.Errorf(`Despite being enabled, prometheus metrics came with an empty name space: config.metrics.prometheus.namespace = %s.`, promMetricsConfig.Namespace)
	}
	if promMetricsConfig.Subsystem == "" {
		return fmt.Errorf(`Despite being enabled, prometheus metrics came with an empty subsystem value: config.metrics.prometheus.subsystem = \"\".`)
	}
	log.Infof("config.metrics.prometheus.namespace: %s", promMetricsConfig.Namespace)
	log.Infof("config.metrics.prometheus.subsystem: %s", promMetricsConfig.Subsystem)
	log.Infof("config.metrics.prometheus.port: %d", promMetricsConfig.Port)

	return nil
}

func (m *PrometheusMetrics) Timeout() time.Duration {
	return time.Duration(m.TimeoutMillisRaw) * time.Millisecond
}
