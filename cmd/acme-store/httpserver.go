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
	
	// Root redirect to UI
	fiberApp.Get("/", handlerRootRedirect)
	
	// API routes
	api := fiberApp.Group("/api")
	api.Get("/version", handlerVersion)
	api.Get("/healthz", handlerHealthCheck)
	api.Get("/certs", handlerCerts)
	
	// Domain management API
	api.Get("/domains", handlerGetDomains)
	api.Post("/domains", handlerAddDomain)
	api.Delete("/domains/:domain", handlerRemoveDomain)
	
	// ACME operations
	api.Post("/trigger-renewal", handlerTriggerRenewal)
	
	// UI routes
	ui := fiberApp.Group("/ui")
	ui.Static("/static", "./web/static")
	ui.Get("/", handlerWebUI)

	listenAddress := fmt.Sprintf("%s:%s", viper.GetString("server.bind_address"), viper.GetString("server.bind_port"))
	log.Infof("server listening on %s", listenAddress)
	log.Infof("web UI available at http://%s/ui", listenAddress)
	log.Infof("api available at http://%s/api", listenAddress)
	err := fiberApp.Listen(listenAddress)
	if err != nil {
		log.Fatalf("could not start http server: %v", err)
	}
}

func handlerRootRedirect(c *fiber.Ctx) (err error) {
	return c.Redirect("/ui", fiber.StatusMovedPermanently)
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
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to get domains: %v", err),
		})
	}

	var response []certInfo

	for _, domain := range domains {
		certAndKey, err := ks.GetCertAndKey(domain)
		if err != nil {
			log.Warnf("failed to get cert for domain %s: %v", domain, err)
			continue
		}
		
		// Skip if certificate data is empty
		if certAndKey.CommonName == "" {
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
			Managed:      certAndKey.Managed,
		})
	}
	
	// Return empty array instead of null if no certificates
	if response == nil {
		response = []certInfo{}
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
	Managed      bool     `json:"managed"`
}

// Domain management handlers

func handlerGetDomains(c *fiber.Ctx) (err error) {
	domains, err := ks.GetManagedDomains()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to get managed domains: %v", err),
		})
	}
	
	return c.Status(200).JSON(&fiber.Map{
		"domains": domains,
		"count":   len(domains),
	})
}

func handlerAddDomain(c *fiber.Ctx) (err error) {
	type addDomainRequest struct {
		Domain string   `json:"domain"`
		SANs   []string `json:"sans,omitempty"`
	}
	
	var req addDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"error": "invalid request body",
		})
	}
	
	if req.Domain == "" {
		return c.Status(400).JSON(&fiber.Map{
			"error": "domain field is required",
		})
	}
	
	err = ks.AddManagedDomainWithSANs(req.Domain, req.SANs)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to add domain: %v", err),
		})
	}
	
	if req.SANs != nil && len(req.SANs) > 0 {
		log.Infof("domain %s added via API with SANs: %v", req.Domain, req.SANs)
	} else {
		log.Infof("domain %s added via API", req.Domain)
	}
	
	return c.Status(201).JSON(&fiber.Map{
		"message": fmt.Sprintf("domain %s added successfully", req.Domain),
		"domain":  req.Domain,
		"sans":    req.SANs,
	})
}

func handlerRemoveDomain(c *fiber.Ctx) (err error) {
	domain := c.Params("domain")
	deleteCert := c.Query("delete_cert", "false") == "true"
	
	if domain == "" {
		return c.Status(400).JSON(&fiber.Map{
			"error": "domain parameter is required",
		})
	}
	
	// Remove from managed domains list
	err = ks.RemoveManagedDomain(domain)
	if err != nil {
		return c.Status(404).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to remove domain: %v", err),
		})
	}
	
	if deleteCert {
		// Hard delete - remove certificate from Vault
		err = ks.DeleteCertAndKey(domain)
		if err != nil {
			log.Warnf("failed to delete cert for domain %s: %v", domain, err)
		} else {
			log.Infof("domain %s removed and certificate deleted via API", domain)
			return c.Status(200).JSON(&fiber.Map{
				"message": fmt.Sprintf("domain %s removed and certificate deleted", domain),
				"domain":  domain,
			})
		}
	} else {
		// Soft delete - mark certificate as unmanaged
		certAndKey, err := ks.GetCertAndKey(domain)
		if err == nil && certAndKey.CommonName != "" {
			certAndKey.Managed = false
			err = ks.StoreCertAndKey(domain, certAndKey)
			if err != nil {
				log.Warnf("failed to mark cert as unmanaged for domain %s: %v", domain, err)
			} else {
				log.Infof("domain %s removed from management (cert marked as unmanaged)", domain)
			}
		}
	}
	
	log.Infof("domain %s removed via API", domain)
	
	return c.Status(200).JSON(&fiber.Map{
		"message": fmt.Sprintf("domain %s removed successfully", domain),
		"domain":  domain,
	})
}

func handlerTriggerRenewal(c *fiber.Ctx) (err error) {
	log.Infof("certificate renewal triggered via API")
	
	// Trigger the ACME daemon to run
	go triggerAcmeOrders()
	
	return c.Status(202).JSON(&fiber.Map{
		"message": "certificate renewal triggered, check logs for progress",
	})
}
