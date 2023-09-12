package acme

import (
	"github.com/caddyserver/certmagic"
	"github.com/libdns/digitalocean"
	"github.com/spf13/viper"
	"time"
)

func defaultDnsSolver() *certmagic.DNS01Solver {
	dnsSolver := &certmagic.DNS01Solver{
		PropagationDelay:   0 * time.Second,
		PropagationTimeout: 2 * time.Minute,
		TTL:                60 * time.Second,
	}
	if dnsServer := viper.GetString("acme.dns_server"); dnsServer != "" {
		dnsSolver.Resolvers = []string{dnsServer}
	}
	return dnsSolver
}

func getDigitaloceanDnsSolver() *certmagic.DNS01Solver {
	doToken := viper.GetString("acme.digitalocean_token")
	doSolver := defaultDnsSolver()
	doSolver.DNSProvider = &digitalocean.Provider{APIToken: doToken}
	return doSolver
}
