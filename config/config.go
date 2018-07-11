package config

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/prebid/prebid-cache/stats"
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
	//Initialize Stats Server
	stats.InitStat(cfg.Stats.StatsHost, cfg.Stats.StatsPort,
		cfg.Server.ServerName,
		cfg.Stats.StatsDCName)
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
	Stats         Stats         `mapstructure:"stats"`
	Server        Server        `mapstructure:"server"`
}

// ValidateAndLog validates the config, terminating the program on any errors.
// It also logs the config values that it used.
func (cfg *Configuration) ValidateAndLog() {
	log.Infof("config.port: %d", cfg.Port)
	log.Infof("config.admin_port: %d", cfg.AdminPort)
	cfg.Log.validateAndLog()
	cfg.RateLimiting.validateAndLog()
	cfg.RequestLimits.validateAndLog()
	cfg.Backend.validateAndLog()
	cfg.Compression.validateAndLog()
	cfg.Metrics.validateAndLog()
	cfg.Stats.validateAndLog()
	cfg.Server.validateAndLog()
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
	MaxSize      int `mapstructure:"max_size_bytes"`
	MaxNumValues int `mapstructure:"max_num_values"`
}

func (cfg *RequestLimits) validateAndLog() {
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
		log.Fatalf(`invalid config.compression.type: %s. It must be "none" or "snappy"`, cfg.Type)
	}
}

type CompressionType string

const (
	CompressionNone   CompressionType = "none"
	CompressionSnappy CompressionType = "snappy"
)

type Metrics struct {
	Type   MetricsType `mapstructure:"type"`
	Influx Influx      `mapstructure:"influx"`
}

func (cfg *Metrics) validateAndLog() {
	log.Infof("config.metrics.type: %s", cfg.Type)
	switch cfg.Type {
	case MetricsNone:
	case MetricsInflux:
		cfg.Influx.validateAndLog()
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

func (cfg *Influx) validateAndLog() {
	log.Infof("config.metrics.influx.host: %s", cfg.Host)
	log.Infof("config.metrics.influx.database: %s", cfg.Database)
	// This intentionally skips username and password for security reasons.
}

type Stats struct {
	StatsHost   string `mapstructure:"host"`
	StatsPort   string `mapstructure:"port"`
	StatsDCName string `mapstructure:"dc_name"`
}

func (cfg *Stats) validateAndLog() {
	log.Infof("config.stats.host: %s", cfg.StatsHost)
	log.Infof("config.stats.port: %s", cfg.StatsPort)
	log.Infof("config.stats.dc_name: %s", cfg.StatsDCName)
}

type Server struct {
	ServerPort string `mapstructure:"port"`
	ServerName string `mapstructure:"name"`
}

func (cfg *Server) validateAndLog() {
	log.Infof("config.server.port: %s", cfg.ServerPort)
	log.Infof("config.server.name: %s", cfg.ServerName)
}

