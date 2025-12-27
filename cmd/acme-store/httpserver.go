package main

import (
	"fmt"
	"go-acme-store/pkg/httpserver"
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"
	"go-acme-store/pkg/meta"
	"go-acme-store/pkg/webui"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
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
	api.Get("/config", handlerConfig)
	api.Get("/certs", handlerCerts)
	api.Get("/certs/:domain/private-key", handlerGetPrivateKey)
	api.Delete("/certs/:domain", handlerDeleteCert)

	// Domain management API
	api.Get("/domains", handlerGetDomains)
	api.Post("/domains", handlerAddDomain)
	api.Delete("/domains/:domain", handlerRemoveDomain)

	// ACME operations
	api.Post("/trigger-renewal", handlerTriggerRenewal)

	// Keystore backup/restore operations
	api.Get("/keystore/export", handlerExportKeystore)
	api.Post("/keystore/import", handlerImportKeystore)

	// UI routes - serve embedded static files
	ui := fiberApp.Group("/ui")
	ui.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(webui.StaticFiles),
		PathPrefix: "static",
		Browse:     false,
	}))
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
	health := &fiber.Map{
		"status": "healthy",
		"keystore": &fiber.Map{
			"connected": false,
			"status":    "unavailable",
		},
	}

	// Check keystore connectivity
	if err := ensureKeystore(); err == nil {
		// Keystore is connected, try a simple operation to verify it's working
		_, err := ks.GetManagedDomains()
		if err == nil {
			(*health)["keystore"] = &fiber.Map{
				"connected": true,
				"status":    "healthy",
			}
		} else {
			(*health)["keystore"] = &fiber.Map{
				"connected": true,
				"status":    "error",
				"error":     err.Error(),
			}
			(*health)["status"] = "degraded"
		}
	} else {
		(*health)["keystore"] = &fiber.Map{
			"connected": false,
			"status":    "unavailable",
		}
		(*health)["status"] = "degraded"
	}

	statusCode := 200
	if (*health)["status"] == "degraded" {
		statusCode = 503
	}

	return c.Status(statusCode).JSON(health)
}

func handlerConfig(c *fiber.Ctx) (err error) {
	// Return non-sensitive configuration
	config := &fiber.Map{
		"log": &fiber.Map{
			"level": viper.GetString("log.level"),
		},
		"server": &fiber.Map{
			"bind_address": viper.GetString("server.bind_address"),
			"bind_port":    viper.GetString("server.bind_port"),
		},
		"acme": &fiber.Map{
			"directory":     viper.GetString("acme.directory"),
			"account_email": viper.GetString("acme.account_email"),
			"dns_server":    viper.GetString("acme.dns_server"),
			"dns_provider":  viper.GetString("acme.dns_provider"),
			// NOTE: digitalocean_token is intentionally excluded (sensitive)
		},
		"keystore": &fiber.Map{
			"backend": viper.GetString("keystore.backend"),
		},
	}

	// Add backend-specific config
	backend := viper.GetString("keystore.backend")
	switch backend {
	case "vault":
		(*config)["vault"] = &fiber.Map{
			"address":         viper.GetString("vault.address"),
			"kv2_mount":       viper.GetString("vault.kv2_mount"),
			"kv2_secret_path": viper.GetString("vault.kv2_secret_path"),
			"tls_skip_verify": viper.GetBool("vault.tls_skip_verify"),
			// NOTE: token is intentionally excluded (sensitive)
		}
	case "filesystem":
		(*config)["filesystem"] = &fiber.Map{
			"base_path": viper.GetString("filesystem.base_path"),
		}
	}

	return c.Status(200).JSON(config)
}

func handlerWebUI(c *fiber.Ctx) (err error) {
	// Read index.html from embedded filesystem
	indexHTML, err := webui.StaticFiles.ReadFile("static/index.html")
	if err != nil {
		return c.Status(500).SendString("Failed to load UI")
	}
	c.Set("Content-Type", "text/html")
	return c.Send(indexHTML)
}

