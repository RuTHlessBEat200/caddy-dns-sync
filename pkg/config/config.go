package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type ProviderConfig struct {
	Cloudflare map[string]CloudflareZoneConfig `json:"cloudflare,omitempty"`
	Bind9      map[string]Bind9ZoneConfig      `json:"bind9,omitempty"`
	Pihole     map[string]PiholeConfig         `json:"pihole,omitempty"`
}

type CloudflareZoneConfig struct {
	IPInterface string `json:"ip_interface"`
	APIToken    string `json:"apitoken"`
	ZoneID      string `json:"zoneid"`
}

type Bind9ZoneConfig struct {
	IPInterface string `json:"ip_interface"`
	Server      string `json:"server"`
	Port        int    `json:"port,omitempty"`
	KeyName     string `json:"key_name"`
	KeySecret   string `json:"key_secret"`
	Algorithm   string `json:"algorithm,omitempty"`
}

type PiholeConfig struct {
	IPInterface string `json:"ip_interface"`
	Server      string `json:"server"`
	Password    string `json:"password"` // Pi-hole web interface password
}

func LoadProviderConfig(path string) (*ProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables in the config
	expandedData := expandEnvVars(string(data))

	var config ProviderConfig
	if err := json.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveProviderConfig(path string, config *ProviderConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600) // More restrictive permissions for credentials
}

// expandEnvVars replaces $(ENV_VAR_NAME) with the actual environment variable value
func expandEnvVars(input string) string {
	// Match $(VAR_NAME) pattern
	re := regexp.MustCompile(`\$\(([A-Z_][A-Z0-9_]*)\)`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "$("), ")")

		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			fmt.Fprintf(os.Stderr, "Warning: Environment variable %s is not set\n", varName)
			return match // Keep original if not found
		}

		return value
	})
}

func GenerateExampleConfig(providers []string) *ProviderConfig {
	config := &ProviderConfig{}

	for _, provider := range providers {
		switch provider {
		case "cloudflare":
			config.Cloudflare = map[string]CloudflareZoneConfig{
				"example.com": {
					IPInterface: "eth0",
					APIToken:    "$(CLOUDFLARE_API_TOKEN)",
					ZoneID:      "$(CLOUDFLARE_ZONE_ID)",
				},
				"example.org": {
					IPInterface: "eth0",
					APIToken:    "your-api-token-here",
					ZoneID:      "your-zone-id-here",
				},
			}

		case "bind9":
			config.Bind9 = map[string]Bind9ZoneConfig{
				"example.com": {
					IPInterface: "eth0",
					Server:      "127.0.0.1",
					Port:        53,
					KeyName:     "ddns-key",
					KeySecret:   "$(BIND9_KEY_SECRET)",
					Algorithm:   "hmac-sha256",
				},
			}

		case "pihole":
			config.Pihole = map[string]PiholeConfig{
				"pihole-server-1": {
					IPInterface: "eth0",
					Server:      "https://dns1.example.com",
					Password:    "$(PIHOLE_PASSWORD)",
				},
			}
		}
	}

	return config
}
