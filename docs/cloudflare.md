# Cloudflare DNS Provider

This guide explains how to configure caddy-dns-sync to work with Cloudflare DNS.

## Overview

Cloudflare provider creates and manages A records in your Cloudflare DNS zones via their API.

## Prerequisites

- A Cloudflare account
- A domain managed by Cloudflare
- API token with DNS edit permissions
- Zone ID for your domain

## Getting Credentials

### API Token

1. Log in to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Navigate to **My Profile** → **API Tokens**
3. Click **Create Token**
4. Use the **Edit zone DNS** template or create custom token with:
   - Permissions: `Zone - DNS - Edit` and `Zone - Zone - Read`
   - Zone Resources: Select specific zone(s)
5. Click **Continue to summary** → **Create Token**
6. Copy the token immediately (you won't see it again)

### Zone ID

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Select your domain
3. Scroll down on the **Overview** page
4. Find **Zone ID** in the API section (right sidebar)
5. Copy the Zone ID

## Configuration

Edit `/etc/caddy-dns-sync/provider.conf`:

```json
{
  "cloudflare": {
    "example.com": {
      "ip_interface": "eth0",
      "apitoken": "$(CLOUDFLARE_API_TOKEN_EXAMPLE)",
      "zoneid": "your-zone-id-here"
    },
    "another-domain.com": {
      "ip_interface": "eth0",
      "apitoken": "your-api-token-here",
      "zoneid": "another-zone-id-here"
    }
  }
}
```

### Configuration Fields

- **ip_interface** (required): Network interface to get IP address from (e.g., `eth0`, `ens3`, `wlan0`)
- **apitoken** (required): Cloudflare API token
- **zoneid** (required): Cloudflare Zone ID

### Environment Variable Substitution

You can use environment variables in the config file:

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

Then export the variables before running:

```bash
export CLOUDFLARE_API_TOKEN="your-token-here"
export CLOUDFLARE_ZONE_ID="your-zone-id-here"
```

## Initialization

Initialize the system with Cloudflare provider only:

```bash
sudo caddy-dns-sync --init --cloudflare
```

This creates:
- `/etc/caddy-dns-sync/` directory
- `provider.conf` with Cloudflare example configuration
- Systemd service and timer files

## How It Works

1. **Domain Extraction**: Parses Caddyfile to get all domain names
2. **IP Detection**: Gets IPv4 address from the specified network interface
3. **DNS Sync**: For each Cloudflare zone configured:
   - Creates A records for new domains
   - Updates A records if IP changed
   - Deletes A records for removed domains
4. **State Tracking**: Saves record IDs in `/etc/caddy-dns-sync/dns.state`

## Testing

Test manually before enabling the timer:

```bash
# Set environment variables
export CLOUDFLARE_API_TOKEN="your-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"

# Run sync
sudo -E caddy-dns-sync
```

## Troubleshooting

### "Failed to get IP from interface"

Check available interfaces:

```bash
ip addr show
```

Update the `ip_interface` in your config to match an existing interface.

### "Cloudflare API error: Invalid token"

- Verify token is correct
- Check token permissions include `Zone - DNS - Edit`
- Ensure token hasn't expired

### "Zone not found"

- Verify Zone ID is correct
- Ensure API token has access to the zone

## Security Best Practices

- Use API tokens (not API keys) with minimal required permissions
- Restrict tokens to specific zones
- Use environment variables for tokens instead of hardcoding
- Set config file permissions: `chmod 600 /etc/caddy-dns-sync/provider.conf`
- Regularly rotate API tokens

## Limitations

- Only supports A records (IPv4)
- Does not support Cloudflare proxy mode (creates unproxied records)
- TTL is set to "Auto" (cannot be customized per record)

## Multiple Zones

You can manage multiple Cloudflare zones in one configuration:

```json
{
  "cloudflare": {
    "domain1.com": {
      "ip_interface": "eth0",
      "apitoken": "$(TOKEN_DOMAIN1)",
      "zoneid": "zone-id-1"
    },
    "domain2.net": {
      "ip_interface": "eth0",
      "apitoken": "$(TOKEN_DOMAIN2)",
      "zoneid": "zone-id-2"
    }
  }
}
```

Each zone can use a different API token if needed.
