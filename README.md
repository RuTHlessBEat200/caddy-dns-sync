# Caddy DNS Sync

Automatically synchronize domain names from your Caddyfile to DNS providers. Supports Cloudflare, BIND9, and Pi-hole.

## Overview

caddy-dns-sync parses your Caddyfile, extracts all domain names, and automatically creates/updates/deletes DNS A records across multiple DNS providers. It runs as a systemd service and keeps your DNS records in sync with your Caddy configuration.

## Features

- **Multi-Provider Support**: Cloudflare, BIND9 (RFC 2136), and Pi-hole
- **Automatic IP Detection**: Gets IP from network interfaces (eth0, ens3, etc.)
- **Environment Variable Substitution**: Secure credential management with `$(ENV_VAR)` syntax
- **State Management**: Tracks records in `/etc/caddy-dns-sync/dns.state`
- **Diff-Based Sync**: Creates, updates, and deletes records as needed
- **Systemd Integration**: Runs automatically via timer
- **Multiple Zones**: Support for multiple domains and DNS zones per provider

## Supported Providers

| Provider | Method | Documentation |
|----------|--------|---------------|
| **Cloudflare** | REST API | [docs/cloudflare.md](docs/cloudflare.md) |
| **BIND9** | RFC 2136 (Dynamic DNS) | [docs/bind9.md](docs/bind9.md) |
| **Pi-hole** | Web API | [docs/pihole.md](docs/pihole.md) |

## Quick Start

### Installation

Download the latest release for your architecture:

```bash
# AMD64
wget https://github.com/yourusername/caddy-dns-sync/releases/latest/download/caddy-dns-sync-linux-amd64
chmod +x caddy-dns-sync-linux-amd64
sudo mv caddy-dns-sync-linux-amd64 /usr/local/bin/caddy-dns-sync

# ARM64
wget https://github.com/yourusername/caddy-dns-sync/releases/latest/download/caddy-dns-sync-linux-arm64
chmod +x caddy-dns-sync-linux-arm64
sudo mv caddy-dns-sync-linux-arm64 /usr/local/bin/caddy-dns-sync
```

### Initialize

Initialize for all providers:

```bash
sudo caddy-dns-sync --init
```

Or for specific providers:

```bash
# Cloudflare only
sudo caddy-dns-sync --init --cloudflare

# Multiple providers
sudo caddy-dns-sync --init --cloudflare --pihole
```

This creates:
- `/etc/caddy-dns-sync/provider.conf` - Provider configuration
- `/etc/systemd/system/caddy-dns-sync.service` - Systemd service
- `/etc/systemd/system/caddy-dns-sync.timer` - Systemd timer (runs every 5 minutes)

### Configure Providers

Edit `/etc/caddy-dns-sync/provider.conf`:

```json
{
  "cloudflare": {
    "example.com": {
      "ip_interface": "eth0",
      "apitoken": "$(CLOUDFLARE_API_TOKEN)",
      "zoneid": "$(CLOUDFLARE_ZONE_ID)"
    }
  },
  "bind9": {
    "internal.example.com": {
      "ip_interface": "eth0",
      "server": "127.0.0.1",
      "port": 53,
      "key_name": "ddns-key",
      "key_secret": "$(BIND9_KEY_SECRET)",
      "algorithm": "hmac-sha256"
    }
  },
  "pihole": {
    "pihole-1": {
      "ip_interface": "eth0",
      "server": "http://192.168.1.2",
      "apitoken": "$(PIHOLE_API_TOKEN)"
    }
  }
}
```

Set environment variables or use direct values in the config.

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable caddy-dns-sync.timer
sudo systemctl start caddy-dns-sync.timer
```

### Monitor

```bash
# Check timer status
sudo systemctl status caddy-dns-sync.timer

# View logs
sudo journalctl -u caddy-dns-sync.service -f

# Check state
sudo cat /etc/caddy-dns-sync/dns.state
```

## Configuration

### Provider Configuration File

Location: `/etc/caddy-dns-sync/provider.conf`

The configuration uses JSON format with support for environment variable substitution.

#### Environment Variable Substitution

Use `$(ENV_VAR_NAME)` syntax to reference environment variables:

```json
{
  "cloudflare": {
    "example.com": {
      "ip_interface": "eth0",
      "apitoken": "$(CLOUDFLARE_API_TOKEN)",
      "zoneid": "$(CLOUDFLARE_ZONE_ID)"
    }
  }
}
```

Variables are expanded at runtime from the environment.

#### IP Interface Detection

The `ip_interface` field specifies which network interface to get the IP address from:

```bash
# List available interfaces
ip addr show

# Common interface names
# eth0, ens3, ens5 - Ethernet
# wlan0 - WiFi
# enp0s3 - Predictable network interface names
```

### State File

Location: `/etc/caddy-dns-sync/dns.state`

Tracks all managed DNS records in JSON format:

```json
{
  "records": {
    "cloudflare:example.com:subdomain.example.com": {
      "domain": "subdomain.example.com",
      "record_id": "cloudflare-record-id",
      "ip_address": "203.0.113.10",
      "provider": "cloudflare",
      "zone": "example.com",
      "created_at": "2026-03-29T10:00:00Z",
      "updated_at": "2026-03-29T10:00:00Z"
    }
  },
  "updated_at": "2026-03-29T10:00:00Z"
}
```

### Command-Line Options

```
Usage: caddy-dns-sync [options]