func handlerCerts(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

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

		// Default status to "issued" if not set (for backward compatibility)
		status := certAndKey.Status
		if status == "" && certAndKey.LeafCertPEM != "" {
			status = "issued"
		} else if status == "" {
			status = "pending"
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
			Status:       status,
			Error:        certAndKey.Error,
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
	Status       string   `json:"status"`
	Error        string   `json:"error,omitempty"`
}

// Domain management handlers

func handlerGetDomains(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

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
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

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
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

	domain := c.Params("domain")
	deleteCert := c.Query("delete_cert", "false") == "true"

	if domain == "" {
		return c.Status(400).JSON(&fiber.Map{
			"error": "domain parameter is required",
		})
	}

	// Remove from managed domains list (may fail if already unmanaged)
	removeErr := ks.RemoveManagedDomain(domain)

	if deleteCert {
		// Hard delete - remove certificate from Vault
		// Always try to delete cert even if domain was already unmanaged
		err = ks.DeleteCertAndKey(domain)
		if err != nil {
			log.Warnf("failed to delete cert for domain %s: %v", domain, err)
			// If domain removal also failed, return that error
			if removeErr != nil {
				return c.Status(404).JSON(&fiber.Map{
					"error": fmt.Sprintf("domain not found: %v", removeErr),
				})
			}
			return c.Status(500).JSON(&fiber.Map{
				"error": fmt.Sprintf("failed to delete certificate: %v", err),
			})
		}
		log.Infof("domain %s removed and certificate deleted via API", domain)
		return c.Status(200).JSON(&fiber.Map{
			"message": fmt.Sprintf("domain %s removed and certificate deleted", domain),
			"domain":  domain,
		})
	}

	// Soft delete path - must be a managed domain
	if removeErr != nil {
		return c.Status(404).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to remove domain: %v", removeErr),
		})
	}

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

	log.Infof("domain %s removed via API", domain)

	return c.Status(200).JSON(&fiber.Map{
		"message": fmt.Sprintf("domain %s removed successfully", domain),
		"domain":  domain,
	})
}

func handlerTriggerRenewal(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

	log.Infof("certificate renewal triggered via API")

	// Trigger the ACME daemon to run
	go triggerAcmeOrders()

	return c.Status(202).JSON(&fiber.Map{
		"message": "certificate renewal triggered, check logs for progress",
	})
}

func handlerGetPrivateKey(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

	domain := c.Params("domain")

	if domain == "" {
		return c.Status(400).JSON(&fiber.Map{
			"error": "domain parameter is required",
		})
	}

	// Get certificate and key data
	certAndKey, err := ks.GetCertAndKey(domain)
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to retrieve certificate: %v", err),
		})
	}

	// Check if certificate exists
	if certAndKey.CommonName == "" {
		return c.Status(404).JSON(&fiber.Map{
			"error": fmt.Sprintf("certificate not found for domain %s", domain),
		})
	}

	log.Infof("private key requested for domain %s via API", domain)

	return c.Status(200).JSON(&fiber.Map{
		"domain":      domain,
		"private_key": certAndKey.PrivateKeyPEM,
	})
}

func handlerDeleteCert(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

	domain := c.Params("domain")

	if domain == "" {
		return c.Status(400).JSON(&fiber.Map{
			"error": "domain parameter is required",
		})
	}

	// Check if certificate exists
	certAndKey, err := ks.GetCertAndKey(domain)
	if err != nil {
		return c.Status(404).JSON(&fiber.Map{
			"error": fmt.Sprintf("certificate not found for domain %s", domain),
		})
	}

	// Check if certificate is managed
	if certAndKey.Managed {
		return c.Status(400).JSON(&fiber.Map{
			"error": fmt.Sprintf("cannot delete managed certificate for domain %s, unmanage it first by calling DELETE /api/domains/%s", domain, domain),
		})
	}

	// Delete the certificate
	err = ks.DeleteCertAndKey(domain)
	if err != nil {
		log.Warnf("failed to delete cert for domain %s: %v", domain, err)
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to delete certificate: %v", err),
		})
	}

	log.Infof("certificate deleted for domain %s via API", domain)
	return c.Status(200).JSON(&fiber.Map{
		"message": fmt.Sprintf("certificate for domain %s deleted successfully", domain),
		"domain":  domain,
	})
}

// Keystore export/import structures and handlers

type keystoreExport struct {
	Version      string                        `json:"version"`
	ExportedAt   string                        `json:"exported_at"`
	AcmeAccount  keystoreAccountExport         `json:"acme_account"`
	Certificates map[string]keystoreCertExport `json:"certificates"`
	Domains      []string                      `json:"managed_domains"`
}

type keystoreAccountExport struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

type keystoreCertExport struct {
	CertChainPEM  string   `json:"cert_chain_pem"`
	LeafCertPEM   string   `json:"leaf_cert_pem"`
	CertChainURL  string   `json:"cert_chain_url"`
	PrivateKeyPEM string   `json:"private_key_pem"`
	Issuers       []string `json:"issuers"`
	IssuedOn      string   `json:"issued_on"`
	Expiration    string   `json:"expires_on"`
	CommonName    string   `json:"common_name"`
	SANs          []string `json:"sans"`
	Managed       bool     `json:"managed"`
	Status        string   `json:"status"`
	Error         string   `json:"error,omitempty"`
}

