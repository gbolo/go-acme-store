//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// Test domain - using example.com which Pebble will validate
var testDomain = "test.example.com"
var testDomainWildcard = "*.test.example.com"

func TestMain(m *testing.M) {
	// Wait for API to be ready
	fmt.Println("Waiting for API to be ready...")
	if !waitForAPI() {
		fmt.Println("API failed to start in time")
		os.Exit(1)
	}
	fmt.Println("API is ready, running tests...")

	// Run tests
	code := m.Run()

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

	domains := result["domains"].([]interface{})
	count := int(result["count"].(float64))

	assert.GreaterOrEqual(t, count, 0)
	assert.Len(t, domains, count)
}

func TestAddDomain(t *testing.T) {
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

	assert.Equal(t, 201, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Contains(t, result["message"], testDomain)
	assert.Equal(t, testDomain, result["domain"])
}

func TestAddDomainWithSANs(t *testing.T) {
	domain := "san-test.example.com"

	payload := map[string]interface{}{
		"domain": domain,
		"sans":   []string{"www.san-test.example.com", "api.san-test.example.com"},
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
	defer removeDomain(t, domain, false)
}

func TestListDomainsAfterAdd(t *testing.T) {
	// Ensure test domain is added
	addDomain(t, testDomain, nil)

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
	// Add domain to ensure at least one certificate
	addDomain(t, testDomain, nil)

	// Trigger renewal to ensure certificate attempt
	triggerRenewal(t)

	// Wait a bit for certificate attempt
	time.Sleep(2 * time.Second)

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
	domain := "remove-test.example.com"

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

	domains := result["domains"].([]interface{})
	assert.NotContains(t, domains, domain)
}

func TestRemoveDomainHard(t *testing.T) {
	domain := "delete-test.example.com"

	// Add domain first
	addDomain(t, domain, nil)

	// Hard delete
	removeDomain(t, domain, true)

	// Verify it's not in managed domains
	resp, err := http.Get(baseURL + "/api/domains")
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	domains := result["domains"].([]interface{})
	assert.NotContains(t, domains, domain)

	// Should also not be in certificates list (or marked as deleted)
	certResp, err := http.Get(baseURL + "/api/certs")
	require.NoError(t, err)
	defer certResp.Body.Close()

	var certs []map[string]interface{}
	err = json.NewDecoder(certResp.Body).Decode(&certs)
	require.NoError(t, err)

	// Domain should not exist in certs
	for _, cert := range certs {
		assert.NotEqual(t, domain, cert["common_name"])
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

	// Accept 201 (created) or 400 (already exists)
	assert.Contains(t, []int{201, 400}, resp.StatusCode)
}

func removeDomain(t *testing.T, domain string, deleteCert bool) {
	url := fmt.Sprintf("%s/api/domains/%s", baseURL, domain)
	if deleteCert {
		url += "?delete_cert=true"
	}

	req, err := http.NewRequest("DELETE", url, nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Accept 200 (deleted) or 404 (not found)
	assert.Contains(t, []int{200, 404}, resp.StatusCode)
}

func triggerRenewal(t *testing.T) {
	resp, err := http.Post(baseURL+"/api/trigger-renewal", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 202, resp.StatusCode)
}
