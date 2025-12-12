package main

import (
	"fmt"
	"go-acme-store/pkg/httpserver"
	"go-acme-store/pkg/log"
	"go-acme-store/pkg/meta"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func httpServerDaemon(appName string) {
	fiberApp := httpserver.GetFiberApp(appName)
	
	// API routes
	fiberApp.Get("/version", handlerVersion)
	fiberApp.Get("/healthz", handlerHealthCheck)
	fiberApp.Get("/certs", handlerCerts)
	
	// Serve static files from web/static directory
	fiberApp.Static("/static", "./web/static")
	
	// Serve the web UI on root path
	fiberApp.Get("/", handlerWebUI)

	listenAddress := fmt.Sprintf("%s:%s", viper.GetString("server.bind_address"), viper.GetString("server.bind_port"))
	log.Infof("server listening on %s", listenAddress)
	log.Infof("web UI available at http://%s", listenAddress)
	err := fiberApp.Listen(listenAddress)
	if err != nil {
		log.Fatalf("could not start http server: %v", err)
	}
}

func handlerVersion(c *fiber.Ctx) (err error) {
	return c.Status(200).JSON(meta.GetAppMetadata())
}

func handlerHealthCheck(c *fiber.Ctx) (err error) {
	return c.Status(200).JSON(&fiber.Map{"status": "success"})
}

func handlerWebUI(c *fiber.Ctx) (err error) {
	return c.SendFile("./web/static/index.html")
}

func handlerCerts(c *fiber.Ctx) (err error) {

	domains, err := ks.GetAllDomains()
	if err != nil {
		return
	}

	var response []certInfo

	for _, domain := range domains {
		certAndKey, err := ks.GetCertAndKey(domain)
		if err != nil {
			log.Warnf("failed to get cert for domain %s: %v", domain, err)
			continue
		}
		response = append(response, certInfo{
			CommonName:   certAndKey.CommonName,
			SANs:         certAndKey.SANs,
			LeafCert:     certAndKey.LeafCertPEM,
			CertChain:    certAndKey.CertChainPEM,
			CertChainURL: certAndKey.CertChainURL,
			Issuers:      certAndKey.Issuers,
			IssuedOn:     certAndKey.IssuedOn,
			ExpiresOn:    certAndKey.Expiration,
		})
	}
	return c.Status(200).JSON(response)
}

type certInfo struct {
	CommonName   string   `json:"common_name"`
	SANs         []string `json:"sans"`
	LeafCert     string   `json:"leaf_cert_pem"`
	CertChain    string   `json:"cert_chain_pem"`
	CertChainURL string   `json:"cert_chain_url"`
	Issuers      []string `json:"issuers"`
	IssuedOn     string   `json:"issued_on"`
	ExpiresOn    string   `json:"expires_on"`
}
