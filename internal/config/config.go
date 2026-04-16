package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type RPCConfig struct {
	URL             string        `yaml:"url"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	CallConcurrency int           `yaml:"call_concurrency"`
}

type SchedulerConfig struct {
	QuickInterval time.Duration `yaml:"quick_interval"`
	DailyAtUTC    string        `yaml:"daily_at_utc"`
}

type NotifyConfig struct {
	SlackWebhookEnv string `yaml:"slack_webhook_env"`
	Enabled         bool   `yaml:"enabled"`
}

type MetricsConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type VersionConfig struct {
	Name           string            `yaml:"name"`
	DataStore      string            `yaml:"data_store"`
	RoleStore      string            `yaml:"role_store"`
	GovTimelock    string            `yaml:"gov_timelock"`
	ConfigTimelock string            `yaml:"config_timelock"`
	UsdtToken      string            `yaml:"usdt_token"`
	Vaults         map[string]string `yaml:"vaults"`
	Markets        map[string]string `yaml:"markets"`
}

type Config struct {
	RPC       RPCConfig         `yaml:"rpc"`
	Scheduler SchedulerConfig   `yaml:"scheduler"`
	Notify    NotifyConfig      `yaml:"notify"`
	Metrics   MetricsConfig     `yaml:"metrics"`
	Log       LogConfig         `yaml:"log"`
	Versions  []VersionConfig   `yaml:"versions"`
	RoleNames []string          `yaml:"role_names"`
	Keepers   map[string]string `yaml:"keepers"`
}

// Load reads YAML config and applies env overrides for the most common keys.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyEnvOverrides(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("RPC_URL"); v != "" {
		cfg.RPC.URL = v
	}
	if v := os.Getenv("METRICS_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Metrics.Port = n
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("NOTIFY_ENABLED"); v != "" {
		cfg.Notify.Enabled = v == "true" || v == "1"
	}
}

func (c *Config) validate() error {
	if c.RPC.URL == "" {
		return fmt.Errorf("rpc.url is required")
	}
	if c.RPC.RequestTimeout == 0 {
		c.RPC.RequestTimeout = 30 * time.Second
	}
	if c.RPC.CallConcurrency <= 0 {
		c.RPC.CallConcurrency = 8
	}
	if c.Metrics.Port == 0 {
		c.Metrics.Port = 8080
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
	if c.Scheduler.QuickInterval == 0 {
		c.Scheduler.QuickInterval = time.Hour
	}
	if c.Scheduler.DailyAtUTC == "" {
		c.Scheduler.DailyAtUTC = "00:00"
	}
	if len(c.Versions) == 0 {
		return fmt.Errorf("at least one version must be configured")
	}
	for i, v := range c.Versions {
		if v.Name == "" || v.DataStore == "" || v.RoleStore == "" {
			return fmt.Errorf("version[%d]: name/data_store/role_store required", i)
		}
	}
	return nil
}

// SlackWebhookURL returns the Slack webhook URL from the env var named in
// notify.slack_webhook_env. Empty string if not set.
func (c *Config) SlackWebhookURL() string {
	if c.Notify.SlackWebhookEnv == "" {
		return ""
	}
	return os.Getenv(c.Notify.SlackWebhookEnv)
}
