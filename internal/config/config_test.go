package config

import (
	"os"
	"testing"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_EMAIL_PASSWORD", "secret123")
	os.Setenv("TEST_SLACK_WEBHOOK", "https://hooks.slack.com/test")
	os.Setenv("TEST_API_TOKEN", "bearer_token_456")
	defer func() {
		os.Unsetenv("TEST_EMAIL_PASSWORD")
		os.Unsetenv("TEST_SLACK_WEBHOOK")
		os.Unsetenv("TEST_API_TOKEN")
	}()

	config := &Config{
		Notifications: Notifications{
			Email: EmailConfig{
				Enabled:  true,
				Password: "${TEST_EMAIL_PASSWORD}",
				Username: "test@example.com",
			},
			Slack: SlackConfig{
				Enabled:    true,
				WebhookURL: "${TEST_SLACK_WEBHOOK}",
			},
		},
		Checks: []CheckConfig{
			{
				CheckConfig: types.CheckConfig{
					Name: "API Test",
					URL:  "https://api.example.com",
					Headers: map[string]string{
						"Authorization": "Bearer ${TEST_API_TOKEN}",
						"Content-Type":  "application/json",
					},
				},
			},
		},
	}

	err := config.ExpandEnvironmentVariables()
	require.NoError(t, err)

	// Verify environment variables were expanded
	assert.Equal(t, "secret123", config.Notifications.Email.Password)
	assert.Equal(t, "https://hooks.slack.com/test", config.Notifications.Slack.WebhookURL)
	assert.Equal(t, "Bearer bearer_token_456", config.Checks[0].Headers["Authorization"])
	assert.Equal(t, "application/json", config.Checks[0].Headers["Content-Type"]) // Unchanged
}

func TestExpandEnvironmentVariables_WithDefaults(t *testing.T) {
	// Don't set the environment variable to test default values
	os.Unsetenv("MISSING_VAR")

	config := &Config{
		Notifications: Notifications{
			Email: EmailConfig{
				Enabled:  true,
				Password: "${MISSING_VAR:default_password}",
				SMTPHost: "${MISSING_VAR}",
			},
		},
	}

	err := config.ExpandEnvironmentVariables()
	require.NoError(t, err)

	// Should use default value when provided
	assert.Equal(t, "default_password", config.Notifications.Email.Password)
	// Should keep placeholder when no default and var is missing
	assert.Equal(t, "${MISSING_VAR}", config.Notifications.Email.SMTPHost)
}

func TestValidateEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		setup   func()
		cleanup func()
		wantErr bool
		errMsg  string
	}{
		{
			name: "AllRequiredVarsPresent",
			config: &Config{
				Notifications: Notifications{
					Email: EmailConfig{
						Enabled:  true,
						Password: "${EMAIL_PASSWORD}",
					},
					Slack: SlackConfig{
						Enabled:    true,
						WebhookURL: "${SLACK_WEBHOOK_URL}",
					},
				},
			},
			setup: func() {
				os.Setenv("EMAIL_PASSWORD", "test123")
				os.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/test")
			},
			cleanup: func() {
				os.Unsetenv("EMAIL_PASSWORD")
				os.Unsetenv("SLACK_WEBHOOK_URL")
			},
			wantErr: false,
		},
		{
			name: "MissingRequiredVars",
			config: &Config{
				Notifications: Notifications{
					Email: EmailConfig{
						Enabled:  true,
						Password: "${MISSING_EMAIL_PASSWORD}",
					},
				},
			},
			setup:   func() { os.Unsetenv("MISSING_EMAIL_PASSWORD") },
			cleanup: func() {},
			wantErr: true,
			errMsg:  "required environment variables not set",
		},
		{
			name: "DisabledNotificationsNoVarsRequired",
			config: &Config{
				Notifications: Notifications{
					Email: EmailConfig{
						Enabled:  false, // Disabled, so no vars required
						Password: "${EMAIL_PASSWORD}",
					},
				},
			},
			setup:   func() { os.Unsetenv("EMAIL_PASSWORD") },
			cleanup: func() {},
			wantErr: false,
		},
		{
			name: "HardcodedValuesNoVarsRequired",
			config: &Config{
				Notifications: Notifications{
					Email: EmailConfig{
						Enabled:  true,
						Password: "hardcoded_password", // No env var placeholder
					},
				},
			},
			setup:   func() {},
			cleanup: func() {},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			err := tt.config.ValidateEnvironmentVariables()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfigWithEnv(t *testing.T) {
	// Create temporary config file
	configContent := `global:
  max_workers: 10
  default_timeout: 15s

notifications:
  email:
    enabled: true
    smtp_host: smtp.gmail.com
    username: test@example.com
    password: "${EMAIL_PASSWORD}"
    from: test@example.com
    to:
      - recipient@example.com
  slack:
    enabled: true
    webhook_url: "${SLACK_WEBHOOK_URL}"

checks:
  - name: "Test API"
    url: "https://api.example.com"
    type: http
    method: GET
    interval: 30s
    timeout: 10s
    headers:
      Authorization: "Bearer ${API_TOKEN}"
`

	configFile := "/tmp/test_config.yml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	// Create temporary .env file
	envContent := `EMAIL_PASSWORD=secret123
SLACK_WEBHOOK_URL=https://hooks.slack.com/test
API_TOKEN=bearer_token_456`

	envFile := "/tmp/test.env"
	err = os.WriteFile(envFile, []byte(envContent), 0644)
	require.NoError(t, err)
	defer os.Remove(envFile)

	// Load config with environment file
	config, err := LoadConfigWithEnv(configFile, envFile)
	require.NoError(t, err)

	// Verify environment variables were loaded and expanded
	assert.Equal(t, "secret123", config.Notifications.Email.Password)
	assert.Equal(t, "https://hooks.slack.com/test", config.Notifications.Slack.WebhookURL)
	assert.Equal(t, "Bearer bearer_token_456", config.Checks[0].Headers["Authorization"])

	// Verify config structure
	assert.Equal(t, 10, config.Global.MaxWorkers)
	assert.Equal(t, 15*time.Second, config.Global.DefaultTimeout)
	assert.True(t, config.Notifications.Email.Enabled)
	assert.Equal(t, "Test API", config.Checks[0].Name)
}

func TestLoadConfigWithEnv_MissingEnvFile(t *testing.T) {
	// Create temporary config file
	configContent := `global:
  max_workers: 5`

	configFile := "/tmp/test_config_simple.yml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	// Try to load with non-existent env file
	_, err = LoadConfigWithEnv(configFile, "/tmp/nonexistent.env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load environment file")
}

func TestConfigSecurityPatterns(t *testing.T) {
	// Test that sensitive values are properly handled
	os.Setenv("SECURE_PASSWORD", "super_secret_password")
	os.Setenv("API_KEY", "sk-1234567890abcdef")
	defer func() {
		os.Unsetenv("SECURE_PASSWORD")
		os.Unsetenv("API_KEY")
	}()

	config := &Config{
		Notifications: Notifications{
			Email: EmailConfig{
				Enabled:  true,
				Password: "${SECURE_PASSWORD}",
			},
			Webhook: WebhookConfig{
				Enabled: true,
				Headers: map[string]string{
					"X-API-Key": "${API_KEY}",
				},
			},
		},
	}

	err := config.ExpandEnvironmentVariables()
	require.NoError(t, err)

	// Verify expansion worked
	assert.Equal(t, "super_secret_password", config.Notifications.Email.Password)
	assert.Equal(t, "sk-1234567890abcdef", config.Notifications.Webhook.Headers["X-API-Key"])
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	assert.Equal(t, 10, config.Global.MaxWorkers)
	assert.Equal(t, 10*time.Second, config.Global.DefaultTimeout)
	assert.Equal(t, 30*time.Second, config.Global.DefaultInterval)
	assert.Equal(t, "./healthcheck.db", config.Global.StoragePath)
	assert.Equal(t, "info", config.Global.LogLevel)
	assert.False(t, config.Global.DisableColors)
	assert.Equal(t, "HealthCheck-CLI/1.0", config.Global.UserAgent)
	assert.Equal(t, 3, config.Global.MaxRetries)
	assert.Equal(t, 5*time.Second, config.Global.RetryDelay)

	// Test notification defaults
	assert.False(t, config.Notifications.GlobalRules.OnSuccess)
	assert.True(t, config.Notifications.GlobalRules.OnFailure)
	assert.True(t, config.Notifications.GlobalRules.OnRecovery)
	assert.True(t, config.Notifications.GlobalRules.OnSlowResponse)
	assert.Equal(t, 5*time.Minute, config.Notifications.GlobalRules.Cooldown)
	assert.Equal(t, 10, config.Notifications.GlobalRules.MaxAlerts)
	assert.Equal(t, 15*time.Minute, config.Notifications.GlobalRules.EscalationDelay)
}

func TestNestedEnvironmentVariables(t *testing.T) {
	// Test complex nested structure with environment variables
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "admin")
	os.Setenv("DB_PASS", "secret")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASS")
	}()

	config := &Config{
		Checks: []CheckConfig{
			{
				CheckConfig: types.CheckConfig{
					Name: "Database Check",
					URL:  "${DB_HOST}:${DB_PORT}",
					Headers: map[string]string{
						"User":     "${DB_USER}",
						"Password": "${DB_PASS}",
					},
				},
			},
		},
	}

	err := config.ExpandEnvironmentVariables()
	require.NoError(t, err)

	check := config.Checks[0]
	assert.Equal(t, "db.example.com:5432", check.URL)
	assert.Equal(t, "admin", check.Headers["User"])
	assert.Equal(t, "secret", check.Headers["Password"])
}