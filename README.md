# HealthCheck CLI

A powerful command-line tool for monitoring the health of your endpoints with support for HTTP and TCP checks, real-time notifications via email and Discord, and a beautiful terminal UI.

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- 🔍 **Multiple Check Types**
  - HTTP/HTTPS endpoints
  - TCP ports
  - Custom headers and body
  - Response validation
  - Response time monitoring

- 📊 **Real-time Monitoring**
  - Beautiful terminal UI
  - Status dashboard
  - Response time tracking
  - Historical data
  - Parallel execution with worker pool

- 🔔 **Smart Notifications**
  - Email notifications (SMTP)
  - Discord webhook integration
  - Configurable notification rules
  - Cooldown periods
  - Status-based alerts

- ⚙️ **Flexible Configuration**
  - YAML configuration files
  - Environment variable support
  - Default values with overrides
  - Multiple check profiles
  - Configurable worker pool

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

# Build the project
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

# Build and install
go build -o healthcheck cmd/healthcheck/main.go
sudo mv healthcheck /usr/local/bin/
```

Note: The `go install` command is not currently supported as the project needs to be built with CGO enabled for SQLite support.

## Quick Start

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
```

3. Start monitoring:
```bash
healthcheck monitor config.yml
```

## Configuration

### Basic Configuration

```yaml
global:
  max_workers: 10        # Maximum number of concurrent checks
  default_timeout: 10s
  default_interval: 30s
  storage_path: "./healthcheck.db"
  log_level: "info"

checks:
  - name: "API Health"
    type: "http"
    url: "https://api.example.com/health"
    interval: 30s
    timeout: 10s
    method: "GET"
    headers:
      Authorization: "Bearer ${API_TOKEN}"
    expected:
      status: 200
      body_contains: "healthy"
      response_time_max: 2s
```

### Email Notifications

```yaml
notifications:
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "your-email@gmail.com"
    password: "${EMAIL_PASSWORD}"  # Use app password for Gmail
    from: "your-email@gmail.com"
    to: ["recipient@example.com"]
    subject: "🚨 HealthCheck Alert: {{.Name}}"
    tls: true
```

### Discord Notifications

```yaml
notifications:
  discord:
    enabled: true
    webhook_url: "${DISCORD_WEBHOOK_URL}"
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

### Performance Optimization

The tool uses a worker pool to execute health checks in parallel, which significantly improves performance when monitoring multiple endpoints. You can configure the number of concurrent workers in the global settings:

```yaml
global:
  max_workers: 10  # Adjust based on your system's capabilities
```

Key benefits of parallel execution:
- Faster overall execution time
- Better resource utilization
- Configurable concurrency level
- Automatic load balancing
- Graceful error handling

## Usage

### Monitor Endpoints

```bash
# Monitor with configuration file
healthcheck monitor config.yml

# Run in daemon mode
healthcheck monitor config.yml --daemon

# Quick check single endpoint
healthcheck quick https://api.example.com/health

# Test endpoint immediately
healthcheck test https://api.example.com/health
```

### View Status

```bash
# Show status dashboard
healthcheck status

# Interactive dashboard
healthcheck status --watch

# With specific config
healthcheck status --config config.yml
```

### Statistics

```bash
# Show stats for all services
healthcheck stats

# Show stats for specific service
healthcheck stats "API Health"

# Show stats since duration
healthcheck stats --since 24h

# JSON output
healthcheck stats --json
```

## Environment Variables

The following environment variables can be used in the configuration:

- `${API_TOKEN}` - API authentication token
- `${EMAIL_PASSWORD}` - Email password
- `${DISCORD_WEBHOOK_URL}` - Discord webhook URL

## Commands

- `quick [URL]` - Quickly check a single endpoint
- `monitor [config-file]` - Monitor endpoints using a configuration file
- `test [URL]` - Test a single endpoint immediately
- `status` - Show status dashboard
- `config` - Configuration management
- `stats [service-name]` - Show statistics from stored data
- `history [service-name]` - Show historical data for a service
- `db-info` - Show database information
- `version` - Show version information

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run with coverage
make test-coverage

# Development mode
make dev

# Format code
make fmt

# Lint code
make lint
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver 