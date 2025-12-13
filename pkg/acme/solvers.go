package acme

import (
	"fmt"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/acmedns"
	"github.com/libdns/digitalocean"
	"github.com/libdns/powerdns"
	"github.com/spf13/viper"
)

func getDigitaloceanDnsSolver() (*certmagic.DNS01Solver, error) {
	doToken := viper.GetString("acme.digitalocean_token")
	if doToken == "" {
		return nil, fmt.Errorf("DigitalOcean DNS provider requires 'acme.digitalocean_token' to be configured")
	}

	doSolver := &certmagic.DNS01Solver{}

	// Configure DNS resolver if specified
	if dnsServer := viper.GetString("acme.dns_server"); dnsServer != "" {
		doSolver.DNSManager.Resolvers = []string{dnsServer}
	}

	doSolver.DNSManager.DNSProvider = &digitalocean.Provider{APIToken: doToken}
	return doSolver, nil
}

func getACMEDNSSolver() (*certmagic.DNS01Solver, error) {
	// Get acme-dns configuration
	serverURL := viper.GetString("acme.acmedns_server_url")
	if serverURL == "" {
		return nil, fmt.Errorf("ACME-DNS provider requires 'acme.acmedns_server_url' to be configured")
	}

	// Load existing accounts from config if provided
	configs := make(map[string]acmedns.DomainConfig)

	// You can optionally pre-configure accounts here
	// For now, acme-dns will auto-register on first use

	acmeDnsSolver := &certmagic.DNS01Solver{}

	// Configure DNS resolver if specified
	if dnsServer := viper.GetString("acme.dns_server"); dnsServer != "" {
		acmeDnsSolver.DNSManager.Resolvers = []string{dnsServer}
	}

	acmeDnsSolver.DNSManager.DNSProvider = &acmedns.Provider{
		ServerURL: serverURL,
		Configs:   configs,
	}
	return acmeDnsSolver, nil
}

func getPowerDNSSolver() (*certmagic.DNS01Solver, error) {
	serverURL := viper.GetString("acme.powerdns_server_url")
	apiToken := viper.GetString("acme.powerdns_api_token")
	serverID := viper.GetString("acme.powerdns_server_id")

	// Validate required configuration
	if serverURL == "" {
		return nil, fmt.Errorf("PowerDNS provider requires 'acme.powerdns_server_url' to be configured")
	}
	if apiToken == "" {
		return nil, fmt.Errorf("PowerDNS provider requires 'acme.powerdns_api_token' to be configured")
	}
	if serverID == "" {
		return nil, fmt.Errorf("PowerDNS provider requires 'acme.powerdns_server_id' to be configured")
	}

	pdnsSolver := &certmagic.DNS01Solver{}

	// Configure DNS resolver if specified - CRITICAL for zone finding!
	if dnsServer := viper.GetString("acme.dns_server"); dnsServer != "" {
		pdnsSolver.DNSManager.Resolvers = []string{dnsServer}
	}

	pdnsSolver.DNSManager.DNSProvider = &powerdns.Provider{
		ServerURL: serverURL,
		APIToken:  apiToken,
		ServerID:  serverID,
	}
	return pdnsSolver, nil
}