Options:
  -init
        Initialize directories and systemd service
  -cloudflare
        Initialize Cloudflare provider (use with -init)
  -bind9
        Initialize BIND9 provider (use with -init)
  -pihole
        Initialize Pi-hole provider (use with -init)
  -version
        Show version information
  -caddyfile string
        Path to Caddyfile (default "/etc/caddy/Caddyfile")
  -config string
        Path to provider configuration file (default "/etc/caddy-dns-sync/provider.conf")
  -state string
        Path to DNS state file (default "/etc/caddy-dns-sync/dns.state")
```

## How It Works

1. **Parse Caddyfile**: Extracts all domain names from server blocks
2. **Load Configuration**: Reads provider settings from `provider.conf`
3. **Get IP Addresses**: Retrieves IPv4 from configured network interfaces
4. **Load State**: Reads current DNS records from `dns.state`
5. **Sync Records**: For each provider:
   - Creates DNS records for new domains
   - Updates records if IP changed
   - Deletes records for removed domains
6. **Save State**: Updates state file with current records

## Provider Setup Guides

Detailed configuration guides for each provider:

- [Cloudflare DNS](docs/cloudflare.md) - API setup, zone configuration, troubleshooting
- [BIND9 DNS](docs/bind9.md) - TSIG keys, RFC 2136 dynamic updates, zone setup
- [Pi-hole DNS](docs/pihole.md) - API tokens, custom DNS records, multiple instances

## Security

### File Permissions

```bash
# Config contains sensitive credentials
sudo chmod 600 /etc/caddy-dns-sync/provider.conf

# State file contains record IDs
sudo chmod 644 /etc/caddy-dns-sync/dns.state

# Directory
sudo chmod 755 /etc/caddy-dns-sync
```

### Service Hardening

The systemd service includes security hardening:

- Runs as `caddy` user (non-root)
- `NoNewPrivileges=true`
- `PrivateTmp=true`
- `ProtectSystem=strict`
- `ProtectHome=true`
- Read-write access only to `/etc/caddy-dns-sync`
- Read-only access to `/etc/caddy`

### Credential Management

Best practices:

- Use environment variables for sensitive data
- Never commit credentials to version control
- Use provider-specific tokens with minimal permissions
- Regularly rotate API tokens and keys
- Restrict DNS provider access to specific zones/domains

## Building from Source

### Prerequisites

- Go 1.21 or later

### Build

```bash
git clone https://github.com/yourusername/caddy-dns-sync.git
cd caddy-dns-sync
go build -o caddy-dns-sync
```

### Using Makefile

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install

# Initialize system
make init
```

## Troubleshooting

### Service Not Running

```bash
# Check service status
sudo systemctl status caddy-dns-sync.service

# Check timer
sudo systemctl status caddy-dns-sync.timer

# View recent logs
sudo journalctl -u caddy-dns-sync.service -n 100
```

### Permission Issues

```bash
# Ensure caddy user can read Caddyfile
sudo chmod 644 /etc/caddy/Caddyfile

# Check config directory permissions
ls -la /etc/caddy-dns-sync/
```

### No Domains Found

```bash
# Test Caddyfile parsing
sudo -u caddy caddy-dns-sync -caddyfile /etc/caddy/Caddyfile
```

### IP Detection Fails

```bash
# List network interfaces
ip addr show

# Update ip_interface in provider.conf to match an existing interface
```

### Provider-Specific Issues

See provider documentation:
- [Cloudflare troubleshooting](docs/cloudflare.md#troubleshooting)
- [BIND9 troubleshooting](docs/bind9.md#troubleshooting)
- [Pi-hole troubleshooting](docs/pihole.md#troubleshooting)

## Systemd Service Management

```bash
# Enable timer (start on boot)
sudo systemctl enable caddy-dns-sync.timer

# Start timer
sudo systemctl start caddy-dns-sync.timer

# Stop timer
sudo systemctl stop caddy-dns-sync.timer

# Run service immediately
sudo systemctl start caddy-dns-sync.service

# View timer schedule
systemctl list-timers | grep caddy-dns-sync

# Change interval (edit timer file)
sudo nano /etc/systemd/system/caddy-dns-sync.timer
sudo systemctl daemon-reload
sudo systemctl restart caddy-dns-sync.timer
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

- GitHub Issues: Report bugs and request features
- Documentation: Check provider-specific docs in `docs/` directory
- Logs: Review systemd journal for detailed information

## Changelog

### Version 1.0.0

- Multi-provider support (Cloudflare, BIND9, Pi-hole)
- Environment variable substitution in config
- Network interface IP detection
- State-based record tracking
- Systemd integration with timer
- Comprehensive documentation
