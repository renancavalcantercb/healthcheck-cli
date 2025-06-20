package notifications

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"text/template"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/security"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// EmailNotifier handles email notifications
type EmailNotifier struct {
	config config.EmailConfig
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(config config.EmailConfig) *EmailNotifier {
	log.Printf("📧 Inicializando notificador de email com configuração:")
	log.Printf("   Host: %s", config.SMTPHost)
	log.Printf("   Porta: %d", config.SMTPPort)
	log.Printf("   From: %s", security.MaskEmail(config.From))
	log.Printf("   To: %v", security.MaskEmailList(config.To))
	log.Printf("   TLS: %v", config.TLS)
	return &EmailNotifier{
		config: config,
	}
}

// Send sends an email notification
func (n *EmailNotifier) Send(result types.Result) error {
	if !n.config.Enabled {
		log.Printf("📧 Notificações de email desabilitadas")
		return nil
	}

	log.Printf("📧 Preparando email para %s (Status: %s)", result.Name, result.Status)

	// Prepare email content
	subject := n.renderTemplate(n.config.Subject, result)
	body := n.generateEmailBody(result)

	log.Printf("📧 Conteúdo do email preparado:")
	log.Printf("   Assunto: %s", subject)
	log.Printf("   Tamanho do corpo: %d bytes", len(body))

	// Prepare message
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s",
		n.config.From,
		strings.Join(n.config.To, ","),
		subject,
		body,
	)

	// Send email
	addr := fmt.Sprintf("%s:%d", n.config.SMTPHost, n.config.SMTPPort)
	log.Printf("📧 Tentando enviar email via %s", addr)
	
	var auth smtp.Auth
	if n.config.Username != "" && n.config.Password != "" {
		// Enforce TLS when using authentication for security
		if !n.config.TLS {
			return fmt.Errorf("TLS is required when using SMTP authentication to protect credentials")
		}
		auth = smtp.PlainAuth("", n.config.Username, n.config.Password, n.config.SMTPHost)
		log.Printf("📧 Usando autenticação SMTP com TLS")
	} else {
		log.Printf("📧 Sem autenticação SMTP")
	}

	// For non-TLS connections (only allowed without authentication)
	if !n.config.TLS {
		if auth != nil {
			return fmt.Errorf("TLS is required when using authentication")
		}
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			log.Printf("❌ Erro ao dividir host/porta: %v", err)
			return fmt.Errorf("invalid SMTP address: %w", err)
		}
		log.Printf("📧 Tentando conexão SMTP sem TLS para %s (sem autenticação)", host)
		err = smtp.SendMail(addr, auth, host, n.config.To, []byte(msg))
		if err != nil {
			log.Printf("❌ Erro ao enviar email: %v", err)
			return err
		}
		log.Printf("✅ Email enviado com sucesso!")
		return nil
	}

	// For TLS connections
	log.Printf("📧 Tentando conexão SMTP com TLS")
	err := smtp.SendMail(addr, auth, n.config.From, n.config.To, []byte(msg))
	if err != nil {
		log.Printf("❌ Erro ao enviar email: %v", err)
		return err
	}
	log.Printf("✅ Email enviado com sucesso!")
	return nil
}

// renderTemplate renders a template with the result data
func (n *EmailNotifier) renderTemplate(tmpl string, result types.Result) string {
	if tmpl == "" {
		return fmt.Sprintf("HealthCheck Alert: %s", result.Name)
	}

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("HealthCheck Alert: %s", result.Name)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, result); err != nil {
		return fmt.Sprintf("HealthCheck Alert: %s", result.Name)
	}

	return buf.String()
}

// generateEmailBody generates the HTML email body
func (n *EmailNotifier) generateEmailBody(result types.Result) string {
	statusEmoji := result.Status.Emoji()
	statusText := result.Status.String()
	timestamp := result.Timestamp.Format("2006-01-02 15:04:05")

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #f8f9fa; padding: 20px; border-radius: 5px; }
				.details { margin-top: 20px; }
				.status { font-size: 24px; font-weight: bold; }
				.status.up { color: #28a745; }
				.status.down { color: #dc3545; }
				.status.slow { color: #ffc107; }
				.metric { margin: 10px 0; }
				.footer { margin-top: 20px; font-size: 12px; color: #6c757d; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>%s %s</h1>
					<p>Service: %s</p>
					<p>Time: %s</p>
				</div>
				<div class="details">
					<div class="metric">
						<strong>Status:</strong> <span class="status %s">%s %s</span>
					</div>
					<div class="metric">
						<strong>Response Time:</strong> %v
					</div>
					<div class="metric">
						<strong>URL:</strong> %s
					</div>
					%s
				</div>
				<div class="footer">
					<p>This is an automated message from HealthCheck CLI</p>
				</div>
			</div>
		</body>
		</html>
	`,
		statusEmoji,
		result.Name,
		result.Name,
		timestamp,
		strings.ToLower(statusText),
		statusEmoji,
		statusText,
		result.ResponseTime,
		result.URL,
		n.generateAdditionalDetails(result),
	)

	return html
}

// generateAdditionalDetails generates additional details based on the result
func (n *EmailNotifier) generateAdditionalDetails(result types.Result) string {
	var details strings.Builder

	if result.StatusCode > 0 {
		details.WriteString(fmt.Sprintf(`
			<div class="metric">
				<strong>HTTP Status:</strong> %d
			</div>
		`, result.StatusCode))
	}

	if result.Error != "" {
		details.WriteString(fmt.Sprintf(`
			<div class="metric">
				<strong>Error:</strong> %s
			</div>
		`, result.Error))
	}

	if result.BodySize > 0 {
		details.WriteString(fmt.Sprintf(`
			<div class="metric">
				<strong>Body Size:</strong> %d bytes
			</div>
		`, result.BodySize))
	}

	if len(result.Headers) > 0 {
		details.WriteString(`
			<div class="metric">
				<strong>Headers:</strong>
				<ul>
		`)
		for key, value := range result.Headers {
			details.WriteString(fmt.Sprintf(`
					<li><strong>%s:</strong> %s</li>
			`, key, value))
		}
		details.WriteString(`
				</ul>
			</div>
		`)
	}

	return details.String()
} 