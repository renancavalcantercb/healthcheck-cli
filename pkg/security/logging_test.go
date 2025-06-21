package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "NormalEmail",
			email: "user@example.com",
			want:  "us***@example.com",
		},
		{
			name:  "ShortEmail",
			email: "ab@example.com",
			want:  "***@example.com",
		},
		{
			name:  "VeryShortEmail",
			email: "a@example.com",
			want:  "***@example.com",
		},
		{
			name:  "EmptyEmail",
			email: "",
			want:  "",
		},
		{
			name:  "InvalidEmail_NoAt",
			email: "invalidemaildotcom",
			want:  "***",
		},
		{
			name:  "InvalidEmail_MultipleAt",
			email: "user@domain@com",
			want:  "***",
		},
		{
			name:  "LongEmail",
			email: "verylongusername@domain.com",
			want:  "ve***@domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.email)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMaskEmailList(t *testing.T) {
	tests := []struct {
		name   string
		emails []string
		want   []string
	}{
		{
			name:   "MultipleEmails",
			emails: []string{"user1@example.com", "user2@domain.org", "short@test.com"},
			want:   []string{"us***@example.com", "us***@domain.org", "sh***@test.com"},
		},
		{
			name:   "EmptyList",
			emails: []string{},
			want:   []string{},
		},
		{
			name:   "SingleEmail",
			emails: []string{"test@example.com"},
			want:   []string{"te***@example.com"},
		},
		{
			name:   "MixedValidInvalid",
			emails: []string{"valid@example.com", "invalid-email", "another@test.org"},
			want:   []string{"va***@example.com", "***", "an***@test.org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmailList(tt.emails)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMaskURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "DiscordWebhook",
			url:  "https://discord.com/api/webhooks/1234567890/secret-token-here",
			want: "https://discord.com/***",
		},
		{
			name: "SlackWebhook",
			url:  "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			want: "https://hooks.slack.com/***",
		},
		{
			name: "APIEndpoint",
			url:  "https://api.example.com/v1/secret-endpoint",
			want: "https://api.example.com/***",
		},
		{
			name: "SimpleURL",
			url:  "https://example.com/path",
			want: "https://example.com/***",
		},
		{
			name: "URLWithoutPath",
			url:  "https://example.com",
			want: "https://example.com",
		},
		{
			name: "URLWithoutPath_WithSlash",
			url:  "https://example.com/",
			want: "https://example.com/",
		},
		{
			name: "EmptyURL",
			url:  "",
			want: "",
		},
		{
			name: "InvalidURL",
			url:  "not-a-url",
			want: "not-a-url",
		},
		{
			name: "URLWithQuery",
			url:  "https://api.example.com/endpoint?token=secret123",
			want: "https://api.example.com/***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskURL(tt.url)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSanitizeForLogs(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "PasswordInMessage",
			message: "Authentication failed with password=secret123",
			want:    "Authentication failed with password=***",
		},
		{
			name:    "TokenInMessage",
			message: "Request failed with token=abc123def456",
			want:    "Request failed with token=***",
		},
		{
			name:    "APIKeyInMessage",
			message: "Using api_key=myapikey for authentication",
			want:    "Using api_key=*** for authentication",
		},
		{
			name:    "SecretInMessage",
			message: "Configuration includes secret=topsecret",
			want:    "Configuration includes secret=***",
		},
		{
			name:    "WebhookInMessage",
			message: "Sending to webhook=https://hooks.example.com/secret",
			want:    "Sending to webhook=***",
		},
		{
			name:    "KeyInMessage",
			message: "Using key=mykey for encryption",
			want:    "Using key=*** for encryption",
		},
		{
			name:    "MultipleSecrets",
			message: "Config: password=secret token=abc123 api_key=key456",
			want:    "Config: password=*** token=*** api_key=***",
		},
		{
			name:    "NoSecrets",
			message: "Regular log message without sensitive data",
			want:    "Regular log message without sensitive data",
		},
		{
			name:    "CaseInsensitive",
			message: "Using PASSWORD=secret and TOKEN=abc123",
			want:    "Using password=*** and token=***",
		},
		{
			name:    "EmptyMessage",
			message: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLogs(tt.message)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMaskEmail_EdgeCases(t *testing.T) {
	// Test edge cases that might cause panics or unexpected behavior
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "EmailWithOnlyAt",
			email: "@",
			want:  "***@",
		},
		{
			name:  "EmailStartingWithAt",
			email: "@domain.com",
			want:  "***@domain.com",
		},
		{
			name:  "EmailEndingWithAt",
			email: "user@",
			want:  "us***@",
		},
		{
			name:  "EmailWithSpaces",
			email: "user @domain.com",
			want:  "us***@domain.com",
		},
		{
			name:  "EmailWithSpecialChars",
			email: "user+tag@domain.com",
			want:  "us***@domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := MaskEmail(tt.email)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMaskURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "URLWithMultipleSlashes",
			url:  "https://example.com//path//to//resource",
			want: "https://example.com/***",
		},
		{
			name: "URLWithFragment",
			url:  "https://example.com/path#fragment",
			want: "https://example.com/***",
		},
		{
			name: "URLWithPort",
			url:  "https://example.com:8080/api/endpoint",
			want: "https://example.com:8080/***",
		},
		{
			name: "FTPProtocol",
			url:  "ftp://example.com/file.txt",
			want: "ftp://example.com/***",
		},
		{
			name: "LocalhostURL",
			url:  "http://localhost:3000/api/secret",
			want: "http://localhost:3000/***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskURL(tt.url)
			assert.Equal(t, tt.want, result)
		})
	}
}