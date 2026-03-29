package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type DNSRecord struct {
	Domain    string    `json:"domain"`
	RecordID  string    `json:"record_id"`
	IPAddress string    `json:"ip_address"`
	Provider  string    `json:"provider"`
	Zone      string    `json:"zone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DNSState struct {
	Records   map[string]DNSRecord `json:"records"` // Key: provider:zone:domain
	UpdatedAt time.Time            `json:"updated_at"`
}

func LoadState(path string) (*DNSState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DNSState{Records: make(map[string]DNSRecord)}, nil
		}
		return nil, err
	}

	var state DNSState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Records == nil {
		state.Records = make(map[string]DNSRecord)
	}

	return &state, nil
}

func SaveState(path string, state *DNSState) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	state.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func MakeStateKey(provider, zone, domain string) string {
	if zone == "" {
		return provider + ":" + domain
	}
	return provider + ":" + zone + ":" + domain
}
