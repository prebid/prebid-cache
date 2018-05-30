package config

import (
	"strings"

	log "github.com/Sirupsen/logrus"
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
	v.SetDefault("request_limits.max_size_bytes", 10*1024)
	v.SetDefault("request_limits.max_num_values", 10)
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

func (cfg *Configuration) LogValues() {
	log.Infof("config.port: %d", cfg.Port)
	log.Infof("config.admin_port: %d", cfg.AdminPort)
	cfg.Log.logValues()
	cfg.RateLimiting.logValues()
	cfg.RequestLimits.logValues()
	cfg.Backend.logValues()
	cfg.Compression.logValues()
	cfg.Metrics.logValues()
}

type Log struct {
	Level LogLevel `mapstructure:"level"`
}

func (cfg *Log) logValues() {
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

func (cfg *RateLimiting) logValues() {
	log.Infof("config.rate_limiter.enabled: %t", cfg.Enabled)
	log.Infof("config.rate_limiter.num_requests: %d", cfg.MaxRequestsPerSecond)
}

type RequestLimits struct {
	MaxSize      int `mapstructure:"max_size_bytes"`
	MaxNumValues int `mapstructure:"max_num_values"`
}

func (cfg *RequestLimits) logValues() {
	log.Infof("config.request_limits.max_size_bytes: %d", cfg.MaxSize)
	log.Infof("config.request_limits.max_num_values: %d", cfg.MaxNumValues)
}

type Compression struct {
	Type CompressionType
}

func (cfg *Compression) logValues() {
	log.Infof("config.compression.type: %s", cfg.Type)
}

type CompressionType string

const (
	CompressionSnappy CompressionType = "snappy"
)

type Metrics struct {
	Type   MetricsType `mapstructure:"type"`
	Influx Influx      `mapstructure:"influx"`
}

func (cfg *Metrics) logValues() {
	log.Infof("config.metrics.type: %s", cfg.Type)
	switch cfg.Type {
	case MetricsNone:
	case MetricsInflux:
		cfg.Influx.logValues()
	default:
		log.Fatalf(`invalid config.metrics.type: %s. It must be "none" or "influx"`, cfg.Type)
	}
}

type MetricsType string

const (
	MetricsNone   MetricsType = "none"
	MetricsInflux MetricsType = "influx"
)

type Influx struct {
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (cfg *Influx) logValues() {
	log.Infof("config.metrics.influx.host: %s", cfg.Host)
	log.Infof("config.metrics.influx.database: %s", cfg.Database)
	// This intentionally skips username and password for security reasons.
}
