package notifications

import (
	"log"
	"sync"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// Manager handles all notification channels
type Manager struct {
	emailNotifier *EmailNotifier
	config        *config.Config
	lastSent      map[string]time.Time
	mu            sync.RWMutex
}

// NewManager creates a new notification manager
func NewManager(config *config.Config) *Manager {
	log.Printf("📢 Inicializando gerenciador de notificações")
	
	manager := &Manager{
		config:   config,
		lastSent: make(map[string]time.Time),
	}

	if config.Notifications.Email.Enabled {
		log.Printf("📢 Configurando notificador de email")
		manager.emailNotifier = NewEmailNotifier(config.Notifications.Email)
	} else {
		log.Printf("📢 Notificações de email desabilitadas na configuração")
	}

	return manager
}

// Notify sends notifications based on the check result
func (m *Manager) Notify(result types.Result) error {
	log.Printf("📢 Processando notificação para %s (Status: %s)", result.Name, result.Status)
	
	// Check if we should send notifications based on rules
	if !m.shouldNotify(result) {
		log.Printf("📢 Notificação ignorada (regras de notificação)")
		return nil
	}

	// Update last sent time
	m.mu.Lock()
	m.lastSent[result.Name] = time.Now()
	m.mu.Unlock()

	// Send email notification if enabled
	if m.emailNotifier != nil {
		log.Printf("📢 Enviando notificação por email")
		if err := m.emailNotifier.Send(result); err != nil {
			log.Printf("❌ Erro ao enviar notificação por email: %v", err)
			return err
		}
	} else {
		log.Printf("📢 Notificador de email não disponível")
	}

	return nil
}

// shouldNotify determines if a notification should be sent based on rules
func (m *Manager) shouldNotify(result types.Result) bool {
	rules := m.config.Notifications.GlobalRules

	// Check cooldown period
	m.mu.RLock()
	lastSent, exists := m.lastSent[result.Name]
	m.mu.RUnlock()

	if exists {
		if time.Since(lastSent) < rules.Cooldown {
			log.Printf("📢 Notificação ignorada (cooldown)")
			return false
		}
	}

	// Check notification rules
	switch result.Status {
	case types.StatusUp:
		return rules.OnSuccess
	case types.StatusDown:
		return rules.OnFailure
	case types.StatusSlow:
		return rules.OnSlowResponse
	case types.StatusError:
		return rules.OnFailure
	case types.StatusWarning:
		return rules.OnFailure
	default:
		return false
	}
} 