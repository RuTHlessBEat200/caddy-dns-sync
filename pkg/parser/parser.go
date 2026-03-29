package parser

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// parseCaddyfile extracts all domain names from a Caddyfile
func ParseCaddyfile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []string
	domainSet := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	// Regex to match valid FQDN patterns (must have at least 2 parts and valid TLD)
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z][-a-zA-Z0-9]*$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip snippet definitions (lines starting with parentheses)
		if strings.HasPrefix(line, "(") {
			continue
		}

		// Skip global options block
		if strings.HasPrefix(line, "{") {
			continue
		}

		// Skip directive lines (lines that don't start with a domain)
		if strings.Contains(line, "{") {
			// Extract the part before the brace
			beforeBrace := strings.Split(line, "{")[0]
			beforeBrace = strings.TrimSpace(beforeBrace)

			// Skip if it starts with http:// or https:// followed by IP
			if strings.HasPrefix(beforeBrace, "http://") || strings.HasPrefix(beforeBrace, "https://") {
				// Check if it's an IP address (simple check for metrics endpoint)
				if strings.Contains(beforeBrace, "://") {
					afterScheme := strings.Split(beforeBrace, "://")[1]
					// If it starts with a number, it's likely an IP
					if len(afterScheme) > 0 && afterScheme[0] >= '0' && afterScheme[0] <= '9' {
						continue
					}
				}
			}

			// Split by comma or space to handle multiple domains
			parts := regexp.MustCompile(`[,\s]+`).Split(beforeBrace, -1)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}

				// Skip if it's an IP address (contains only digits, dots, and colons)
				if isIPAddress(part) {
					continue
				}

				// Validate domain format: must match FQDN pattern
				if domainRegex.MatchString(part) && !isDirective(part) {
					domainSet[part] = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Convert set to slice
	for domain := range domainSet {
		domains = append(domains, domain)
	}

	return domains, nil
}

// isDirective checks if a string is likely a Caddyfile directive rather than a domain
func isDirective(s string) bool {
	directives := []string{
		"import", "reverse_proxy", "file_server", "respond", "handle",
		"route", "redir", "rewrite", "try_files", "root", "log",
		"encode", "header", "basicauth", "request_header", "tls",
	}

	for _, d := range directives {
		if strings.HasPrefix(s, d) {
			return true
		}
	}

	return false
}

// isIPAddress checks if a string looks like an IP address (IPv4 or IPv6)
func isIPAddress(s string) bool {
	// Simple check: if it starts with a digit and contains only digits, dots, or colons
	if len(s) == 0 {
		return false
	}

	// IPv4 pattern
	if s[0] >= '0' && s[0] <= '9' {
		for _, c := range s {
			if !(c >= '0' && c <= '9' || c == '.' || c == ':') {
				return false
			}
		}
		return true
	}

	return false
}
