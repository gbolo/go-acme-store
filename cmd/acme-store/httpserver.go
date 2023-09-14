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
	fiberApp.Get("/version", handlerVersion)
	fiberApp.Get("/healthz", handlerHealthCheck)
	fiberApp.Get("/certs", handlerCerts)

	listenAddress := fmt.Sprintf("%s:%s", viper.GetString("server.bind_address"), viper.GetString("server.bind_port"))
	log.Infof("server listening on %s", listenAddress)
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

func handlerCerts(c *fiber.Ctx) (err error) {

	domains, err := ks.GetAllDomains()
	if err != nil {
		return
	}

	var response []certInfo

	for _, domain := range domains {
		cert, _ := ks.GetLeafCert(domain)
		response = append(response, certInfo{
			CommonName: domain,
			LeafCert:   cert,
		})
	}
	return c.Status(200).JSON(response)
}

type certInfo struct {
	CommonName string `json:"common_name"`
	LeafCert   string `json:"leaf_cert_pem"`
}
