package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckForUpdatesFromURL_NewerRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(githubReleasePayload{
			TagName:     "v9.9.9",
			HTMLURL:     "https://github.com/Rafifdev/Velora/releases/tag/v9.9.9",
			Body:        "Bug fixes and faster updates",
			PublishedAt: "2026-11-26T10:00:00Z",
			Assets: []githubReleaseAsset{
				{Name: "Velora-Setup.exe", BrowserDownloadURL: "https://github.com/Rafifdev/Velora/releases/download/v9.9.9/Velora-Setup.exe"},
				{Name: "Velora.exe", BrowserDownloadURL: "https://github.com/Rafifdev/Velora/releases/download/v9.9.9/Velora.exe"},
			},
		})
	}))
	defer server.Close()

	res, err := checkForUpdatesFromURL(server.URL)
	if err != nil {
		t.Fatalf("checkForUpdatesFromURL failed: %v", err)
	}
	if !res.HasUpdate {
		t.Fatalf("expected HasUpdate=true, got false")
	}
	if res.LatestVersion != "9.9.9" {
		t.Fatalf("expected LatestVersion=9.9.9, got %q", res.LatestVersion)
	}
	expectedURL := "https://github.com/Rafifdev/Velora/releases/download/v9.9.9/Velora.exe"
	if res.DownloadURL != expectedURL {
		t.Fatalf("expected portable Velora.exe asset %q, got %q", expectedURL, res.DownloadURL)
	}
	if res.StoreURL == "" {
		t.Fatalf("expected non-empty StoreURL")
	}
}

func TestCheckForUpdatesFromURL_UpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(githubReleasePayload{
			TagName: "v" + veloraAppVersion,
			HTMLURL: "https://github.com/Rafifdev/Velora/releases/tag/v" + veloraAppVersion,
		})
	}))
	defer server.Close()

	res, err := checkForUpdatesFromURL(server.URL)
	if err != nil {
		t.Fatalf("checkForUpdatesFromURL failed: %v", err)
	}
	if res.HasUpdate {
		t.Fatalf("expected HasUpdate=false when version matches %s", veloraAppVersion)
	}
}

func TestSelectWindowsUpdateAsset_FallbackSetup(t *testing.T) {
	url := selectWindowsUpdateAsset([]githubReleaseAsset{
		{Name: "Velora-1.3.0-amd64.msix", BrowserDownloadURL: "https://github.com/Rafifdev/Velora/releases/download/v1.3.0/Velora.msix"},
		{Name: "Velora-Setup.exe", BrowserDownloadURL: "https://github.com/Rafifdev/Velora/releases/download/v1.3.0/Velora-Setup.exe"},
	})
	if url != "https://github.com/Rafifdev/Velora/releases/download/v1.3.0/Velora-Setup.exe" {
		t.Fatalf("expected fallback to Velora-Setup.exe, got %q", url)
	}
}

func TestValidateUpdateDownloadURL(t *testing.T) {
	valid := []string{
		"https://github.com/Rafifdev/Velora/releases/download/v1.3.0/Velora.exe",
		"https://objects.githubusercontent.com/github-production-release-asset/Velora.exe",
	}
	for _, u := range valid {
		if err := validateUpdateDownloadURL(u); err != nil {
			t.Errorf("expected %q to be valid, got %v", u, err)
		}
	}

	invalid := []string{
		"",
		"http://github.com/Rafifdev/Velora/releases/download/v1.3.0/Velora.exe",
		"https://evil.example.com/Velora.exe",
		"ms-windows-store://pdp/?ProductId=9PN43QQ19SS9",
	}
	for _, u := range invalid {
		if err := validateUpdateDownloadURL(u); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}
}

func TestVerifyPEExecutable(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.exe")
	bad := filepath.Join(dir, "bad.exe")
	if err := os.WriteFile(good, []byte("MZ\x90\x00test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("<html>404</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyPEExecutable(good); err != nil {
		t.Errorf("expected good PE header to pass, got %v", err)
	}
	if err := verifyPEExecutable(bad); err == nil {
		t.Errorf("expected non-MZ file to fail")
	}
}
