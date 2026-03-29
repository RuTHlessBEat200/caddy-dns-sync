package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PiholeProvider handles DNS updates via Pi-hole's API v6
type PiholeProvider struct {
	server   string
	password string
	client   *http.Client
	name     string
	sid      string // Session ID
}

type piholeAuthRequest struct {
	Password string `json:"password"`
}

type piholeAuthResponse struct {
	Session struct {
		SID string `json:"sid"`
	} `json:"session"`
}

type piholeConfigResponse struct {
	Config struct {
		DNS struct {
			Hosts []string `json:"hosts"`
		} `json:"dns"`
	} `json:"config"`
}

func NewPiholeProvider(server, password, name string) *PiholeProvider {
	return &PiholeProvider{
		server:   strings.TrimSuffix(server, "/"),
		password: password,
		name:     name,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// authenticate obtains a session ID from Pi-hole API v6
func (p *PiholeProvider) authenticate() error {
	endpoint := fmt.Sprintf("%s/api/auth", p.server)

	authReq := piholeAuthRequest{Password: p.password}
	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp piholeAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	p.sid = authResp.Session.SID
	return nil
}

// findExistingEntry searches for an existing DNS entry for the domain
func (p *PiholeProvider) findExistingEntry(domain string) (string, error) {
	if p.sid == "" {
		if err := p.authenticate(); err != nil {
			return "", err
		}
	}

	endpoint := fmt.Sprintf("%s/api/config/dns", p.server)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-FTL-SID", p.sid)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get DNS config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Session expired, re-authenticate
		if err := p.authenticate(); err != nil {
			return "", err
		}
		return p.findExistingEntry(domain) // Retry
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get DNS config (status %d): %s", resp.StatusCode, string(body))
	}

	var configResp piholeConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return "", fmt.Errorf("failed to decode DNS config: %w", err)
	}

	// Find entry that ends with the domain
	for _, host := range configResp.Config.DNS.Hosts {
		if strings.HasSuffix(host, " "+domain) {
			return host, nil
		}
	}

	return "", nil // Not found
}

func (p *PiholeProvider) CreateRecord(domain, ip string) (string, error) {
	if p.sid == "" {
		if err := p.authenticate(); err != nil {
			return "", err
		}
	}

	// Check if entry exists and delete it
	oldEntry, err := p.findExistingEntry(domain)
	if err != nil {
		return "", err
	}

	if oldEntry != "" {
		if err := p.deleteEntry(oldEntry); err != nil {
			return "", fmt.Errorf("failed to delete old entry: %w", err)
		}
	}

	// Add new entry
	newEntry := fmt.Sprintf("%s %s", ip, domain)
	encoded := url.QueryEscape(newEntry)
	endpoint := fmt.Sprintf("%s/api/config/dns/hosts/%s", p.server, encoded)

	req, err := http.NewRequest("PUT", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-FTL-SID", p.sid)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create DNS record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Session expired, re-authenticate and retry
		if err := p.authenticate(); err != nil {
			return "", err
		}
		return p.CreateRecord(domain, ip)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create DNS record (status %d): %s", resp.StatusCode, string(body))
	}

	return fmt.Sprintf("pihole-%s-%s", p.name, domain), nil
}

func (p *PiholeProvider) UpdateRecord(recordID, domain, ip string) error {
	// Pi-hole API v6 doesn't have direct update - delete old and create new
	_, err := p.CreateRecord(domain, ip)
	return err
}

func (p *PiholeProvider) DeleteRecord(recordID string) error {
	// Extract domain from recordID (format: pihole-name-domain.com)
	parts := strings.SplitN(recordID, "-", 3)
	if len(parts) < 3 {
		return fmt.Errorf("invalid record ID format: %s", recordID)
	}
	domain := parts[2]

	// Find the existing entry
	entry, err := p.findExistingEntry(domain)
	if err != nil {
		return err
	}

	if entry == "" {
		return nil // Already deleted
	}

	return p.deleteEntry(entry)
}

func (p *PiholeProvider) GetRecordIP(domain string) (string, error) {
	entry, err := p.findExistingEntry(domain)
	if err != nil {
		return "", err
	}

	if entry == "" {
		return "", nil // Not found
	}

	// Entry format is "IP domain"
	parts := strings.Fields(entry)
	if len(parts) >= 2 {
		return parts[0], nil
	}

	return "", nil
}

// deleteEntry removes a DNS entry from Pi-hole
func (p *PiholeProvider) deleteEntry(entry string) error {
	if p.sid == "" {
		if err := p.authenticate(); err != nil {
			return err
		}
	}

	encoded := url.QueryEscape(entry)
	endpoint := fmt.Sprintf("%s/api/config/dns/hosts/%s", p.server, encoded)

	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-FTL-SID", p.sid)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete DNS entry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Session expired, re-authenticate and retry
		if err := p.authenticate(); err != nil {
			return err
		}
		return p.deleteEntry(entry)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete DNS entry (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
