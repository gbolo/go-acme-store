package acme

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"go-acme-store/pkg/crypto"
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"
	"strings"

	"github.com/mholt/acmez"
	"github.com/mholt/acmez/acme"
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

	client := acmez.Client{
		Client: &acme.Client{
			Directory: viper.GetString("acme.directory"),
			Logger:    log.Base,
		},
		ChallengeSolvers: map[string]acmez.Solver{
			// TODO: in the future we will probably have other solvers
			//       so this needs to be adjustable
			acme.ChallengeTypeDNS01: getDigitaloceanDnsSolver(),
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

	certs, err := client.ObtainCertificate(ctx, *account, certPrivateKey, []string{cn})
	if err != nil || len(certs) == 0 {
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
