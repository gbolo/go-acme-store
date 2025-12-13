//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL     = "http://127.0.0.1:15872"
	testTimeout = 30 * time.Second
)

// testZone is the DNS zone configured in PowerDNS for testing
var testZone = "acme.test"

// Test domains within our test zone
var testDomain = "test." + testZone
var testDomainWildcard = "*." + testZone
var testDomain2 = "test2." + testZone

func TestMain(m *testing.M) {
	// Wait for API to be ready
	fmt.Println("Waiting for API to be ready...")
	if !waitForAPI() {
		fmt.Println("API failed to start in time")
		os.Exit(1)
	}
	fmt.Println("API is ready, running tests...")

	// Verify environment is clean before running tests
	fmt.Println("Verifying clean test environment...")
	if !verifyNoDomains() {
		fmt.Println("❌ FAIL: Test environment is not clean. Existing domains found.")
		fmt.Println("Please clean up existing domains before running tests.")
		fmt.Println("Run: make test-cleanup")
		os.Exit(1)
	}
	if !verifyNoCerts() {
		fmt.Println("❌ FAIL: Test environment is not clean. Existing certificates found.")
		fmt.Println("Please clean up existing certificates before running tests.")
		fmt.Println("Run: make test-cleanup")
		os.Exit(1)
	}
	fmt.Println("✓ Test environment is clean")

	// Run tests
	code := m.Run()

	// Final cleanup after all tests
	fmt.Println("Final cleanup after all tests...")
	cleanupAllDomains()

	os.Exit(code)
}

func waitForAPI() bool {
	start := time.Now()
	for time.Since(start) < testTimeout {
		resp, err := http.Get(baseURL + "/api/healthz")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "success", result["status"])
}

func TestVersion(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/version")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Version endpoint returns direct fields with lowercase first letter
	assert.NotNil(t, result["app_name"])
	assert.NotNil(t, result["app_version"])
}

func TestConfig(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/config")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify config structure
	assert.NotNil(t, result["log"])
	assert.NotNil(t, result["server"])
	assert.NotNil(t, result["acme"])
	assert.NotNil(t, result["vault"])

	// Verify sensitive values are not exposed
	vault := result["vault"].(map[string]interface{})
	assert.NotContains(t, vault, "token")
}

func TestListDomainsInitiallyEmpty(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/domains")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	count := int(result["count"].(float64))
	assert.Equal(t, 0, count, "domain list should be empty at start of tests")

	// domains field may be null or empty array when count is 0
	if result["domains"] != nil {
		domains := result["domains"].([]interface{})
		assert.Len(t, domains, 0, "domains array should be empty")
	}
}

