package acme

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"go-acme-store/pkg/crypto"
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mholt/acmez/v3"
	"github.com/mholt/acmez/v3/acme"
	"github.com/spf13/viper"
)

func retrieveAcmeAccount(ks keystore.Keystore, client *acmez.Client) (*acme.Account, error) {
	savedAccount, err := ks.GetAcmeAccount()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve account from keystore: %v", err)
	}

	accountPrivateKey, err := crypto.DecodePrivateKey(savedAccount.Key)
	if err != nil {
		return nil, fmt.Errorf("could not decode account provate key from keystore: %v", err)
	}
	account, err := client.GetAccount(context.Background(), acme.Account{
		Contact:              []string{fmt.Sprintf("mailto:%s", savedAccount.Email)},
		TermsOfServiceAgreed: true,
		PrivateKey:           accountPrivateKey,
	})

	if err == nil {
		log.Infof("account %s was retrieved from acme provider", savedAccount.Email)
		return &account, nil
	}

	// determine if we should create an account
	// currently, only attempt this when ProblemTypeAccountDoesNotExist
	if !strings.Contains(err.Error(), acme.ProblemTypeAccountDoesNotExist) {
		return nil, fmt.Errorf("could not retrieve account from acme provider: %v", err)
	}

	log.Warnf("account %v does not exist with this key", savedAccount.Email)
	account, err = client.NewAccount(context.Background(), account)
	if err != nil {
		return nil, fmt.Errorf("could not create new acme account: %v", err)
	}

	log.Infof("created new account %s with acme provider", savedAccount.Email)
	return &account, nil
}

func RenewCertIfNeeded(ks keystore.Keystore, cn string, sans []string) (err error) {

	existingCert, err := ks.GetLeafCert(cn)
	if err != nil {
		return fmt.Errorf("could not retrieve existing cert from keystore: %v", err)
	}

	// check if existing cert needs to be renewed
	if !crypto.CheckIfCertNeedsUpdate(existingCert, 21) {
		log.Infof("cert for domain %s still has %.2f day(s) remaining", cn, crypto.DaysRemainingUntilCertExpires(existingCert))
		return nil
	}

	// TODO: context should probably have a timeout
	ctx := context.Background()

	// Get the appropriate DNS solver
	var dnsSolver acmez.Solver
	dnsProvider := viper.GetString("acme.dns_provider")
	switch dnsProvider {
	case "digitalocean":
		dnsSolver = getDigitaloceanDnsSolver()
	case "acmedns":
		dnsSolver = getACMEDNSSolver()
	case "powerdns":
		dnsSolver = getPowerDNSSolver()
	default:
		dnsSolver = getDigitaloceanDnsSolver()
	}

	// Create HTTP client with optional TLS skip verify
	httpClient := &http.Client{}
	if viper.GetBool("acme.tls_insecure_skip_verify") {
		log.Warn("ACME TLS verification is disabled - this should only be used for testing!")
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	client := acmez.Client{
		Client: &acme.Client{
			Directory:  viper.GetString("acme.directory"),
			Logger:     slog.Default(),
			HTTPClient: httpClient,
		},
		ChallengeSolvers: map[string]acmez.Solver{
			acme.ChallengeTypeDNS01: dnsSolver,
		},
	}

	// attempt to reuse existing account
	account, err := retrieveAcmeAccount(ks, &client)
	if err != nil {
		return
	}

	// we should create a new private key each time we renew our cert
	certPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating certificate key: %v", err)
	}

	// Build list of SANs (acmez v3 requires Subject Alt Names, not identifiers)
	var allSANs []string
	allSANs = append(allSANs, cn)

	if sans != nil && len(sans) > 0 {
		allSANs = append(allSANs, sans...)
		log.Infof("requesting certificate for %s with SANs: %v", cn, sans)
	}

	certs, err := client.ObtainCertificateForSANs(ctx, *account, certPrivateKey, allSANs)
	if err != nil || len(certs) == 0 {
		// Update certificate status to failed
		errMsg := fmt.Sprintf("%v", err)

		// Get existing certificate to update status
		existingCert, getErr := ks.GetCertAndKey(cn)
		if getErr == nil && existingCert.CommonName != "" {
			existingCert.Status = "failed"
			existingCert.Error = errMsg
			if updateErr := ks.StoreCertAndKey(cn, existingCert); updateErr != nil {
				log.Errorf("failed to update certificate status: %v", updateErr)
			}
		}

		return fmt.Errorf("obtaining certificate: %v", err)
	}

	// now that we successfully obtained a new cert, we should store it
	// NOTE: let's only store the first full chain certs[0].ChainPEM.
	//       The second one is the shorter chain we shouldn't need it...
	certPrivateKeyPEM, _ := crypto.EncodePrivateKey(certPrivateKey)
	data, err := keystore.NewCertAndKey(string(certs[0].ChainPEM), certPrivateKeyPEM, certs[0].URL)
	if err != nil {
		return fmt.Errorf("could not populate cert for domain %s: %v", cn, err)
	}

	err = ks.StoreCertAndKey(cn, data)
	if err != nil {
		return fmt.Errorf("could not store certificate and key for domain %s: %v", cn, err)
	}

	return
}