func handlerExportKeystore(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

	log.Infof("keystore export requested via API")

	// Get ACME account
	account, err := ks.GetAcmeAccount()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to get ACME account: %v", err),
		})
	}

	// Get all domains
	domains, err := ks.GetAllDomains()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to get domains: %v", err),
		})
	}

	// Get managed domains
	managedDomains, err := ks.GetManagedDomains()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"error": fmt.Sprintf("failed to get managed domains: %v", err),
		})
	}

	// Build export data
	exportData := keystoreExport{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		AcmeAccount: keystoreAccountExport{
			Email: account.Email,
			Key:   account.Key,
		},
		Certificates: make(map[string]keystoreCertExport),
		Domains:      managedDomains,
	}

	// Export all certificates
	for _, domain := range domains {
		certAndKey, err := ks.GetCertAndKey(domain)
		if err != nil {
			log.Warnf("failed to get cert for domain %s during export: %v", domain, err)
			continue
		}

		exportData.Certificates[domain] = keystoreCertExport{
			CertChainPEM:  certAndKey.CertChainPEM,
			LeafCertPEM:   certAndKey.LeafCertPEM,
			CertChainURL:  certAndKey.CertChainURL,
			PrivateKeyPEM: certAndKey.PrivateKeyPEM,
			Issuers:       certAndKey.Issuers,
			IssuedOn:      certAndKey.IssuedOn,
			Expiration:    certAndKey.Expiration,
			CommonName:    certAndKey.CommonName,
			SANs:          certAndKey.SANs,
			Managed:       certAndKey.Managed,
			Status:        certAndKey.Status,
			Error:         certAndKey.Error,
		}
	}

	log.Infof("keystore exported: %d certificates, %d managed domains", len(exportData.Certificates), len(exportData.Domains))

	return c.Status(200).JSON(exportData)
}

func handlerImportKeystore(c *fiber.Ctx) (err error) {
	if err := ensureKeystore(); err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"error": err.Error(),
		})
	}

	log.Infof("keystore import requested via API")

	// Parse import data
	var importData keystoreExport
	if err := c.BodyParser(&importData); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"error": fmt.Sprintf("invalid import data: %v", err),
		})
	}

	// Validate version
	if importData.Version != "1.0" {
		return c.Status(400).JSON(&fiber.Map{
			"error": fmt.Sprintf("unsupported export version: %s (expected 1.0)", importData.Version),
		})
	}

	// Track import statistics
	var (
		certsImported   = 0
		certsFailed     = 0
		domainsImported = 0
		domainsFailed   = 0
	)

	// Import certificates
	for domain, cert := range importData.Certificates {
		certAndKey := keystore.CertAndKey{
			CertChainPEM:  cert.CertChainPEM,
			LeafCertPEM:   cert.LeafCertPEM,
			CertChainURL:  cert.CertChainURL,
			PrivateKeyPEM: cert.PrivateKeyPEM,
			Issuers:       cert.Issuers,
			IssuedOn:      cert.IssuedOn,
			Expiration:    cert.Expiration,
			CommonName:    cert.CommonName,
			SANs:          cert.SANs,
			Managed:       cert.Managed,
			Status:        cert.Status,
			Error:         cert.Error,
		}

		if err := ks.StoreCertAndKey(domain, certAndKey); err != nil {
			log.Warnf("failed to import certificate for %s: %v", domain, err)
			certsFailed++
		} else {
			certsImported++
		}
	}

	// Import managed domains
	for _, domain := range importData.Domains {
		// Check if domain already exists
		isManaged, err := ks.IsManagedDomain(domain)
		if err != nil {
			log.Warnf("failed to check if domain %s is managed: %v", domain, err)
			domainsFailed++
			continue
		}

		if !isManaged {
			// Get SANs from certificate if available
			var sans []string
			if cert, ok := importData.Certificates[domain]; ok {
				sans = cert.SANs
			}

			if len(sans) > 0 {
				err = ks.AddManagedDomainWithSANs(domain, sans)
			} else {
				err = ks.AddManagedDomain(domain)
			}

			if err != nil {
				log.Warnf("failed to import managed domain %s: %v", domain, err)
				domainsFailed++
			} else {
				domainsImported++
			}
		}
	}

	log.Infof("keystore import complete: certificates=%d/%d, managed_domains=%d/%d",
		certsImported, len(importData.Certificates), domainsImported, len(importData.Domains))

	return c.Status(200).JSON(&fiber.Map{
		"message": "keystore import completed",
		"statistics": &fiber.Map{
			"certificates_imported": certsImported,
			"certificates_failed":   certsFailed,
			"certificates_total":    len(importData.Certificates),
			"domains_imported":      domainsImported,
			"domains_failed":        domainsFailed,
			"domains_total":         len(importData.Domains),
		},
	})
}
