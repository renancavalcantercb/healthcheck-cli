package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/env"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/security"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Global        GlobalConfig   `yaml:"global"`
	Checks        []CheckConfig  `yaml:"checks"`
	Notifications Notifications `yaml:"notifications"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	MaxWorkers        int                           `yaml:"max_workers"`
	DefaultTimeout    time.Duration                 `yaml:"default_timeout"`
	DefaultInterval   time.Duration                 `yaml:"default_interval"`
	StoragePath       string                        `yaml:"storage_path"`
	LogLevel          string                        `yaml:"log_level"`
	DisableColors     bool                          `yaml:"disable_colors"`
	UserAgent         string                        `yaml:"user_agent"`
	MaxRetries        int                           `yaml:"max_retries"`
	RetryDelay        time.Duration                 `yaml:"retry_delay"`
	RateLimit         types.RateLimitConfig         `yaml:"rate_limit"`
	CircuitBreaker    types.CircuitBreakerConfig    `yaml:"circuit_breaker"`
	MemoryManagement  types.MemoryManagementConfig  `yaml:"memory_management"`
}

// CheckConfig wraps the types.CheckConfig with YAML tags
type CheckConfig struct {
	types.CheckConfig `yaml:",inline"`
}

// Notifications contains notification settings
type Notifications struct {
	Email       EmailConfig       `yaml:"email"`
	Slack       SlackConfig       `yaml:"slack"`
	Webhook     WebhookConfig     `yaml:"webhook"`
	Discord     DiscordConfig     `yaml:"discord"`
	Telegram    TelegramConfig    `yaml:"telegram"`
	GlobalRules NotificationRules `yaml:"global_rules"`
}

// EmailConfig contains email notification settings
type EmailConfig struct {
	Enabled    bool              `yaml:"enabled"`
	SMTPHost   string            `yaml:"smtp_host"`
	SMTPPort   int               `yaml:"smtp_port"`
	Username   string            `yaml:"username"`
	Password   string            `yaml:"password"`
	From       string            `yaml:"from"`
	To         []string          `yaml:"to"`
	Subject    string            `yaml:"subject"`
	Template   string            `yaml:"template"`
	TLS        bool              `yaml:"tls"`
}

// SlackConfig contains Slack notification settings
type SlackConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
	IconEmoji  string `yaml:"icon_emoji"`
	Template   string `yaml:"template"`
}

// WebhookConfig contains generic webhook settings
type WebhookConfig struct {
	Enabled bool              `yaml:"enabled"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Timeout time.Duration     `yaml:"timeout"`
}

// DiscordConfig contains Discord webhook settings
type DiscordConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Username   string `yaml:"username"`
	AvatarURL  string `yaml:"avatar_url"`
}

