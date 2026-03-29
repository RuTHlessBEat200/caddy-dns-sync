package main

import (
	"fmt"
	"log"
	"time"

	"github.com/yourusername/caddy-dns-sync/pkg/config"
	"github.com/yourusername/caddy-dns-sync/pkg/providers"
	"github.com/yourusername/caddy-dns-sync/pkg/state"
)

func syncDNS(cfg *config.ProviderConfig, domains []string, dnsState *state.DNSState) error {
	// Sync Cloudflare zones
	if cfg.Cloudflare != nil {
		for zone, zoneConfig := range cfg.Cloudflare {
			if err := syncCloudflareZone(zone, zoneConfig, domains, dnsState); err != nil {
				log.Printf("Error syncing Cloudflare zone %s: %v", zone, err)
			}
		}
	}

	// Sync Bind9 zones
	if cfg.Bind9 != nil {
		for zone, zoneConfig := range cfg.Bind9 {
			if err := syncBind9Zone(zone, zoneConfig, domains, dnsState); err != nil {
				log.Printf("Error syncing Bind9 zone %s: %v", zone, err)
			}
		}
	}

	// Sync Pi-hole instances
	if cfg.Pihole != nil {
		for name, piholeConfig := range cfg.Pihole {
			if err := syncPihole(name, piholeConfig, domains, dnsState); err != nil {
				log.Printf("Error syncing Pi-hole %s: %v", name, err)
			}
		}
	}

	return nil
}

func syncCloudflareZone(zone string, zoneCfg config.CloudflareZoneConfig, domains []string, dnsState *state.DNSState) error {
	log.Printf("[Cloudflare] Syncing zone: %s", zone)

	ip, err := config.GetIPFromInterface(zoneCfg.IPInterface)
	if err != nil {
		return fmt.Errorf("failed to get IP from interface %s: %w", zoneCfg.IPInterface, err)
	}
	log.Printf("[Cloudflare] Using IP %s from interface %s", ip, zoneCfg.IPInterface)

	provider := providers.NewCloudflareProvider(zoneCfg.APIToken, zoneCfg.ZoneID, zone)
	return syncProviderDomains(provider, "cloudflare", zone, domains, ip, dnsState)
}

func syncBind9Zone(zone string, zoneCfg config.Bind9ZoneConfig, domains []string, dnsState *state.DNSState) error {
	log.Printf("[Bind9] Syncing zone: %s", zone)

	ip, err := config.GetIPFromInterface(zoneCfg.IPInterface)
	if err != nil {
		return fmt.Errorf("failed to get IP from interface %s: %w", zoneCfg.IPInterface, err)
	}
	log.Printf("[Bind9] Using IP %s from interface %s", ip, zoneCfg.IPInterface)

	provider := providers.NewBind9Provider(zoneCfg.Server, zoneCfg.Port, zoneCfg.KeyName, zoneCfg.KeySecret, zoneCfg.Algorithm, zone)
	return syncProviderDomains(provider, "bind9", zone, domains, ip, dnsState)
}

func syncPihole(name string, piholeCfg config.PiholeConfig, domains []string, dnsState *state.DNSState) error {
	log.Printf("[Pi-hole] Syncing instance: %s", name)

	ip, err := config.GetIPFromInterface(piholeCfg.IPInterface)
	if err != nil {
		return fmt.Errorf("failed to get IP from interface %s: %w", piholeCfg.IPInterface, err)
	}
	log.Printf("[Pi-hole] Using IP %s from interface %s", ip, piholeCfg.IPInterface)

	provider := providers.NewPiholeProvider(piholeCfg.Server, piholeCfg.Password, name)
	return syncProviderDomains(provider, "pihole", name, domains, ip, dnsState)
}

func syncProviderDomains(provider providers.DNSProvider, providerName, zone string, domains []string, targetIP string, dnsState *state.DNSState) error {
	zoneDomains := filterDomainsForZone(domains, zone)
	currentDomains := make(map[string]bool)
	for _, domain := range zoneDomains {
		currentDomains[domain] = true
	}

	// Add or update domains
	for _, domain := range zoneDomains {
		stateKey := state.MakeStateKey(providerName, zone, domain)
		existingRecord, existsInState := dnsState.Records[stateKey]

		// Query actual DNS record from provider API
		actualIP, err := provider.GetRecordIP(domain)
		if err != nil {
			log.Printf("[%s:%s] Warning: Failed to query DNS record for %s: %v", providerName, zone, domain, err)
		}

		recordExists := actualIP != ""

		if !recordExists {
			// Record doesn't exist in DNS provider, create it
			recordID, err := provider.CreateRecord(domain, targetIP)
			if err != nil {
				log.Printf("[%s:%s] Error creating %s: %v", providerName, zone, domain, err)
				continue
			}

			dnsState.Records[stateKey] = state.DNSRecord{
				Domain:    domain,
				RecordID:  recordID,
				IPAddress: targetIP,
				Provider:  providerName,
				Zone:      zone,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			log.Printf("[%s:%s] Created: %s -> %s", providerName, zone, domain, targetIP)
		} else if actualIP != targetIP {
			// Record exists but IP is different, update it
			// Use recordID from state if available, otherwise create new
			recordID := ""
			if existsInState {
				recordID = existingRecord.RecordID
			}

			err := provider.UpdateRecord(recordID, domain, targetIP)
			if err != nil {
				log.Printf("[%s:%s] Error updating %s: %v", providerName, zone, domain, err)
				continue
			}

			// Update state with actual values
			if existsInState {
				existingRecord.IPAddress = targetIP
				existingRecord.UpdatedAt = time.Now()
				dnsState.Records[stateKey] = existingRecord
			} else {
				// Record exists in DNS but not in state, add to state
				dnsState.Records[stateKey] = state.DNSRecord{
					Domain:    domain,
					RecordID:  recordID,
					IPAddress: targetIP,
					Provider:  providerName,
					Zone:      zone,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			}
			log.Printf("[%s:%s] Updated: %s -> %s (was %s)", providerName, zone, domain, targetIP, actualIP)
		} else {
			// Record is up to date, sync state file with actual IP silently
			if existsInState && existingRecord.IPAddress != actualIP {
				existingRecord.IPAddress = actualIP
				existingRecord.UpdatedAt = time.Now()
				dnsState.Records[stateKey] = existingRecord
			} else if !existsInState {
				// Add to state if missing
				dnsState.Records[stateKey] = state.DNSRecord{
					Domain:    domain,
					RecordID:  "", // Unknown
					IPAddress: actualIP,
					Provider:  providerName,
					Zone:      zone,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			}
		}
	}

	// Delete records that are in state but not in Caddyfile
	for stateKey, record := range dnsState.Records {
		if record.Provider != providerName || record.Zone != zone {
			continue
		}

		if !currentDomains[record.Domain] {
			err := provider.DeleteRecord(record.RecordID)
			if err != nil {
				log.Printf("[%s:%s] Error deleting %s: %v", providerName, zone, record.Domain, err)
				continue
			}

			delete(dnsState.Records, stateKey)
			log.Printf("[%s:%s] Deleted: %s", providerName, zone, record.Domain)
		}
	}

	return nil
}

func filterDomainsForZone(domains []string, zone string) []string {
	// For now, return all domains
	// TODO: Filter domains that belong to this zone
	return domains
}
