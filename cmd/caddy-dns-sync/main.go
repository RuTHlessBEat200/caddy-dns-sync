package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/RuTHlessBEat200/caddy-dns-sync/pkg/config"
	"github.com/RuTHlessBEat200/caddy-dns-sync/pkg/parser"
	"github.com/RuTHlessBEat200/caddy-dns-sync/pkg/state"
)

const (
	Version = "1.0.0"
)

func main() {
	var (
		initMode       = flag.Bool("init", false, "Initialize directories and systemd service")
		cloudflareInit = flag.Bool("cloudflare", false, "Initialize only Cloudflare provider (use with --init)")
		bind9Init      = flag.Bool("bind9", false, "Initialize only Bind9 provider (use with --init)")
		piholeInit     = flag.Bool("pihole", false, "Initialize only Pi-hole provider (use with --init)")
		versionFlag    = flag.Bool("version", false, "Show version information")
		caddyfile      = flag.String("caddyfile", "/etc/caddy/Caddyfile", "Path to Caddyfile")
		providerConfig = flag.String("config", "/etc/caddy-dns-sync/provider.conf", "Path to provider configuration file")
		stateFile      = flag.String("state", "/etc/caddy-dns-sync/dns.state", "Path to DNS state file")
	)

	flag.Parse()

	if *versionFlag {
		fmt.Printf("caddy-dns-sync version %s\n", Version)
		os.Exit(0)
	}

	if *initMode {
		providers := []string{}

		if *cloudflareInit || *bind9Init || *piholeInit {
			if *cloudflareInit {
				providers = append(providers, "cloudflare")
			}
			if *bind9Init {
				providers = append(providers, "bind9")
			}
			if *piholeInit {
				providers = append(providers, "pihole")
			}
		} else {
			providers = []string{"cloudflare", "bind9", "pihole"}
		}

		if err := initializeSystem(providers); err != nil {
			log.Fatalf("Failed to initialize: %v", err)
		}
		fmt.Println("System initialized successfully!")
		fmt.Printf("Edit /etc/caddy-dns-sync/provider.conf to configure your DNS providers (%s)\n", strings.Join(providers, ", "))
		os.Exit(0)
	}

	// Parse Caddyfile
	log.Printf("Parsing Caddyfile: %s", *caddyfile)
	domains, err := parser.ParseCaddyfile(*caddyfile)
	if err != nil {
		log.Fatalf("Failed to parse Caddyfile: %v", err)
	}
	log.Printf("Found %d domains in Caddyfile", len(domains))

	// Load provider configuration
	log.Printf("Loading provider config: %s", *providerConfig)
	cfg, err := config.LoadProviderConfig(*providerConfig)
	if err != nil {
		log.Fatalf("Failed to load provider config: %v", err)
	}

	// Load existing state
	dnsState, err := state.LoadState(*stateFile)
	if err != nil {
		log.Printf("Warning: Failed to load state, starting fresh: %v", err)
		dnsState = &state.DNSState{Records: make(map[string]state.DNSRecord)}
	}

	// Sync DNS records
	if err := syncDNS(cfg, domains, dnsState); err != nil {
		log.Fatalf("Failed to sync DNS records: %v", err)
	}

	// Save updated state
	if err := state.SaveState(*stateFile, dnsState); err != nil {
		log.Fatalf("Failed to save state: %v", err)
	}

	log.Println("DNS sync completed successfully")
}
