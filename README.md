# HealthCheck CLI

A powerful command-line tool for monitoring the health of your endpoints with support for HTTP and TCP checks, real-time notifications via email and Discord, and a beautiful terminal UI.

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## 🔥 Features

- 🔍 **Multiple Check Types**
  - HTTP/HTTPS endpoints with SSL validation
  - TCP ports connectivity
  - Custom headers and body
  - Response validation
  - Response time monitoring

- 📊 **Real-time Monitoring**
  - Beautiful terminal UI
  - Status dashboard
  - Response time tracking
  - Historical data with SQLite storage
  - Concurrent execution with service layer architecture

- 🔔 **Smart Notifications**
  - Email notifications (SMTP with TLS)
  - Discord webhook integration
  - Configurable notification rules
  - Cooldown periods and rate limiting
  - Status-based alerts with templates

- ⚙️ **Flexible Configuration**
  - YAML configuration files
  - Secure file permissions (0600)
  - Default values with overrides
  - Multiple check profiles
  - Service-oriented architecture

- 🔒 **Security Features**
  - SSRF protection with URL validation
  - HTTP header injection prevention
  - TLS enforcement for SMTP authentication
  - Secure file permissions
  - Sensitive data masking in logs

## Installation

### Quick Install (Recommended)

```bash
# Download and run the install script
curl -sSL https://raw.githubusercontent.com/renancavalcantercb/healthcheck-cli/main/install.sh | bash
```

### From Source

```bash
# Clone the repository
git clone https://github.com/renancavalcantercb/healthcheck-cli.git
cd healthcheck-cli

# Build the project (uses service layer architecture)
make build

# Install (optional)
make install
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/renancavalcantercb/healthcheck-cli.git
cd healthcheck-cli

# Install dependencies
make deps

# Build with proper package structure
go build -o healthcheck ./cmd/healthcheck
sudo mv healthcheck /usr/local/bin/
```

**Note:** The project requires CGO for SQLite support and uses a modern service layer architecture for better maintainability.

## 🚀 Quick Start

1. Generate an example configuration:
```bash
healthcheck config example config.yml
```

2. Edit the configuration file to add your endpoints:
```yaml
checks:
  - name: "API Health"
    type: "http"
    url: "https://api.example.com/health"
    interval: 30s
    timeout: 10s
    expected:
      status: 200
      body_contains: "healthy"
      response_time_max: 2s
```

3. Start monitoring:
```bash
healthcheck monitor config.yml
```

## 📋 Configuration

### Basic Configuration

```yaml
global:
  max_workers: 10        # Concurrent health checks
  default_timeout: 10s
  default_interval: 30s
  storage_path: "./healthcheck.db"  # SQLite database (secure permissions)
  log_level: "info"
  disable_colors: false
  user_agent: "HealthCheck-CLI/1.0"

checks:
  - name: "API Health"
    type: "http"
    url: "https://api.example.com/health"
    interval: 30s
    timeout: 10s
    method: "GET"
    headers:
      Authorization: "Bearer YOUR_TOKEN"  # Note: Use environment variables in production
    expected:
      status: 200
      body_contains: "healthy"
      response_time_max: 2s
    retry:
      attempts: 3
      delay: 2s
      backoff: "exponential"  # or "linear"
      max_delay: 30s
```

### 🔒 Secure Email Notifications

```yaml
notifications:
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"  # Use app password for Gmail
    from: "your-email@gmail.com"
    to: ["recipient@example.com"]
    subject: "🚨 HealthCheck Alert: {{.Name}}"
    tls: true  # Required for authentication (security enhancement)
```

**Security Notes:**
- TLS is **mandatory** when using SMTP authentication
- Sensitive information is masked in logs
- Use app passwords instead of account passwords

### Discord Notifications

```yaml
notifications:
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK"  # Masked in logs
    username: "HealthCheck Bot"
    avatar_url: "https://raw.githubusercontent.com/renancavalcantercb/healthcheck-cli/main/assets/logo.png"
```

### Notification Rules

```yaml
notifications:
  global_rules:
    on_success: true
    on_failure: true
    on_recovery: true
    on_slow_response: true
    cooldown: 5m
    max_alerts: 10
    escalation_delay: 15m
```

## 🔧 Usage

### Monitor Endpoints

