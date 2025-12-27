package keystore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func setupFilesystemTest(t *testing.T) (*FilesystemKeystore, string) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "acme-store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Set up viper config
	viper.Reset()
	viper.Set("filesystem.base_path", tempDir)

	// Create keystore
	ks, err := NewFilesystemKeystoreFromViper("test@example.com", true)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create filesystem keystore: %v", err)
	}

	return ks.(*FilesystemKeystore), tempDir
}

func cleanupFilesystemTest(tempDir string) {
	os.RemoveAll(tempDir)
	viper.Reset()
}

func TestFilesystemKeystore_AccountManagement(t *testing.T) {
	ks, tempDir := setupFilesystemTest(t)
	defer cleanupFilesystemTest(tempDir)

	// Test getting account
	account, err := ks.GetAcmeAccount()
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", account.Email)
	}

	if account.Key == "" {
		t.Error("expected non-empty account key")
	}

	// Verify account file was created
	if _, err := os.Stat(ks.accountFile); os.IsNotExist(err) {
		t.Error("account file was not created")
	}
}

func TestFilesystemKeystore_CertOperations(t *testing.T) {
	ks, tempDir := setupFilesystemTest(t)
	defer cleanupFilesystemTest(tempDir)

	domain := "example.com"
	testCert := CertAndKey{
		CommonName:    domain,
		CertChainPEM:  "test-cert-chain",
		LeafCertPEM:   "test-leaf-cert",
		PrivateKeyPEM: "test-private-key",
		Managed:       true,
		Status:        "issued",
	}

	// Test storing certificate
	err := ks.StoreCertAndKey(domain, testCert)
	if err != nil {
		t.Fatalf("failed to store cert: %v", err)
	}

	// Verify domain file was created
	domainFile := filepath.Join(tempDir, "domains", domain+".json")
	if _, err := os.Stat(domainFile); os.IsNotExist(err) {
		t.Error("domain file was not created")
	}

	// Test retrieving certificate
	retrieved, err := ks.GetCertAndKey(domain)
	if err != nil {
		t.Fatalf("failed to get cert: %v", err)
	}

	if retrieved.CommonName != testCert.CommonName {
		t.Errorf("expected CommonName '%s', got '%s'", testCert.CommonName, retrieved.CommonName)
	}

	if retrieved.CertChainPEM != testCert.CertChainPEM {
		t.Errorf("expected CertChainPEM '%s', got '%s'", testCert.CertChainPEM, retrieved.CertChainPEM)
	}

	// Test getting leaf cert only
	leafCert, err := ks.GetLeafCert(domain)
	if err != nil {
		t.Fatalf("failed to get leaf cert: %v", err)
	}

	if leafCert != testCert.LeafCertPEM {
		t.Errorf("expected leaf cert '%s', got '%s'", testCert.LeafCertPEM, leafCert)
	}

	// Test GetAllDomains
	domains, err := ks.GetAllDomains()
	if err != nil {
		t.Fatalf("failed to get all domains: %v", err)
	}

	if len(domains) != 1 {
		t.Errorf("expected 1 domain, got %d", len(domains))
	}

	if len(domains) > 0 && domains[0] != domain {
		t.Errorf("expected domain '%s', got '%s'", domain, domains[0])
	}

	// Test deleting certificate
	err = ks.DeleteCertAndKey(domain)
	if err != nil {
		t.Fatalf("failed to delete cert: %v", err)
	}

	// Verify domain file was deleted
	if _, err := os.Stat(domainFile); !os.IsNotExist(err) {
		t.Error("domain file was not deleted")
	}

	// Test GetAllDomains after deletion
	domains, err = ks.GetAllDomains()
	if err != nil {
		t.Fatalf("failed to get all domains after deletion: %v", err)
	}

	if len(domains) != 0 {
		t.Errorf("expected 0 domains after deletion, got %d", len(domains))
	}
}

