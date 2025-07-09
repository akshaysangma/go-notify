package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// LoadConfig loads application configuration from file and environment variables
func LoadConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 2. Add these lines to help Viper correctly map and prioritize environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("WARNING: Config file not found. Using environment variables.")
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config into struct: %w", err)
	}

	// Validate essential configurations
	if cfg.Database.ConnectionString == "" {
		return nil, fmt.Errorf("database connection string is not configured")
	}
	if cfg.Webhook.URL == "" {
		return nil, fmt.Errorf("webhook URL is not configured")
	}

	if cfg.Scheduler.MessageRate <= 0 {
		fmt.Println("WARNING: Scheduler Message Rate set to 0 or less, defaulting to 2")
		cfg.Scheduler.MessageRate = 2
	}
	if cfg.Scheduler.RunsEvery <= 0*time.Second {
		fmt.Println("WARNING: Scheduler interval set to 0 or less, defaulting to 2 minute")
		cfg.Scheduler.RunsEvery = 2 * time.Minute
	}
	if cfg.Scheduler.GracePeriod <= 0*time.Second || cfg.Scheduler.GracePeriod >= cfg.Scheduler.RunsEvery {
		fmt.Println("WARNING: Scheduler grace period set to 0 or greater than scheduler Interval, defaulting to 30 secs")
		cfg.Scheduler.GracePeriod = 30 * time.Second
	}

	return &cfg, nil
}