```bash
# Monitor with configuration file
healthcheck monitor config.yml

# Run in daemon mode (background monitoring)
healthcheck monitor config.yml --daemon

# Quick check single endpoint
healthcheck quick https://api.example.com/health

# Quick check with custom interval
healthcheck quick https://api.example.com/health --interval 10s --daemon

# Test endpoint immediately
healthcheck test https://api.example.com/health --timeout 5s --verbose
```

### View Status

```bash
# Show status dashboard
healthcheck status

# Interactive dashboard (real-time)
healthcheck status --watch

# With specific config
healthcheck status --config config.yml
```

### Statistics & Analytics

```bash
# Show stats for all services
healthcheck stats

# Show stats for specific service
healthcheck stats "API Health"

# Show stats since duration
healthcheck stats --since 24h

# JSON output for integrations
healthcheck stats --json

# Show historical data
healthcheck history "API Health" --limit 100 --since 7d

# Database information
healthcheck db-info
```

### Configuration Management

```bash
# Validate configuration file
healthcheck config validate config.yml

# Generate example configuration
healthcheck config example
healthcheck config example custom-config.yml
```

## 🏗️ Architecture

The application uses a modern **service layer architecture** with the following components:

```
├── cmd/healthcheck/          # CLI interface
├── internal/
│   ├── app/                 # Application orchestration
│   ├── services/            # Business logic layer
│   │   ├── healthcheck.go   # Health check service
│   │   ├── stats.go         # Statistics service
│   │   └── config.go        # Configuration service
│   ├── checker/             # Health check implementations
│   ├── storage/             # Data persistence (SQLite)
│   ├── notifications/       # Notification providers
│   └── tui/                # Terminal UI
├── pkg/
│   ├── interfaces/          # Service interfaces
│   ├── types/              # Shared types
│   └── security/           # Security utilities
```

### Key Benefits:
- **Dependency Injection**: Easy testing and mocking
- **Interface-driven**: Pluggable components
- **Concurrent Safe**: Proper goroutine management
- **Context Propagation**: Cancellation and timeouts
- **Security First**: Input validation and secure defaults

## 🔒 Security Features

### Input Validation
- **URL Validation**: Prevents SSRF attacks
- **Header Validation**: Prevents injection attacks
- **Path Validation**: Prevents directory traversal

### Secure Communications
- **TLS Enforcement**: Required for SMTP authentication
- **Certificate Validation**: SSL/TLS certificates verified
- **Redirect Limits**: Maximum 5 redirects to prevent loops

### Data Protection
- **File Permissions**: Database and config files use 0600 permissions
- **Log Masking**: Sensitive data masked in logs
- **Secure Defaults**: Security-first configuration

## 📊 Commands Reference

| Command | Description | Example |
|---------|-------------|---------|
| `quick [URL]` | Quick endpoint check | `healthcheck quick https://api.com` |
| `monitor [config]` | Configuration-based monitoring | `healthcheck monitor config.yml` |
| `test [URL]` | Immediate endpoint test | `healthcheck test https://api.com` |
| `status` | Status dashboard | `healthcheck status --watch` |
| `stats [service]` | Show statistics | `healthcheck stats "API Health"` |
| `history [service]` | Historical data | `healthcheck history "API" --since 24h` |
| `config validate` | Validate configuration | `healthcheck config validate config.yml` |
| `config example` | Generate example config | `healthcheck config example` |
| `db-info` | Database information | `healthcheck db-info` |
| `version` | Show version | `healthcheck version` |

## 🛠️ Development

### Building

```bash
# Install dependencies
make deps

# Clean and build
make clean && make build

# Build for multiple platforms
make build-all
```

### Code Quality

```bash
# Format code
make fmt

# Tidy dependencies
make tidy

# Clean build artifacts
make clean
```

### Testing

```bash
# Note: Comprehensive test suite is planned for next release
# Current: Manual testing with real endpoints

# Quick test
make run

# Development with example
make dev
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Follow the service layer architecture patterns
4. Ensure security best practices
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Architecture Guidelines
- Use dependency injection for services
- Implement interfaces for testability
- Follow security-first principles
- Add proper error handling with context

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

## 🔮 Roadmap

- [ ] Comprehensive test coverage
- [ ] Environment variable support for secrets
- [ ] Circuit breaker patterns
- [ ] Structured logging
- [ ] Metrics and observability
- [ ] Plugin architecture
- [ ] Multi-storage backends