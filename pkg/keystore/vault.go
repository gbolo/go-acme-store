package keystore

import (
	"encoding/json"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go-acme-store/pkg/crypto"
	"go-acme-store/pkg/log"
)

type VaultKeystore struct {
	client     *vault.Client
	mountPath  string
	secretPath string
}

func (v *VaultKeystore) getBasePath() string {
	return fmt.Sprintf("%s/data/%s", v.mountPath, v.secretPath)
}

func (v *VaultKeystore) getAccountPath() string {
	return fmt.Sprintf("%s/account", v.getBasePath())
}

func (v *VaultKeystore) getDomainListPath() string {
	return fmt.Sprintf("%s/metadata/%s/domain", v.mountPath, v.secretPath)
}

func (v *VaultKeystore) getDomainPath(domain string) string {
	return fmt.Sprintf("%s/domain/%s", v.getBasePath(), domain)
}

func (v *VaultKeystore) storeAccount(account AcmeAccount) (err error) {
	vaultData, err := encodeForVault(account)
	if err != nil {
		return
	}
	result, err := v.client.Logical().Write(v.getAccountPath(), vaultData)
	if err != nil {
		log.Infof("stored account info for %s -> %s version %v", account.Email, v.getAccountPath(), result.Data["version"])
	}
	return
}

func (v *VaultKeystore) getAccount() (account *AcmeAccount, err error) {
	err = v.readDataIntoInterface(v.getAccountPath(), &account)
	return
}

func (v *VaultKeystore) readDataIntoInterface(path string, destInterface interface{}) (err error) {
	vaultData, err := v.client.Logical().Read(path)
	if err != nil {
		return
	}
	if vaultData == nil {
		log.Debugf("no secrets found in path %s", path)
		return
	}

	if _, exists := vaultData.Data["data"]; !exists {
		log.Debugf("no data found in secret path %s", path)
		return
	}

	log.Debugf("loaded data from secret path %s", path)
	err = mapstructure.Decode(vaultData.Data["data"], &destInterface)
	return
}

func NewVaultKeystoreFromViper(accountEmail string) (Keystore, error) {
	config := vault.DefaultConfig()
	config.Address = viper.GetString("vault.address")
	client, _ := vault.NewClient(config)

	// set a token only if one is specified
	if viper.GetString("vault.token") != "" {
		client.SetToken("vault.token")
	}

	ks := VaultKeystore{
		client:     client,
		mountPath:  viper.GetString("vault.kv2_mount"),
		secretPath: viper.GetString("vault.kv2_secret_path"),
	}

	// attempt to reuse existing acme account
	storedAccount, err := ks.getAccount()
	if err != nil {
		return nil, err
	}
	if storedAccount != nil {
		log.Infof("loaded saved acme account %s from %s", storedAccount.Email, ks.getAccountPath())
		return &ks, err
	}

	// create a new account key because one does not exist
	accountKeyPEM, err := crypto.GenerateECKeyPEM()
	if err != nil {
		return nil, err
	}

	err = ks.storeAccount(AcmeAccount{
		Email: accountEmail,
		Key:   accountKeyPEM,
	})
	if err == nil {
		log.Infof("account pki generated and stored for %s", accountEmail)
	}
	return &ks, err
}

func (v *VaultKeystore) GetAcmeAccount() (account AcmeAccount, err error) {
	accountPtr, err := v.getAccount()
	if accountPtr != nil {
		account = *accountPtr
	}
	return
}

func (v *VaultKeystore) GetAllDomains() (domains []string, err error) {
	result, err := v.client.Logical().List(v.getDomainListPath())
	if err != nil {
		return
	}

	if _, exists := result.Data["keys"]; !exists {
		return
	}

	for _, domain := range result.Data["keys"].([]interface{}) {
		domains = append(domains, fmt.Sprintf("%v", domain))
	}
	return
}

func (v *VaultKeystore) GetCertAndKey(domain string) (data CertAndKey, err error) {
	err = v.readDataIntoInterface(v.getDomainPath(domain), &data)
	return
}

func (v *VaultKeystore) GetLeafCert(domain string) (cert string, err error) {
	data, err := v.GetCertAndKey(domain)
	if err != nil {
		return
	}
	cert = data.LeafCertPEM
	return
}

func (v *VaultKeystore) StoreCertAndKey(domain string, data CertAndKey) (err error) {
	vaultData, err := encodeForVault(data)
	if err != nil {
		return
	}
	result, err := v.client.Logical().Write(v.getDomainPath(domain), vaultData)
	if err != nil {
		log.Infof("stored cert and key for %s -> %s version %v", domain, v.getDomainPath(domain), result.Data["version"])
	}
	return
}

func encodeForVault(input interface{}) (output map[string]interface{}, err error) {
	var inputData map[string]interface{}
	dataBytes, err := json.Marshal(input)
	err = json.Unmarshal(dataBytes, &inputData)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %v", err)
	}
	output = make(map[string]interface{})
	output["data"] = inputData
	return
}
