package notifications

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// Manager handles all notification channels
type Manager struct {
	config           *config.Config
	emailNotifier    *EmailNotifier
	discordNotifier  *DiscordNotifier
	lastNotification map[string]time.Time
	mu               sync.RWMutex
}

// NewManager creates a new notification manager
func NewManager(config *config.Config) *Manager {
	log.Printf("📢 Initializing notification manager")
	
	manager := &Manager{
		config:           config,
		lastNotification: make(map[string]time.Time),
	}

	// Initialize email notifier if enabled
	if config.Notifications.Email.Enabled {
		log.Printf("📢 Configuring email notifier")
		manager.emailNotifier = NewEmailNotifier(config.Notifications.Email)
	} else {
		log.Printf("📢 Email notifications disabled in configuration")
	}

	// Initialize Discord notifier if enabled
	if config.Notifications.Discord.Enabled {
		log.Printf("📢 Configuring Discord notifier")
		manager.discordNotifier = NewDiscordNotifier(config.Notifications.Discord)
	} else {
		log.Printf("📢 Discord notifications disabled in configuration")
	}

	return manager
}

// Notify sends notifications based on the check result
func (m *Manager) Notify(result types.Result) error {
	log.Printf("📢 Processing notification for %s (Status: %s)", result.Name, result.Status)
	
	// Check if we should notify based on rules
	if !m.shouldNotify(result) {
		log.Printf("📢 Notification ignored (notification rules)")
		return nil
	}

	// Check cooldown
	if !m.checkCooldown(result.Name) {
		log.Printf("📢 Notification ignored (cooldown)")
		return nil
	}

	// Send email notification if enabled
	if m.emailNotifier != nil {
		log.Printf("📢 Sending email notification")
		if err := m.emailNotifier.Send(result); err != nil {
			log.Printf("❌ Error sending email notification: %v", err)
			return fmt.Errorf("error sending email notification: %w", err)
		}
	}

	// Send Discord notification if enabled
	if m.discordNotifier != nil {
		log.Printf("📢 Sending Discord notification")
		if err := m.discordNotifier.Send(result); err != nil {
			log.Printf("❌ Error sending Discord notification: %v", err)
			return fmt.Errorf("error sending Discord notification: %w", err)
		}
	}

	// Update last notification time
	m.mu.Lock()
	m.lastNotification[result.Name] = time.Now()
	m.mu.Unlock()

	return nil
}

// shouldNotify determines if a notification should be sent based on rules
func (m *Manager) shouldNotify(result types.Result) bool {
	rules := m.config.Notifications.GlobalRules

	switch result.Status {
	case types.StatusUp:
		return rules.OnSuccess
	case types.StatusDown:
		return rules.OnFailure
	case types.StatusSlow:
		return rules.OnSlowResponse
	default:
		return false
	}
}

// checkCooldown checks if enough time has passed since the last notification
func (m *Manager) checkCooldown(name string) bool {
	m.mu.RLock()
	lastTime, exists := m.lastNotification[name]
	m.mu.RUnlock()
	
	if !exists {
		return true
	}

	cooldown := m.config.Notifications.GlobalRules.Cooldown
	return time.Since(lastTime) >= cooldown
}

// UpdateConfig updates the manager configuration and returns the manager
func (m *Manager) UpdateConfig(config *config.Config) *Manager {
	m.config = config
	
	// Reinitialize notifiers if needed
	if config.Notifications.Email.Enabled {
		m.emailNotifier = NewEmailNotifier(config.Notifications.Email)
	} else {
		m.emailNotifier = nil
	}
	
	if config.Notifications.Discord.Enabled {
		m.discordNotifier = NewDiscordNotifier(config.Notifications.Discord)
	} else {
		m.discordNotifier = nil
	}
	
	return m
} 