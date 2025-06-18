# HealthCheck CLI

A powerful command-line tool for monitoring the health of your HTTP and TCP endpoints with a beautiful terminal UI.

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- 🔍 Monitor HTTP and TCP endpoints
- 📊 Beautiful terminal UI dashboard
- 📈 Historical data tracking with SQLite storage
- 🔄 Configurable check intervals and timeouts
- 🔁 Automatic retry with exponential backoff
- 📝 YAML/JSON configuration support
- 📱 Multi-platform support (Linux, macOS, Windows)
- 🎨 Colored output with emoji status indicators
- 📊 Detailed statistics and history
- 🔔 Configurable notifications (Email, Slack, Discord, Telegram, Webhook)

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

### Quick Check

```bash
# Check a single endpoint
healthcheck quick https://api.example.com/health

# Run in daemon mode
healthcheck quick https://api.example.com/health --daemon

# Custom interval
healthcheck quick https://api.example.com/health --interval=1m
```

### Interactive Dashboard

```bash
# Start the interactive dashboard
healthcheck status --watch

# Use with configuration file
healthcheck status --watch --config=config.yaml
```

### Configuration File

Create a `config.yaml` file:

```yaml
global:
  max_workers: 20
  default_timeout: 10s
  default_interval: 30s
  storage_path: ./healthcheck.db
  log_level: info

checks:
  - name: API Health
    type: http
    url: https://api.example.com/health
    interval: 30s
    timeout: 10s
    method: GET
    headers:
      Authorization: Bearer ${API_TOKEN}
    expected:
      status: 200
      body_contains: healthy
      response_time_max: 2s
    retry:
      attempts: 3
      delay: 5s
      backoff: exponential

  - name: Database Connection
    type: tcp
    url: db.example.com:5432
    interval: 1m
    timeout: 5s
```

Run with configuration:

```bash
healthcheck monitor config.yaml
```

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

## Configuration

### Global Settings

- `max_workers`: Maximum number of concurrent checks
- `default_timeout`: Default timeout for checks
- `default_interval`: Default check interval
- `storage_path`: Path to SQLite database
- `log_level`: Logging level (debug, info, warn, error)
- `disable_colors`: Disable colored output
- `user_agent`: Custom User-Agent for HTTP checks
- `max_retries`: Default number of retry attempts
- `retry_delay`: Default delay between retries

### Check Configuration

- `name`: Unique name for the check
- `type`: Check type (http or tcp)
- `url`: Endpoint URL
- `interval`: Check interval
- `timeout`: Request timeout
- `method`: HTTP method (for HTTP checks)
- `headers`: HTTP headers
- `expected`: Expected response criteria
- `retry`: Retry configuration

### Notifications

The tool supports multiple notification channels:

- Email
- Slack
- Discord
- Telegram
- Generic Webhook

Configure notifications in your config file:

```yaml
notifications:
  email:
    enabled: true
    smtp_host: smtp.gmail.com
    smtp_port: 587
    username: your-email@gmail.com
    password: ${EMAIL_PASSWORD}
    from: your-email@gmail.com
    to: ["admin@example.com"]
    subject: "HealthCheck Alert: {{.Name}}"

  slack:
    enabled: true
    webhook_url: ${SLACK_WEBHOOK_URL}
    channel: "#alerts"
```

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