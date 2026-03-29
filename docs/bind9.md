# BIND9 DNS Provider

This guide explains how to configure caddy-dns-sync to work with BIND9 DNS server using RFC 2136 dynamic updates.

## Overview

BIND9 provider uses RFC 2136 protocol to dynamically update DNS records on your BIND9 server.

## Prerequisites

- BIND9 DNS server
- TSIG key for authentication
- Dynamic zone configuration in BIND9

## BIND9 Server Configuration

### 1. Generate TSIG Key

Generate a TSIG key for authentication:

```bash
tsig-keygen -a hmac-sha256 ddns-key > /etc/bind/ddns-key.conf
```

This creates a file like:

```
key "ddns-key" {
    algorithm hmac-sha256;
    secret "base64-encoded-secret-here==";
};
```

### 2. Configure BIND9 Zone

Edit your zone configuration (e.g., `/etc/bind/named.conf.local`):

```
include "/etc/bind/ddns-key.conf";

zone "example.com" {
    type master;
    file "/var/lib/bind/example.com.zone";
    allow-update { key ddns-key; };
};
```

### 3. Set Proper Permissions

```bash
chown bind:bind /var/lib/bind/example.com.zone
chmod 644 /var/lib/bind/example.com.zone
```

### 4. Restart BIND9

```bash
systemctl restart bind9
```

## Configuration

Edit `/etc/caddy-dns-sync/provider.conf`:

```json
{
  "bind9": {
    "example.com": {
      "ip_interface": "eth0",
      "server": "127.0.0.1",
      "port": 53,
      "key_name": "ddns-key",
      "key_secret": "$(BIND9_KEY_SECRET)",
      "algorithm": "hmac-sha256"
    }
  }
}
```

### Configuration Fields

- **ip_interface** (required): Network interface to get IP address from
- **server** (required): BIND9 server address
- **port** (optional): DNS server port (default: 53)
- **key_name** (required): TSIG key name
- **key_secret** (required): TSIG key secret (base64-encoded)
- **algorithm** (optional): TSIG algorithm (default: hmac-sha256)

Supported algorithms:
- `hmac-md5`
- `hmac-sha1`
- `hmac-sha256` (recommended)
- `hmac-sha512`

### Environment Variable Substitution

Store the key secret in an environment variable:

```bash
export BIND9_KEY_SECRET="your-base64-secret-here"
```

## Initialization

Initialize with BIND9 provider only:

```bash
sudo caddy-dns-sync --init --bind9
```

## How It Works

1. Parses Caddyfile to extract domains
2. Gets IP from specified network interface
3. For each BIND9 zone:
   - Sends RFC 2136 UPDATE messages to BIND9 server
   - Authenticates using TSIG key
   - Creates/updates/deletes A records

## Testing

Test the configuration:

```bash
# Set environment variable
export BIND9_KEY_SECRET="your-secret"

# Test manually
sudo -E caddy-dns-sync
```

Check BIND9 logs:

```bash
tail -f /var/log/syslog | grep named
```

## Troubleshooting

### "Connection refused"

- Verify BIND9 is running: `systemctl status bind9`
- Check firewall allows port 53
- Verify server address in config

### "Update failed: REFUSED"

- Check TSIG key is correct
- Verify key name matches BIND9 configuration
- Ensure zone allows updates from the key

### "Zone not found"

- Verify zone name in BIND9 configuration
- Check zone file exists
- Ensure BIND9 loaded the zone: `rndc reload`

## Security Best Practices

- Use strong TSIG keys (hmac-sha256 or hmac-sha512)
- Restrict `allow-update` to specific keys only
- Store key secrets in environment variables
- Set restrictive permissions on key files: `chmod 600 /etc/bind/ddns-key.conf`
- Use firewall to restrict access to port 53

## Advanced Configuration

### Multiple Zones

```json
{
  "bind9": {
    "example.com": {
      "ip_interface": "eth0",
      "server": "127.0.0.1",
      "port": 53,
      "key_name": "ddns-key-example",
      "key_secret": "$(BIND9_SECRET_EXAMPLE)",
      "algorithm": "hmac-sha256"
    },
    "another.net": {
      "ip_interface": "eth0",
      "server": "192.168.1.10",
      "port": 53,
      "key_name": "ddns-key-another",
      "key_secret": "$(BIND9_SECRET_ANOTHER)",
      "algorithm": "hmac-sha256"
    }
  }
}
```

### Remote BIND9 Server

If BIND9 runs on a different server:

```json
{
  "bind9": {
    "example.com": {
      "ip_interface": "eth0",
      "server": "192.168.1.5",
      "port": 53,
      "key_name": "ddns-key",
      "key_secret": "$(BIND9_KEY_SECRET)",
      "algorithm": "hmac-sha256"
    }
  }
}
```

Ensure the remote server accepts updates from your IP.

## Limitations

- Only supports A records (IPv4)
- Requires BIND9 zones to be configured for dynamic updates
- TTL is determined by zone defaults (cannot be set per record)

## BIND9 Logs

Enable query logging in BIND9 to debug:

```
logging {
    channel update_log {
        file "/var/log/bind/updates.log" versions 3 size 5m;
        severity info;
        print-category yes;
        print-severity yes;
        print-time yes;
    };
    category update { update_log; };
};
```

Restart BIND9 and check `/var/log/bind/updates.log` for update attempts.