func TestFilesystemKeystore_ManagedDomains(t *testing.T) {
	ks, tempDir := setupFilesystemTest(t)
	defer cleanupFilesystemTest(tempDir)

	domain1 := "example.com"
	domain2 := "test.com"
	sans := []string{"www.example.com", "api.example.com"}

	// Test adding managed domain without SANs
	err := ks.AddManagedDomain(domain1)
	if err != nil {
		t.Fatalf("failed to add managed domain: %v", err)
	}

	// Test adding managed domain with SANs
	err = ks.AddManagedDomainWithSANs(domain2, sans)
	if err != nil {
		t.Fatalf("failed to add managed domain with SANs: %v", err)
	}

	// Verify managed domains file was created
	if _, err := os.Stat(ks.managedFile); os.IsNotExist(err) {
		t.Error("managed domains file was not created")
	}

	// Test getting managed domains
	domains, err := ks.GetManagedDomains()
	if err != nil {
		t.Fatalf("failed to get managed domains: %v", err)
	}

	if len(domains) != 2 {
		t.Errorf("expected 2 managed domains, got %d", len(domains))
	}

	// Test IsManagedDomain
	isManaged, err := ks.IsManagedDomain(domain1)
	if err != nil {
		t.Fatalf("failed to check if domain is managed: %v", err)
	}

	if !isManaged {
		t.Error("expected domain1 to be managed")
	}

	isManaged, err = ks.IsManagedDomain("nonexistent.com")
	if err != nil {
		t.Fatalf("failed to check if nonexistent domain is managed: %v", err)
	}

	if isManaged {
		t.Error("expected nonexistent domain to not be managed")
	}

	// Test adding duplicate domain
	err = ks.AddManagedDomain(domain1)
	if err == nil {
		t.Error("expected error when adding duplicate domain")
	}

	// Test removing managed domain
	err = ks.RemoveManagedDomain(domain1)
	if err != nil {
		t.Fatalf("failed to remove managed domain: %v", err)
	}

	domains, err = ks.GetManagedDomains()
	if err != nil {
		t.Fatalf("failed to get managed domains after removal: %v", err)
	}

	if len(domains) != 1 {
		t.Errorf("expected 1 managed domain after removal, got %d", len(domains))
	}

	// Test removing non-existent domain
	err = ks.RemoveManagedDomain("nonexistent.com")
	if err == nil {
		t.Error("expected error when removing non-existent domain")
	}
}

func TestFilesystemKeystore_ConcurrentAccess(t *testing.T) {
	ks, tempDir := setupFilesystemTest(t)
	defer cleanupFilesystemTest(tempDir)

	domain := "concurrent.com"
	testCert := CertAndKey{
		CommonName:    domain,
		CertChainPEM:  "test-cert",
		PrivateKeyPEM: "test-key",
		Managed:       true,
		Status:        "issued",
	}

	// Test concurrent writes and reads
	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func() {
			err := ks.StoreCertAndKey(domain, testCert)
			if err != nil {
				t.Errorf("concurrent store failed: %v", err)
			}
			done <- true
		}()

		go func() {
			_, err := ks.GetCertAndKey(domain)
			if err != nil {
				// It's okay if the cert doesn't exist yet
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	retrieved, err := ks.GetCertAndKey(domain)
	if err != nil {
		t.Fatalf("failed to get cert after concurrent access: %v", err)
	}

	if retrieved.CommonName != testCert.CommonName {
		t.Errorf("expected CommonName '%s', got '%s'", testCert.CommonName, retrieved.CommonName)
	}
}

func TestFilesystemKeystore_EmptyDirectory(t *testing.T) {
	ks, tempDir := setupFilesystemTest(t)
	defer cleanupFilesystemTest(tempDir)

	// Test GetAllDomains with no domains
	domains, err := ks.GetAllDomains()
	if err != nil {
		t.Fatalf("failed to get all domains: %v", err)
	}

	if len(domains) != 0 {
		t.Errorf("expected 0 domains in empty directory, got %d", len(domains))
	}

	// Test GetManagedDomains with no managed domains
	managedDomains, err := ks.GetManagedDomains()
	if err != nil {
		t.Fatalf("failed to get managed domains: %v", err)
	}

	if len(managedDomains) != 0 {
		t.Errorf("expected 0 managed domains, got %d", len(managedDomains))
	}

	// Test GetCertAndKey for non-existent domain
	cert, err := ks.GetCertAndKey("nonexistent.com")
	if err != nil {
		t.Fatalf("expected no error for non-existent domain, got: %v", err)
	}

	if cert.CommonName != "" {
		t.Error("expected empty cert for non-existent domain")
	}
}

