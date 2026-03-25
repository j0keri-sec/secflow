package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/secflow/client/pkg/vulngrabber"
)

func TestVulnGrabberRegistry(t *testing.T) {
	// Test Available() returns non-empty list
	sources := vulngrabber.Available()
	assert.NotEmpty(t, sources)
	assert.Contains(t, sources, "avd")
	assert.Contains(t, sources, "seebug")
	assert.Contains(t, sources, "nvd")
}

func TestVulnGrabberByName(t *testing.T) {
	// Test getting existing grabbers
	testCases := []struct {
		name        string
		source      string
		wantDisplay string
	}{
		{"AVD", "avd", "AVD"},
		{"Seebug", "seebug", "Seebug"},
		{"NVD", "nvd", "NVD"},
		{"KEV", "kev", "KEV"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g, err := vulngrabber.ByName(tc.source)
			require.NoError(t, err)
			require.NotNil(t, g)

			provider := g.ProviderInfo()
			require.NotNil(t, provider)
			assert.Equal(t, tc.wantDisplay, provider.DisplayName)
		})
	}
}

func TestVulnGrabberByNameNotFound(t *testing.T) {
	_, err := vulngrabber.ByName("nonexistent-source")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVulnGrabberProviderInfo(t *testing.T) {
	// Test all grabbers have valid ProviderInfo
	sources := vulngrabber.Available()
	for _, source := range sources {
		g, err := vulngrabber.ByName(source)
		require.NoError(t, err, "failed to get grabber for %s", source)

		provider := g.ProviderInfo()
		assert.NotNil(t, provider, "ProviderInfo nil for %s", source)
		assert.NotEmpty(t, provider.Name, "ProviderInfo.Name empty for %s", source)
		assert.NotEmpty(t, provider.DisplayName, "ProviderInfo.DisplayName empty for %s", source)
	}
}

func TestVulnGrabberInterface(t *testing.T) {
	// Ensure VulnGrabber implements the expected interface
	sources := vulngrabber.Available()
	for _, source := range sources {
		g, err := vulngrabber.ByName(source)
		require.NoError(t, err)

		// Test GetUpdate method exists and can be called
		ctx := context.Background()
		vulns, err := g.GetUpdate(ctx, 1)

		// We don't assert no error because network may be unavailable
		// but we verify the method exists and returns the expected type
		if err == nil {
			assert.IsType(t, []*vulngrabber.VulnInfo{}, vulns)
		} else {
			// Network error is acceptable in test environment
			t.Logf("GetUpdate for %s failed (network issue?): %v", source, err)
		}
	}
}

func TestVulnInfoStructure(t *testing.T) {
	// Test VulnInfo structure has expected fields
	vuln := &vulngrabber.VulnInfo{
		UniqueKey:    "CVE-2024-1234",
		Title:        "Test Vulnerability",
		Description:  "A test vulnerability description",
		Severity:     "HIGH",
		CVE:          "CVE-2024-1234",
		Disclosure:   "2024-01-15",
		Solutions:    "Upgrade to version 2.0",
		References:   []string{"https://example.com/advisory"},
		Tags:         []string{"remote", "code-execution"},
		GithubSearch: []string{"example/vulnerable-repo"},
		From:         "https://example.com/cve-2024-1234",
	}

	assert.Equal(t, "CVE-2024-1234", vuln.UniqueKey)
	assert.Equal(t, "Test Vulnerability", vuln.Title)
	assert.Equal(t, "HIGH", vuln.Severity)
	assert.Equal(t, "CVE-2024-1234", vuln.CVE)
	assert.Len(t, vuln.References, 1)
	assert.Len(t, vuln.Tags, 2)
	assert.Len(t, vuln.GithubSearch, 1)
}

func TestVulnInfoSeverity(t *testing.T) {
	// Test severity values
	validSeverities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", ""}
	for _, sev := range validSeverities {
		vuln := &vulngrabber.VulnInfo{
			UniqueKey: "test",
			Severity:  vulngrabber.SeverityLevel(sev),
		}
		assert.Equal(t, sev, vuln.Severity)
	}
}

func TestAllSourcesRegistered(t *testing.T) {
	// Verify all documented sources are registered
	knownSources := []string{
		"avd",
		"seebug",
		"nvd",
		"kev",
		"threatbook",
		"chaitin",
		"oscs",
		"venustech",
		"struts2",
		"exploitdb",
		"packetstorm",
		"vulhub",
		"ti",
		"nox",
		"cn_sources",
	}

	available := vulngrabber.Available()
	for _, source := range knownSources {
		found := false
		for _, a := range available {
			if a == source {
				found = true
				break
			}
		}
		assert.True(t, found, "source %q should be registered", source)
	}
}

func TestGrabberDeduplication(t *testing.T) {
	// Test that grabbers can be used for deduplication
	ctx := context.Background()

	// Get the same grabber twice
	g1, err := vulngrabber.ByName("kev")
	require.NoError(t, err)

	g2, err := vulngrabber.ByName("kev")
	require.NoError(t, err)

	// Both should return the same instance type
	vulns1, err := g1.GetUpdate(ctx, 1)
	vulns2, err := g2.GetUpdate(ctx, 1)

	// If both succeed, they should have the same structure
	if err == nil {
		assert.Equal(t, len(vulns1), len(vulns2))
	}
}

func TestVulnGrabberSourcesList(t *testing.T) {
	// Test the sources list contains expected entries
	sources := vulngrabber.Available()

	// Should have at least 15 sources as documented
	assert.GreaterOrEqual(t, len(sources), 15)

	// Verify some key sources
	keySources := map[string]bool{
		"avd": true,
		"seebug": true,
		"nvd": true,
		"kev": true,
	}

	found := 0
	for _, s := range sources {
		if keySources[s] {
			found++
		}
	}
	assert.Equal(t, len(keySources), found, "not all key sources found")
}
