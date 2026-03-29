package providers

// DNSProvider is the interface all DNS providers must implement
type DNSProvider interface {
	CreateRecord(domain, ip string) (string, error)
	UpdateRecord(recordID, domain, ip string) error
	DeleteRecord(recordID string) error
	GetRecordIP(domain string) (string, error) // Returns current IP for domain, empty string if not found
}
