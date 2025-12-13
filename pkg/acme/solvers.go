package acme

import (
	"github.com/caddyserver/certmagic"
	"github.com/libdns/acmedns"
	"github.com/libdns/digitalocean"
	"github.com/libdns/powerdns"
	"github.com/spf13/viper"
)

func getDigitaloceanDnsSolver() *certmagic.DNS01Solver {
	doToken := viper.GetString("acme.digitalocean_token")
	doSolver := &certmagic.DNS01Solver{}

	// Configure DNS resolver if specified
	if dnsServer := viper.GetString("acme.dns_server"); dnsServer != "" {
		doSolver.DNSManager.Resolvers = []string{dnsServer}
	}

	doSolver.DNSManager.DNSProvider = &digitalocean.Provider{APIToken: doToken}
	return doSolver
}

func getACMEDNSSolver() *certmagic.DNS01Solver {
	// Get acme-dns configuration
	serverURL := viper.GetString("acme.acmedns_server_url")
	if serverURL == "" {
		serverURL = "http://localhost:8053" // Default
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
	return acmeDnsSolver
}

func getPowerDNSSolver() *certmagic.DNS01Solver {
	pdnsSolver := &certmagic.DNS01Solver{}

	// Configure DNS resolver if specified - CRITICAL for zone finding!
	if dnsServer := viper.GetString("acme.dns_server"); dnsServer != "" {
		pdnsSolver.DNSManager.Resolvers = []string{dnsServer}
	}

	pdnsSolver.DNSManager.DNSProvider = &powerdns.Provider{
		ServerURL: viper.GetString("acme.powerdns_server_url"),
		APIToken:  viper.GetString("acme.powerdns_api_token"),
		ServerID:  viper.GetString("acme.powerdns_server_id"),
	}
	return pdnsSolver
}
