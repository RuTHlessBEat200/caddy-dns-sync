package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yourusername/caddy-dns-sync/pkg/config"
)

const (
	configDir   = "/etc/caddy-dns-sync"
	systemdDir  = "/etc/systemd/system"
	serviceName = "caddy-dns-sync.service"
	timerName   = "caddy-dns-sync.timer"
)

func initializeSystem(providers []string) error {
	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	fmt.Printf("Created config directory: %s\n", configDir)

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Create example provider config
	providerConfPath := filepath.Join(configDir, "provider.conf")
	if _, err := os.Stat(providerConfPath); os.IsNotExist(err) {
		exampleConfig := config.GenerateExampleConfig(providers)
		if err := config.SaveProviderConfig(providerConfPath, exampleConfig); err != nil {
			return fmt.Errorf("failed to create provider config: %w", err)
		}
		fmt.Printf("Created example provider config: %s\n", providerConfPath)
	} else {
		fmt.Printf("Provider config already exists: %s\n", providerConfPath)
	}

	// Create systemd service
	serviceContent := fmt.Sprintf(`[Unit]
Description=Caddy DNS Sync Service
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=caddy
Group=caddy
ExecStart=%s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=caddy-dns-sync

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/caddy-dns-sync
ReadOnlyPaths=/etc/caddy

[Install]
WantedBy=multi-user.target
`, execPath)

	servicePath := filepath.Join(systemdDir, serviceName)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create systemd service file: %w", err)
	}
	fmt.Printf("Created systemd service: %s\n", servicePath)

	// Create systemd timer
	timerContent := `[Unit]
Description=Run Caddy DNS Sync every 5 minutes
Requires=caddy-dns-sync.service

[Timer]
OnBootSec=1min
OnUnitActiveSec=5min
AccuracySec=1s
Persistent=true

[Install]
WantedBy=timers.target
`

	timerPath := filepath.Join(systemdDir, timerName)
	if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("failed to create systemd timer file: %w", err)
	}
	fmt.Printf("Created systemd timer: %s\n", timerPath)

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w\nOutput: %s", err, output)
	}
	fmt.Println("Reloaded systemd daemon")

	fmt.Println("\nSetup complete! Next steps:")
	fmt.Printf("1. Edit %s and configure your DNS provider(s)\n", providerConfPath)
	fmt.Println("\n2. Enable and start the timer:")
	fmt.Printf("   sudo systemctl enable %s\n", timerName)
	fmt.Printf("   sudo systemctl start %s\n", timerName)
	fmt.Println("\n3. Check status:")
	fmt.Printf("   sudo systemctl status %s\n", timerName)
	fmt.Printf("   sudo journalctl -u %s -f\n", serviceName)

	return nil
}
