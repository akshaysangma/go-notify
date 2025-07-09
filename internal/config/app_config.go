package config

import "time"

// AppConfig holds the entire application configuration.
type AppConfig struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Webhook   WebhookConfig   `mapstructure:"webhook"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	App       AppEnvConfig    `mapstructure:"app"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	GracePeriod  time.Duration `mapstructure:"grace_period"`
}

// DatabaseConfig holds PostgreSQL database configuration.
type DatabaseConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
	MaxConns         int32  `mapstructure:"max_conns"`
}

// RedisConfig holds Redis caching configuration.
type RedisConfig struct {
	Address string `mapstructure:"address"`
}

// WebhookConfig holds webhook.site configuration.
type WebhookConfig struct {
	URL            string `mapstructure:"url"`
	CharacterLimit int    `mapstructure:"character_limit"`
}

// SchedulerConfig holds the message dispatch scheduler configuration.
type SchedulerConfig struct {
	MessageRate int           `mapstructure:"message_rate"`
	RunsEvery   time.Duration `mapstructure:"runs_every"`
	GracePeriod time.Duration `mapstructure:"grace_period"`
	JobTimeout  time.Duration `mapstructure:"job_timeout"`
}

// AppEnvConfig holds application environment settings.
type AppEnvConfig struct {
	Environment string `mapstructure:"environment"`
}
