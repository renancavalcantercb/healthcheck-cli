package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// DiscordNotifier handles Discord webhook notifications
type DiscordNotifier struct {
	config config.DiscordConfig
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(config config.DiscordConfig) *DiscordNotifier {
	log.Printf("📢 Initializing Discord notifier with configuration:")
	log.Printf("   Webhook URL: %s", maskWebhookURL(config.WebhookURL))
	log.Printf("   Username: %s", config.Username)
	log.Printf("   Avatar URL: %s", config.AvatarURL)
	return &DiscordNotifier{
		config: config,
	}
}

// Send sends a Discord notification
func (n *DiscordNotifier) Send(result types.Result) error {
	log.Printf("📢 Preparing Discord notification for %s (Status: %s)", result.Name, result.Status)

	// Prepare the message content
	content, err := n.prepareMessage(result)
	if err != nil {
		return fmt.Errorf("failed to prepare Discord message: %w", err)
	}

	log.Printf("📢 Discord message prepared:")
	log.Printf("   Content length: %d bytes", len(content))

	// Create HTTP request
	req, err := http.NewRequest("POST", n.config.WebhookURL, bytes.NewBuffer(content))
	if err != nil {
		return fmt.Errorf("failed to create Discord request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Discord API returned error status: %d", resp.StatusCode)
	}

	log.Printf("✅ Discord notification sent successfully")
	return nil
}

// prepareMessage prepares the Discord message content
func (n *DiscordNotifier) prepareMessage(result types.Result) ([]byte, error) {
	// Create embed color based on status
	var color int
	switch result.Status {
	case types.StatusUp:
		color = 0x00FF00 // Green
	case types.StatusDown:
		color = 0xFF0000 // Red
	case types.StatusSlow:
		color = 0xFFA500 // Orange
	default:
		color = 0x808080 // Gray
	}

	// Create the Discord message structure
	message := struct {
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		Embeds    []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Color       int    `json:"color"`
			Fields      []struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Inline bool   `json:"inline"`
			} `json:"fields"`
			Timestamp string `json:"timestamp"`
		} `json:"embeds"`
	}{
		Username:  n.config.Username,
		AvatarURL: n.config.AvatarURL,
		Embeds: []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Color       int    `json:"color"`
			Fields      []struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Inline bool   `json:"inline"`
			} `json:"fields"`
			Timestamp string `json:"timestamp"`
		}{
			{
				Title:       fmt.Sprintf("🚨 HealthCheck Alert: %s", result.Name),
				Description: fmt.Sprintf("Status: %s", result.Status),
				Color:       color,
				Fields: []struct {
					Name   string `json:"name"`
					Value  string `json:"value"`
					Inline bool   `json:"inline"`
				}{
					{
						Name:   "URL",
						Value:  result.URL,
						Inline: false,
					},
					{
						Name:   "Response Time",
						Value:  result.ResponseTime.String(),
						Inline: true,
					},
					{
						Name:   "Status Code",
						Value:  fmt.Sprintf("%d", result.StatusCode),
						Inline: true,
					},
				},
				Timestamp: result.Timestamp.Format(time.RFC3339),
			},
		},
	}

	// Add error message if present
	if result.Error != "" {
		message.Embeds[0].Fields = append(message.Embeds[0].Fields, struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Inline bool   `json:"inline"`
		}{
			Name:   "Error",
			Value:  result.Error,
			Inline: false,
		})
	}

	// Convert to JSON
	content, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	return content, nil
}

// maskWebhookURL masks the webhook URL for logging
func maskWebhookURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:10] + "..." + url[len(url)-10:]
} 