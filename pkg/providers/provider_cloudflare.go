package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	CloudflareAPIBase = "https://api.cloudflare.com/client/v4"
)

type CloudflareProvider struct {
	apiToken string
	zoneID   string
	zone     string
	client   *http.Client
}

type CloudflareDNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type CloudflareResponse struct {
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
	Result  interface{}   `json:"result"`
}

type CloudflareListResponse struct {
	Success bool                  `json:"success"`
	Errors  []interface{}         `json:"errors"`
	Result  []CloudflareDNSRecord `json:"result"`
}

func NewCloudflareProvider(apiToken, zoneID, zone string) *CloudflareProvider {
	return &CloudflareProvider{
		apiToken: apiToken,
		zoneID:   zoneID,
		zone:     zone,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *CloudflareProvider) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	url := CloudflareAPIBase + endpoint
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	return c.client.Do(req)
}

func (c *CloudflareProvider) CreateRecord(domain, ip string) (string, error) {
	record := CloudflareDNSRecord{
		Type:    "A",
		Name:    domain,
		Content: ip,
		TTL:     1, // Auto TTL
		Proxied: false,
	}

	endpoint := fmt.Sprintf("/zones/%s/dns_records", c.zoneID)
	resp, err := c.makeRequest("POST", endpoint, record)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(bodyData, &cfResp); err != nil {
		return "", err
	}

	if !cfResp.Success {
		return "", fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	// Parse the result as a DNS record
	resultJSON, err := json.Marshal(cfResp.Result)
	if err != nil {
		return "", err
	}

	var createdRecord CloudflareDNSRecord
	if err := json.Unmarshal(resultJSON, &createdRecord); err != nil {
		return "", err
	}

	return createdRecord.ID, nil
}

func (c *CloudflareProvider) UpdateRecord(recordID, domain, ip string) error {
	record := CloudflareDNSRecord{
		Type:    "A",
		Name:    domain,
		Content: ip,
		TTL:     1,
		Proxied: false,
	}

	endpoint := fmt.Sprintf("/zones/%s/dns_records/%s", c.zoneID, recordID)
	resp, err := c.makeRequest("PUT", endpoint, record)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(bodyData, &cfResp); err != nil {
		return err
	}

	if !cfResp.Success {
		return fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	return nil
}

func (c *CloudflareProvider) DeleteRecord(recordID string) error {
	endpoint := fmt.Sprintf("/zones/%s/dns_records/%s", c.zoneID, recordID)
	resp, err := c.makeRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(bodyData, &cfResp); err != nil {
		return err
	}

	if !cfResp.Success {
		return fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	return nil
}

func (c *CloudflareProvider) GetRecordIP(domain string) (string, error) {
	endpoint := fmt.Sprintf("/zones/%s/dns_records?type=A&name=%s", c.zoneID, domain)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var cfResp CloudflareListResponse
	if err := json.Unmarshal(bodyData, &cfResp); err != nil {
		return "", err
	}

	if !cfResp.Success {
		return "", fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	if len(cfResp.Result) == 0 {
		return "", nil // Not found
	}

	return cfResp.Result[0].Content, nil
}