// TelegramConfig contains Telegram bot settings
type TelegramConfig struct {
	Enabled bool   `yaml:"enabled"`
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

// NotificationRules defines when and how to send notifications
type NotificationRules struct {
	OnSuccess       bool          `yaml:"on_success"`
	OnFailure       bool          `yaml:"on_failure"`
	OnRecovery      bool          `yaml:"on_recovery"`
	OnSlowResponse  bool          `yaml:"on_slow_response"`
	Cooldown        time.Duration `yaml:"cooldown"`
	MaxAlerts       int           `yaml:"max_alerts"`
	EscalationDelay time.Duration `yaml:"escalation_delay"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			MaxWorkers:      10,
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 30 * time.Second,
			StoragePath:     "./healthcheck.db",
			LogLevel:        "info",
			DisableColors:   false,
			UserAgent:       "HealthCheck-CLI/1.0",
			MaxRetries:      3,
			RetryDelay:      5 * time.Second,
			RateLimit: types.RateLimitConfig{
				Enabled:      true,
				DefaultLimit: 1.0, // 1 request per second by default
				DefaultBurst: 5,   // Allow burst of 5 requests
				PerEndpoint:  true, // Enable per-endpoint rate limiting
			},
			CircuitBreaker: types.CircuitBreakerConfig{
				Enabled:          true,
				MaxFailures:      5,                // Open circuit after 5 failures
				Timeout:          60 * time.Second, // Wait 60 seconds before trying again
				SuccessThreshold: 3,                // Need 3 successes to close circuit
			},
			MemoryManagement: types.MemoryManagementConfig{
				Enabled:                true,
				MaxHistoryPerService:   100,                // Keep last 100 results per service
				MaxHistoryAge:          24 * time.Hour,     // Remove results older than 24 hours
				CleanupInterval:        5 * time.Minute,    // Run cleanup every 5 minutes
				MaxTotalMemoryMB:       100,                // Limit total memory usage to 100MB
			},
		},
		Notifications: Notifications{
			GlobalRules: NotificationRules{
				OnSuccess:       false,
				OnFailure:       true,
				OnRecovery:      true,
				OnSlowResponse:  true,
				Cooldown:        5 * time.Minute,
				MaxAlerts:       10,
				EscalationDelay: 15 * time.Minute,
			},
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(filePath string) (*Config, error) {
	// Start with defaults
	config := DefaultConfig()
	
	// If no file specified, return defaults
	if filePath == "" {
		return config, nil
	}
	
	// Validate file path for security
	if err := security.ValidateFilePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid config file path: %w", err)
	}
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", filePath)
	}
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, config)
	case ".json":
		// Note: yaml.Unmarshal can also handle JSON
		err = yaml.Unmarshal(data, config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json)", ext)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Expand environment variables in the configuration
	if err := config.ExpandEnvironmentVariables(); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}
	
	// Validate and apply defaults
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	config.ApplyDefaults()
	
	return config, nil
}

// LoadConfigWithEnv loads configuration from a file and optionally loads environment variables from .env file
func LoadConfigWithEnv(filePath, envPath string) (*Config, error) {
	// Load .env file if specified
	if envPath != "" {
		if err := env.LoadEnvFile(envPath); err != nil {
			return nil, fmt.Errorf("failed to load environment file: %w", err)
		}
	}
	
	// Load configuration
	return LoadConfig(filePath)
}

// ExpandEnvironmentVariables expands environment variables in the configuration
func (c *Config) ExpandEnvironmentVariables() error {
	// Convert config to map for recursive processing
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config for env expansion: %w", err)
	}
	
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config for env expansion: %w", err)
	}
	
	// Expand environment variables
	env.ExpandEnvironmentVariablesInMap(configMap)
	
	// Convert back to config struct
	expandedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal expanded config: %w", err)
	}
	
	if err := yaml.Unmarshal(expandedData, c); err != nil {
		return fmt.Errorf("failed to unmarshal expanded config: %w", err)
	}
	
	return nil
}

// ValidateEnvironmentVariables validates that required environment variables are set
func (c *Config) ValidateEnvironmentVariables() error {
	var requiredVars []string
	
	// Check email configuration
	if c.Notifications.Email.Enabled {
		if strings.Contains(c.Notifications.Email.Password, "${") {
			requiredVars = append(requiredVars, "EMAIL_PASSWORD")
		}
	}
	
	// Check Slack configuration
	if c.Notifications.Slack.Enabled {
		if strings.Contains(c.Notifications.Slack.WebhookURL, "${") {
			requiredVars = append(requiredVars, "SLACK_WEBHOOK_URL")
		}
	}
	
	// Check Discord configuration
	if c.Notifications.Discord.Enabled {
		if strings.Contains(c.Notifications.Discord.WebhookURL, "${") {
			requiredVars = append(requiredVars, "DISCORD_WEBHOOK_URL")
		}
	}
	
	// Check Telegram configuration
	if c.Notifications.Telegram.Enabled {
		if strings.Contains(c.Notifications.Telegram.BotToken, "${") {
			requiredVars = append(requiredVars, "TELEGRAM_BOT_TOKEN")
		}
	}
	
	// Check webhook configuration
	if c.Notifications.Webhook.Enabled {
		for key, value := range c.Notifications.Webhook.Headers {
			if strings.Contains(value, "${") {
				requiredVars = append(requiredVars, strings.ToUpper(key)+"_TOKEN")
			}
		}
	}
	
	// Check checks for API tokens in headers
	for _, check := range c.Checks {
		for key, value := range check.Headers {
			if strings.Contains(value, "${") && strings.Contains(strings.ToLower(key), "authorization") {
				requiredVars = append(requiredVars, "API_TOKEN")
			}
		}
	}
	
	// Validate required variables are set
	return env.ValidateRequiredEnvVars(requiredVars)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate global settings
	if c.Global.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be greater than 0")
	}
	
	if c.Global.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be greater than 0")
	}
	
	if c.Global.DefaultInterval <= 0 {
		return fmt.Errorf("default_interval must be greater than 0")
	}
	
	// Validate checks
	if len(c.Checks) == 0 {
		return fmt.Errorf("at least one check must be defined")
	}
	
	for i, check := range c.Checks {
		if err := c.validateCheck(check, i); err != nil {
			return err
		}
	}
	
	// Validate notifications
	if err := c.validateNotifications(); err != nil {
		return err
	}
	
	return nil
}

func (c *Config) validateCheck(check CheckConfig, index int) error {
	if check.Name == "" {
		return fmt.Errorf("check[%d]: name is required", index)
	}
	
	if check.URL == "" {
		return fmt.Errorf("check[%d]: URL is required", index)
	}
	
	// Validate URL format based on type
	switch check.Type {
	case types.CheckTypeHTTP:
		if !strings.HasPrefix(check.URL, "http://") && !strings.HasPrefix(check.URL, "https://") {
			return fmt.Errorf("check[%d]: HTTP checks require http:// or https:// URL", index)
		}
	case types.CheckTypeTCP:
		if strings.Contains(check.URL, "://") {
			return fmt.Errorf("check[%d]: TCP checks should use host:port format", index)
		}
	case types.CheckTypeSSL:
		// SSL checks can accept both URL format and host:port format
		// No strict validation needed as the SSL checker handles both
	}
	
	if check.Interval <= 0 {
		return fmt.Errorf("check[%d]: interval must be greater than 0", index)
	}
	
	if check.Timeout <= 0 {
		return fmt.Errorf("check[%d]: timeout must be greater than 0", index)
	}
	
	if check.Timeout >= check.Interval {
		return fmt.Errorf("check[%d]: timeout must be less than interval", index)
	}
	
	return nil
}

func (c *Config) validateNotifications() error {
	// Validate email config
	if c.Notifications.Email.Enabled {
		if c.Notifications.Email.SMTPHost == "" {
			return fmt.Errorf("email notifications enabled but smtp_host not configured")
		}
		if c.Notifications.Email.From == "" {
			return fmt.Errorf("email notifications enabled but from address not configured")
		}
		if len(c.Notifications.Email.To) == 0 {
			return fmt.Errorf("email notifications enabled but no recipients configured")
		}
	}
	
	// Validate Slack config
	if c.Notifications.Slack.Enabled {
		if c.Notifications.Slack.WebhookURL == "" {
			return fmt.Errorf("slack notifications enabled but webhook_url not configured")
		}
	}
	
	// Validate webhook config
	if c.Notifications.Webhook.Enabled {
		if c.Notifications.Webhook.URL == "" {
			return fmt.Errorf("webhook notifications enabled but url not configured")
		}
	}
	
	return nil
}

// ApplyDefaults applies default values to checks that don't have them specified
func (c *Config) ApplyDefaults() {
	for i := range c.Checks {
		check := &c.Checks[i]
		
		// Apply default type
		if check.Type == "" {
			if strings.HasPrefix(check.URL, "http") {
				check.Type = types.CheckTypeHTTP
			} else {
				check.Type = types.CheckTypeTCP
			}
		}
		
		// Apply default method for HTTP checks
		if check.Type == types.CheckTypeHTTP && check.Method == "" {
			check.Method = "GET"
		}
		
		// Apply default timeout
		if check.Timeout == 0 {
			check.Timeout = c.Global.DefaultTimeout
		}
		
		// Apply default interval
		if check.Interval == 0 {
			check.Interval = c.Global.DefaultInterval
		}
		
		// Apply default expected status
		if check.Expected.Status == 0 {
			check.Expected.Status = 200
		}
		
		// Apply default retry config
		if check.Retry.Attempts == 0 {
			check.Retry.Attempts = c.Global.MaxRetries
		}
		if check.Retry.Delay == 0 {
			check.Retry.Delay = c.Global.RetryDelay
		}
		if check.Retry.Backoff == "" {
			check.Retry.Backoff = "exponential"
		}
	}
	
	// Apply notification defaults
	if c.Notifications.Email.SMTPPort == 0 {
		c.Notifications.Email.SMTPPort = 587
	}
	if c.Notifications.Email.Subject == "" {
		c.Notifications.Email.Subject = "HealthCheck Alert: {{.Name}}"
	}
	if c.Notifications.Webhook.Method == "" {
		c.Notifications.Webhook.Method = "POST"
	}
	if c.Notifications.Webhook.Timeout == 0 {
		c.Notifications.Webhook.Timeout = 10 * time.Second
	}
}

// SaveExample saves an example configuration file
func SaveExample(filePath string) error {
	config := &Config{
		Global: GlobalConfig{
			MaxWorkers:      20,
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 30 * time.Second,
			StoragePath:     "./healthcheck.db",
			LogLevel:        "info",
			UserAgent:       "HealthCheck-CLI/1.0",
		},
		Checks: []CheckConfig{
			{
				CheckConfig: types.CheckConfig{
					Name:     "API Health",
					Type:     types.CheckTypeHTTP,
					URL:      "https://api.example.com/health",
					Method:   "GET",
					Interval: 30 * time.Second,
					Timeout:  10 * time.Second,
					Headers: map[string]string{
						"Authorization": "Bearer ${API_TOKEN}",
						"Accept":        "application/json",
					},
					Expected: types.Expected{
						Status:          200,
						BodyContains:    "healthy",
						ResponseTimeMax: 2 * time.Second,
					},
					Retry: types.RetryConfig{
						Attempts: 3,
						Delay:    5 * time.Second,
						Backoff:  "exponential",
					},
					Tags: []string{"api", "critical"},
				},
			},
			{
				CheckConfig: types.CheckConfig{
					Name:     "Database Connection",
					Type:     types.CheckTypeTCP,
					URL:      "db.example.com:5432",
					Interval: 60 * time.Second,
					Timeout:  5 * time.Second,
					Tags:     []string{"database", "infrastructure"},
				},
			},
			{
				CheckConfig: types.CheckConfig{
					Name:     "Google DNS",
					Type:     types.CheckTypeTCP,
					URL:      "8.8.8.8:53",
					Interval: 120 * time.Second,
					Timeout:  3 * time.Second,
					Expected: types.Expected{
						ResponseTimeMax: 100 * time.Millisecond,
					},
					Tags: []string{"dns", "external"},
				},
			},
			{
				CheckConfig: types.CheckConfig{
					Name:     "HTTPBin Test",
					Type:     types.CheckTypeHTTP,
					URL:      "https://httpbin.org/get",
					Method:   "GET",
					Interval: 45 * time.Second,
					Timeout:  15 * time.Second,
					Expected: types.Expected{
						Status:          200,
						BodyContains:    "origin",
						ResponseTimeMax: 3 * time.Second,
						ContentType:     "application/json",
					},
					Tags: []string{"test", "external"},
				},
			},
			{
				CheckConfig: types.CheckConfig{
					Name:     "GitHub SSL Certificate",
					Type:     types.CheckTypeSSL,
					URL:      "https://github.com",
					Interval: 24 * time.Hour, // Check once per day
					Timeout:  10 * time.Second,
					Expected: types.Expected{
						CertExpiryDays:   30, // Alert if expires within 30 days
						CertValidDomains: []string{"github.com", "www.github.com"},
						ResponseTimeMax:  2 * time.Second,
					},
					Tags: []string{"ssl", "external", "critical"},
				},
			},
			{
				CheckConfig: types.CheckConfig{
					Name:     "Internal API SSL",
					Type:     types.CheckTypeSSL,
					URL:      "api.example.com:443",
					Interval: 12 * time.Hour, // Check twice per day
					Timeout:  5 * time.Second,
					Expected: types.Expected{
						CertExpiryDays:   14, // Alert if expires within 14 days
						CertValidDomains: []string{"api.example.com"},
						ResponseTimeMax:  1 * time.Second,
					},
					Tags: []string{"ssl", "internal", "api"},
				},
			},
		},
		Notifications: Notifications{
			Email: EmailConfig{
				Enabled:  false, // Disabled by default
				SMTPHost: "smtp.gmail.com",
				SMTPPort: 587,
				Username: "alerts@example.com",
				Password: "${EMAIL_PASSWORD}",
				From:     "alerts@example.com",
				To:       []string{"team@example.com"},
				Subject:  "🚨 HealthCheck Alert: {{.Name}}",
				TLS:      true,
			},
			Slack: SlackConfig{
				Enabled:    false, // Disabled by default
				WebhookURL: "${SLACK_WEBHOOK_URL}",
				Channel:    "#alerts",
				Username:   "HealthCheck Bot",
				IconEmoji:  ":hospital:",
			},
			GlobalRules: NotificationRules{
				OnSuccess:       false,
				OnFailure:       true,
				OnRecovery:      true,
				OnSlowResponse:  true,
				Cooldown:        5 * time.Minute,
				MaxAlerts:       10,
				EscalationDelay: 15 * time.Minute,
			},
		},
	}
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal example config: %w", err)
	}
	
	// Add header comment
	header := `# HealthCheck CLI Configuration Example
# 
# This file contains example configurations for monitoring various endpoints.
# Copy this file and modify it according to your needs.
#
# Environment variables can be used with ${VAR_NAME} syntax.
# Example: password: "${EMAIL_PASSWORD}"
#
# For more information, visit: https://github.com/your-username/healthcheck-cli

`
	
	fullContent := header + string(data)
	
	if filePath == "" {
		// Output to stdout
		fmt.Print(fullContent)
		return nil
	}
	
	// Use secure file permissions (0600 - owner read/write only) for config files
	if err := os.WriteFile(filePath, []byte(fullContent), 0600); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}
	
	return nil
}