func TestAddDomain(t *testing.T) {
	// Ensure clean state
	removeDomain(t, testDomain, true)
	time.Sleep(500 * time.Millisecond)

	// Add a domain
	payload := map[string]interface{}{
		"domain": testDomain,
	}

	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(
		baseURL+"/api/domains",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 201 for new domain
	assert.Equal(t, 201, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Contains(t, result["message"], testDomain)
	assert.Equal(t, testDomain, result["domain"])

	// Cleanup
	defer removeDomain(t, testDomain, true)
}

func TestAddDomainAlreadyExists(t *testing.T) {
	// Add domain first time
	addDomain(t, testDomain, nil)
	defer removeDomain(t, testDomain, true)

	// Try to add the same domain again
	payload := map[string]interface{}{
		"domain": testDomain,
	}

	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(
		baseURL+"/api/domains",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 for duplicate domain
	assert.Equal(t, 400, resp.StatusCode)
}

func TestAddDomainWithSANs(t *testing.T) {
	domain := "san-test." + testZone

	// Ensure clean state
	removeDomain(t, domain, true)
	time.Sleep(500 * time.Millisecond)

	payload := map[string]interface{}{
		"domain": domain,
		"sans":   []string{"www.san-test." + testZone, "api.san-test." + testZone},
	}

	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(
		baseURL+"/api/domains",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, domain, result["domain"])
	sans := result["sans"].([]interface{})
	assert.Len(t, sans, 2)

	// Clean up
	defer removeDomain(t, domain, true)
}

func TestListDomainsAfterAdd(t *testing.T) {
	// Ensure clean state
	removeDomain(t, testDomain, true)
	time.Sleep(500 * time.Millisecond)

	// Add test domain
	addDomain(t, testDomain, nil)
	defer removeDomain(t, testDomain, true)

	resp, err := http.Get(baseURL + "/api/domains")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	domains := result["domains"].([]interface{})
	count := int(result["count"].(float64))

	assert.Greater(t, count, 0)
	assert.Contains(t, domains, testDomain)
}

func TestListCertificates(t *testing.T) {
	// Ensure clean state
	removeDomain(t, testDomain, true)
	time.Sleep(500 * time.Millisecond)

	// Add domain to ensure at least one certificate
	addDomain(t, testDomain, nil)
	defer removeDomain(t, testDomain, true)

	// Trigger renewal to ensure certificate attempt
	triggerRenewal(t)

	// Wait for certificate to be issued
	cert := waitForCertificateStatus(t, testDomain, "issued", 60*time.Second)
	require.NotNil(t, cert, "certificate should be issued within timeout")

	// Verify certificate was issued by Pebble
	issuers := cert["issuers"].([]interface{})
	require.NotEmpty(t, issuers, "certificate should have issuers")

	firstIssuer := issuers[0].(string)
	assert.Contains(t, firstIssuer, "Pebble", "certificate should be issued by Pebble")

	t.Logf("✅ Certificate issued by: %s", firstIssuer)

	resp, err := http.Get(baseURL + "/api/certs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var certs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&certs)
	require.NoError(t, err)

	// Should have at least one certificate (even if pending/failed)
	assert.GreaterOrEqual(t, len(certs), 1)

	// Find our test domain
	var testCert map[string]interface{}
	for _, cert := range certs {
		if cert["common_name"] == testDomain {
			testCert = cert
			break
		}
	}

	if testCert != nil {
		// Verify certificate structure
		assert.NotNil(t, testCert["status"])
		assert.NotNil(t, testCert["managed"])

		// Status should be one of: pending, issued, failed
		status := testCert["status"].(string)
		assert.Contains(t, []string{"pending", "issued", "failed"}, status)

		t.Logf("Certificate status for %s: %s", testDomain, status)

		if status == "failed" {
			t.Logf("Error: %v", testCert["error"])
		}
	}
}

func TestTriggerRenewal(t *testing.T) {
	resp, err := http.Post(baseURL+"/api/trigger-renewal", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 202, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Contains(t, result["message"], "triggered")
}

func TestRemoveDomainSoft(t *testing.T) {
	domain := "soft-delete-test." + testZone

	// Add domain first
	addDomain(t, domain, nil)

	// Soft delete (unmanage)
	removeDomain(t, domain, false)

	// Verify it's not in managed domains list
	resp, err := http.Get(baseURL + "/api/domains")
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Check if domains field exists and is not nil
	if result["domains"] != nil {
		domains := result["domains"].([]interface{})
		assert.NotContains(t, domains, domain)
	}

	// Final cleanup
	removeDomain(t, domain, true)
}

func TestDeleteUnmanagedCert(t *testing.T) {
	domain := "delete-unmanaged-test." + testZone

	// Add domain first
	addDomain(t, domain, nil)
	time.Sleep(500 * time.Millisecond)

	// Try to delete managed certificate (should fail)
	req, err := http.NewRequest("DELETE", baseURL+"/api/certs/"+domain, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode, "should not be able to delete managed certificate")

	// Soft delete (unmanage) the domain
	removeDomain(t, domain, false)
	time.Sleep(500 * time.Millisecond)

	// Verify certificate is unmanaged
	resp2, err := http.Get(baseURL + "/api/certs")
	require.NoError(t, err)
	defer resp2.Body.Close()

	var certs []map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&certs)
	require.NoError(t, err)

	found := false
	for _, cert := range certs {
		if cn, ok := cert["common_name"].(string); ok && cn == domain {
			found = true
			managed, _ := cert["managed"].(bool)
			assert.False(t, managed, "certificate should be unmanaged")
			break
		}
	}
	assert.True(t, found, "certificate should exist after soft delete")

	// Now delete the unmanaged certificate (should succeed)
	req2, err := http.NewRequest("DELETE", baseURL+"/api/certs/"+domain, nil)
	require.NoError(t, err)

	resp3, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp3.Body.Close()

	if resp3.StatusCode != 200 {
		body, _ := io.ReadAll(resp3.Body)
		t.Fatalf("expected status 200 when deleting unmanaged cert, got %d: %s", resp3.StatusCode, string(body))
	}

	// Verify certificate is completely deleted
	resp4, err := http.Get(baseURL + "/api/certs")
	require.NoError(t, err)
	defer resp4.Body.Close()

	var certsAfter []map[string]interface{}
	err = json.NewDecoder(resp4.Body).Decode(&certsAfter)
	require.NoError(t, err)

	for _, cert := range certsAfter {
		if cn, ok := cert["common_name"].(string); ok && cn == domain {
			t.Fatalf("certificate for %s still exists after deletion", domain)
		}
	}
}

func TestInvalidDomainRejection(t *testing.T) {
	// Try to add invalid domain
	payload := map[string]interface{}{
		"domain": "", // Empty domain
	}

	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(
		baseURL+"/api/domains",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)
}

func TestCertificateIssuance(t *testing.T) {
	// Test domain within our zone
	domain := "cert-test." + testZone

	// Clean up any existing certificate
	removeDomain(t, domain, true)
	time.Sleep(1 * time.Second)

	// Add domain
	addDomain(t, domain, nil)

	// Trigger renewal
	triggerRenewal(t)

	// Wait for certificate to be issued
	t.Logf("⏳ Waiting for certificate issuance for %s...", domain)
	cert := waitForCertificateStatus(t, domain, "issued", 60*time.Second)
	require.NotNil(t, cert, "certificate should be issued within 60 seconds")

	// Verify certificate details
	assert.Equal(t, domain, cert["common_name"], "common name should match domain")
	assert.Equal(t, "issued", cert["status"], "status should be issued")
	assert.Equal(t, true, cert["managed"], "certificate should be managed")

	// Verify Pebble issued the certificate
	issuers := cert["issuers"].([]interface{})
	require.NotEmpty(t, issuers, "certificate should have issuers")

	firstIssuer := issuers[0].(string)
	assert.Contains(t, firstIssuer, "Pebble", "certificate should be issued by Pebble")

	// Verify expiration dates exist
	assert.NotEmpty(t, cert["issued_on"], "issued_on should be set")
	assert.NotEmpty(t, cert["expires_on"], "expires_on should be set")

	// Verify PEM data exists (private key is not exposed via API for security)
	assert.NotEmpty(t, cert["leaf_cert_pem"], "leaf_cert_pem should be present")
	assert.NotEmpty(t, cert["cert_chain_pem"], "cert_chain_pem should be present")

	t.Logf("✅ Certificate successfully issued by: %s", firstIssuer)

	// Clean up
	removeDomain(t, domain, true)
}

func TestCertificateIssuanceWithSANs(t *testing.T) {
	// Test domain with SANs
	domain := "multi." + testZone
	sans := []string{"www.multi." + testZone, "api.multi." + testZone}

	// Clean up any existing certificate
	removeDomain(t, domain, true)
	time.Sleep(1 * time.Second)

	// Add domain with SANs
	addDomain(t, domain, sans)

	// Trigger renewal
	triggerRenewal(t)

	// Wait for certificate to be issued
	t.Logf("⏳ Waiting for certificate issuance for %s with SANs...", domain)
	cert := waitForCertificateStatus(t, domain, "issued", 60*time.Second)
	require.NotNil(t, cert, "certificate with SANs should be issued within 60 seconds")

	// Verify SANs are in the certificate
	certSANs := cert["sans"].([]interface{})
	require.Len(t, certSANs, 3, "certificate should have 3 SANs (CN + 2 additional)")

	// Verify all domains are in SANs
	sanStrings := make([]string, len(certSANs))
	for i, san := range certSANs {
		sanStrings[i] = san.(string)
	}
	assert.Contains(t, sanStrings, domain, "SANs should contain the main domain")
	assert.Contains(t, sanStrings, sans[0], "SANs should contain first SAN")
	assert.Contains(t, sanStrings, sans[1], "SANs should contain second SAN")

	// Verify Pebble issued the certificate
	issuers := cert["issuers"].([]interface{})
	firstIssuer := issuers[0].(string)
	assert.Contains(t, firstIssuer, "Pebble", "certificate should be issued by Pebble")

	t.Logf("✅ Certificate with SANs successfully issued by: %s", firstIssuer)
	t.Logf("   SANs: %v", sanStrings)

	// Clean up
	removeDomain(t, domain, true)
}

func TestRootRedirect(t *testing.T) {
	t.Skip("Skipping frontend tests")
}

func TestUIAccessible(t *testing.T) {
	t.Skip("Skipping frontend tests")
}

// Helper functions

func addDomain(t *testing.T, domain string, sans []string) {
	payload := map[string]interface{}{
		"domain": domain,
	}
	if sans != nil {
		payload["sans"] = sans
	}

	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(
		baseURL+"/api/domains",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should succeed (201 created)
	if resp.StatusCode != 201 {
		var errResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResult)
		t.Logf("Failed to add domain %s: %v", domain, errResult)
	}
	require.Equal(t, 201, resp.StatusCode, "domain should be added successfully")
}

func removeDomain(t *testing.T, domain string, deleteCert bool) {
	url := fmt.Sprintf("%s/api/domains/%s", baseURL, domain)
	if deleteCert {
		url += "?delete_cert=true"
	}

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Logf("Failed to create delete request for %s: %v", domain, err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to delete domain %s: %v", domain, err)
		return
	}
	defer resp.Body.Close()

	// Accept 200 (deleted) or 404 (not found) - both are acceptable for cleanup
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		t.Logf("Unexpected status code %d when deleting domain %s", resp.StatusCode, domain)
	}
}

// verifyNoDomains checks if there are any existing domains and returns false if found
func verifyNoDomains() bool {
	resp, err := http.Get(baseURL + "/api/domains")
	if err != nil {
		fmt.Printf("Failed to get domains: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false
	}

	count, ok := result["count"].(float64)
	if !ok {
		return false
	}

	if count > 0 {
		// List the domains that were found
		if result["domains"] != nil {
			domains := result["domains"].([]interface{})
			fmt.Printf("Found %d existing domain(s): %v\n", int(count), domains)
		}
		return false
	}

	return true
}

// verifyNoCerts checks if there are any existing managed certificates and returns false if found
func verifyNoCerts() bool {
	resp, err := http.Get(baseURL + "/api/certs")
	if err != nil {
		fmt.Printf("Failed to get certificates: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	var certs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&certs)
	if err != nil {
		return false
	}

	if len(certs) > 0 {
		fmt.Printf("Found %d existing certificate(s): %v\n", len(certs), certs)
		return false
	}

	return true
}

// cleanupAllDomains removes all managed domains for test isolation
func cleanupAllDomains() {
	resp, err := http.Get(baseURL + "/api/domains")
	if err != nil {
		fmt.Printf("Failed to get domains for cleanup: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	domains, ok := result["domains"].([]interface{})
	if !ok {
		return
	}

	for _, d := range domains {
		domain := d.(string)
		url := fmt.Sprintf("%s/api/domains/%s?delete_cert=true", baseURL, domain)
		req, _ := http.NewRequest("DELETE", url, nil)
		client := &http.Client{}
		resp, _ := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Printf("Cleaned up domain: %s\n", domain)
	}

	// Wait for cleanup to complete
	time.Sleep(1 * time.Second)
}

func triggerRenewal(t *testing.T) {
	resp, err := http.Post(baseURL+"/api/trigger-renewal", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 202, resp.StatusCode)
}

// waitForCertificateStatus polls the API until the certificate reaches the expected status
func waitForCertificateStatus(t *testing.T, domain, expectedStatus string, timeout time.Duration) map[string]interface{} {
	start := time.Now()
	for time.Since(start) < timeout {
		cert := getCertificate(t, domain)
		if cert != nil {
			status, ok := cert["status"].(string)
			if ok && status == expectedStatus {
				return cert
			}
			// If failed, return immediately
			if ok && status == "failed" {
				t.Logf("❌ Certificate for %s failed: %v", domain, cert["error"])
				return cert
			}
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

// getCertificate retrieves a specific certificate by common name
func getCertificate(t *testing.T, domain string) map[string]interface{} {
	resp, err := http.Get(baseURL + "/api/certs")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var certs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&certs)
	if err != nil {
		return nil
	}

	for _, cert := range certs {
		if cert["common_name"] == domain {
			return cert
		}
	}
	return nil
}
