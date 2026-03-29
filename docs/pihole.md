# Pi-hole DNS Provider

This guide explains how to configure caddy-dns-sync to work with Pi-hole's custom DNS functionality using API v6.

## Overview

Pi-hole provider manages custom DNS records in Pi-hole using its modern API v6 authentication and DNS management endpoints.

## Prerequisites

- Pi-hole v6 or later installed and running
- Pi-hole web interface accessible (HTTP or HTTPS)
- Web interface password

## Getting Password

### Method 1: From Web Interface

Your Pi-hole web interface password is the same password you use to log in to the admin panel.

### Method 2: Set/Reset Password via Command Line

SSH into Pi-hole and run:

```bash
pihole -a -p
```

This will prompt you to set a new password.

## Configuration

Edit `/etc/caddy-dns-sync/provider.conf`:

```json
{
  "pihole": {
    "pihole-primary": {
      "ip_interface": "eth0",
      "server": "https://dns1.example.com",
      "password": "$(PIHOLE_PASSWORD)"
    },
    "pihole-secondary": {
      "ip_interface": "eth0",
      "server": "http://192.168.1.3",
      "password": "another-password-here"
    }
  }
}
```

### Configuration Fields

- **ip_interface** (required): Network interface to get IP address from
- **server** (required): Pi-hole web interface URL (include http:// or https://)
- **password** (required): Pi-hole web interface password (plain text)

### Environment Variable Substitution

Store the password securely:

```bash
export PIHOLE_PASSWORD="your-password-here"
```

## Initialization

Initialize with Pi-hole provider only:

```bash
sudo caddy-dns-sync --init --pihole
```

## How It Works

1. **Authentication**: Authenticates with Pi-hole API v6 using POST `/api/auth` to obtain a session ID (SID)
2. **IP Discovery**: Gets IP from specified network interface
3. **Record Management**: For each Pi-hole instance:
   - Looks up existing DNS records via GET `/api/config/dns`
   - Deletes old records if IP changed using DELETE `/api/config/dns/hosts/{entry}`
   - Creates new records using PUT `/api/config/dns/hosts/{entry}`
   - Session automatically re-authenticates if expired

## API v6 Endpoints Used

- `POST /api/auth` - Authenticate and obtain session ID
- `GET /api/config/dns` - List all custom DNS entries
- `PUT /api/config/dns/hosts/{entry}` - Add DNS entry (format: "IP domain.com")
- `DELETE /api/config/dns/hosts/{entry}` - Remove DNS entry

## Custom DNS in Pi-hole

Pi-hole stores custom DNS records in `/etc/pihole/custom.list` in the format:

```
192.168.1.100 example.com
192.168.1.100 www.example.com
```

caddy-dns-sync manages these entries automatically.

## Testing

Test the configuration:

```bash
# Set environment variable
export PIHOLE_API_TOKEN="your-token"

# Run sync
sudo -E caddy-dns-sync
```

Verify in Pi-hole:

```bash
# On Pi-hole server
cat /etc/pihole/custom.list
```

Or through the web interface:
- **Local DNS** → **DNS Records**

## Troubleshooting

### "Connection refused" or "Connection timeout"

- Verify Pi-hole is running: `pihole status`
- Check firewall allows HTTP/HTTPS access
- Verify server URL in config (include http:// or https://)

### "Invalid API token"

- Verify token is correct
- Try regenerating the token: `pihole -a -p`
- Check token hasn't been changed in Pi-hole settings

### "Records not appearing"

- Check Pi-hole logs: `pihole -t`
- Verify FTL is running: `systemctl status pihole-FTL`
- Restart Pi-hole: `pihole restartdns`

## Security Best Practices

- Use HTTPS for Pi-hole web interface when possible
- Store API tokens in environment variables
- Restrict network access to Pi-hole admin interface
- Use strong web password
- Set config file permissions: `chmod 600 /etc/caddy-dns-sync/provider.conf`

## Multiple Pi-hole Instances

You can sync to multiple Pi-hole servers for redundancy:

```json
{
  "pihole": {
    "pihole-1": {
      "ip_interface": "eth0",
      "server": "http://192.168.1.2",
      "apitoken": "$(PIHOLE_TOKEN_1)"
    },
    "pihole-2": {
      "ip_interface": "eth0",
      "server": "http://192.168.1.3",
      "apitoken": "$(PIHOLE_TOKEN_2)"
    }
  }
}
```

All Pi-hole instances will receive the same DNS records.

## Advanced Configuration

### Using HTTPS

If Pi-hole has SSL certificate:

```json
{
  "pihole": {
    "pihole-secure": {
      "ip_interface": "eth0",
      "server": "https://pihole.example.com",
      "apitoken": "$(PIHOLE_API_TOKEN)"
    }
  }
}
```

### Different IP per Pi-hole

If Pi-hole servers are in different networks:

```json
{
  "pihole": {
    "pihole-lan": {
      "ip_interface": "eth0",
      "server": "http://192.168.1.2",
      "apitoken": "$(PIHOLE_TOKEN_LAN)"
    },
    "pihole-wan": {
      "ip_interface": "eth1",
      "server": "http://10.0.0.2",
      "apitoken": "$(PIHOLE_TOKEN_WAN)"
    }
  }
}
```

## Limitations

- Only supports A records (IPv4)
- Pi-hole doesn't support record IDs (uses domain name as identifier)
- Updates may take a few seconds to propagate to FTL
- TTL is controlled by Pi-hole settings (cannot be set per record)

## Pi-hole API Reference

Pi-hole API endpoints used by caddy-dns-sync:

- `POST /admin/api.php?customdns=add` - Add custom DNS record
- `POST /admin/api.php?customdns=delete` - Delete custom DNS record

## Verifying DNS Records

Test DNS resolution from a client using Pi-hole:

```bash
dig @192.168.1.2 example.com

# Or
nslookup example.com 192.168.1.2
```

## Pi-hole Logs

Monitor Pi-hole logs for DNS queries:

```bash
# Live tail
pihole -t

# View FTL log
tail -f /var/log/pihole-FTL.log
```
