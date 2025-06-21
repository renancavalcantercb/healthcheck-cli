# HealthCheck CLI

A powerful command-line tool for monitoring the health of your endpoints with support for HTTP, TCP, and SSL certificate checks, real-time notifications via email and Discord, and a beautiful terminal UI.

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- 🔍 **Multiple Check Types**
  - HTTP/HTTPS endpoints
  - TCP ports
  - SSL certificate monitoring
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

The script will:
1. **Try to download** the latest release binary (fastest, no dependencies)
2. **Fallback to build** from source if release download fails (requires Go + Git)

Installation options:
- **System-wide** (`/usr/local/bin`) - requires sudo
- **User-local** (`~/.local/bin`) - no sudo required, auto-adds to PATH

### Download Release (Manual)

Download the latest release for your platform from the [releases page](https://github.com/renancavalcantercb/healthcheck-cli/releases):

```bash
# Example for Linux AMD64
wget https://github.com/renancavalcantercb/healthcheck-cli/releases/latest/download/healthcheck-linux-amd64.tar.gz
tar -xzf healthcheck-linux-amd64.tar.gz
sudo mv healthcheck-linux-amd64 /usr/local/bin/healthcheck
chmod +x /usr/local/bin/healthcheck
```

**Supported Platforms:**
- Linux: `amd64`, `arm64`
- macOS: `amd64` (Intel), `arm64` (Apple Silicon)
- Windows: `amd64`

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

# Build with CGO enabled for SQLite support
CGO_ENABLED=1 go build -o healthcheck cmd/healthcheck/*.go

# Install (choose one):
# Option 1: System-wide (requires sudo)
sudo mv healthcheck /usr/local/bin/

# Option 2: User-local (no sudo required)
mkdir -p ~/.local/bin
mv healthcheck ~/.local/bin/
export PATH="$PATH:$HOME/.local/bin"  # Add to your shell RC file
```

**Note**: The project requires CGO to be enabled for SQLite support. If you get SQLite-related errors, the binary will automatically fall back to file-based storage.

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
  
  - name: "SSL Certificate Check"
    type: "ssl"
    url: "https://api.example.com"
    interval: 24h
    timeout: 10s
    expected:
      cert_expiry_days: 30
      cert_valid_domains: ["api.example.com"]
```

3. Start monitoring:
```bash
healthcheck monitor config.yml
```

## Check Types

HealthCheck CLI supports three types of monitoring:

### 1. HTTP/HTTPS Checks (`type: http`)
Monitor web endpoints, APIs, and web services:
- **Status code validation** (200, 404, etc.)
- **Response body content** (contains/not contains text)
- **Response time monitoring**
- **Custom headers and request body**
- **Content-Type validation**
- **Minimum body size checks**

### 2. TCP Port Checks (`type: tcp`)
Monitor TCP services and port connectivity:
- **Port connectivity testing**
- **Connection time monitoring**
- **Network service availability**

### 3. SSL Certificate Checks (`type: ssl`)
Monitor SSL/TLS certificate health:
- **Certificate expiration monitoring**
- **Domain validation (CN + SAN)**
- **SSL handshake performance**
- **Certificate chain validation**
- **Issuer and subject information**

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

  - name: "Database TCP Check"
    type: "tcp"
    url: "db.example.com:5432"
    interval: 60s
    timeout: 5s
    expected:
      response_time_max: 500ms

  - name: "SSL Certificate Monitor"
    type: "ssl"
    url: "https://api.example.com"
    interval: 24h
    timeout: 10s
    expected:
      cert_expiry_days: 30      # Alert if expires within 30 days
      cert_valid_domains: ["api.example.com", "www.api.example.com"]
      response_time_max: 5s
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

### SSL Certificate Monitoring

Monitor SSL certificates for expiration and validity:

```yaml
checks:
  - name: "Production API SSL"
    type: "ssl"
    url: "https://api.production.com"
    interval: 12h                    # Check twice daily
    timeout: 10s
    expected:
      cert_expiry_days: 14          # Alert if expires within 14 days
      cert_valid_domains:           # Verify certificate covers these domains
        - "api.production.com"
        - "www.api.production.com"
      response_time_max: 3s         # Alert if SSL handshake is slow
    tags: ["ssl", "production", "critical"]

  # Alternative host:port format
  - name: "Database SSL"
    type: "ssl"
    url: "db.internal.com:5432"
    interval: 24h
    timeout: 5s
    expected:
      cert_expiry_days: 30
      cert_valid_domains: ["db.internal.com"]
      response_time_max: 2s
    tags: ["ssl", "database"]
```

**SSL Check Features:**
- **Certificate expiration monitoring** - Get alerts before certificates expire
- **Domain validation** - Ensure certificates cover expected domains (CN + SAN)
- **SSL handshake performance** - Monitor connection establishment time
- **Detailed certificate info** - Subject, issuer, expiry date, and valid domains
- **Flexible URL formats** - Support both `https://domain.com` and `domain.com:443`

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

# Test SSL certificate
healthcheck test https://github.com  # Will use HTTP checker
# For dedicated SSL certificate testing, use configuration files
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

# Build for current platform
make build

# Build for all platforms (requires cross-compilation setup)
make build-all

# Build release archive for current platform
make build-releases

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

### Creating Releases

For maintainers, to create a new release:

```bash
# Use the release helper script
./release.sh v1.0.0

# Or manually:
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This will trigger GitHub Actions to automatically build cross-platform binaries and create a GitHub release.

## Troubleshooting

### Installation Issues

**Problem**: `mv: No such file or directory` during installation
- **Solution**: The binary build failed. Check that Go 1.21+ is installed and try building manually

**Problem**: `SQLite requires CGO but binary was compiled with CGO_ENABLED=0`
- **Solution**: This is expected when using pre-built release binaries. The tool will automatically fall back to JSON file storage with the same functionality

**Problem**: `command not found: healthcheck` after installation
- **Solution**: If installed locally, restart your terminal or run `source ~/.zshrc` (or your shell's RC file)

### Common Configuration Issues

**Problem**: SSL checks showing validation errors for `host:port` format
- **Solution**: Ensure the hostname resolves properly. Use `https://domain.com` format if needed

**Problem**: Notifications not working
- **Solution**: Check that environment variables are set correctly and notification services are enabled

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