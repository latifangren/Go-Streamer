package streamer

import (
	"io"
	"strings"
	"testing"
)

func TestEmbeddedFS_Open(t *testing.T) {
	fsys := GetFileSystem()
	if fsys == nil {
		t.Fatalf("GetFileSystem returned nil")
	}

	// 1. Buka index.html yang ada
	f, err := fsys.Open("/index.html")
	if err != nil {
		t.Fatalf("expected to open /index.html successfully, got err: %v", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(content)), "go-streamer") {
		t.Errorf("expected index.html to contain 'go-streamer', got: %s", string(content))
	}

	// 2. SPA fallback untuk route browser sembarang
	spaRoutes := []string{
		"/dashboard",
		"/slots/1",
		"/settings/alerts",
		"/unknown/page",
	}

	for _, route := range spaRoutes {
		spaFile, err := fsys.Open(route)
		if err != nil {
			t.Errorf("expected SPA route %s to fallback to index.html, got err: %v", route, err)
			continue
		}
		spaContent, err := io.ReadAll(spaFile)
		spaFile.Close()
		if err != nil {
			t.Errorf("failed to read fallback file for %s: %v", route, err)
			continue
		}
		if !strings.Contains(strings.ToLower(string(spaContent)), "go-streamer") {
			t.Errorf("expected route %s fallback to contain 'go-streamer'", route)
		}
	}

	// 3. Request API tidak boleh di-fallback ke index.html
	apiRoutes := []string{
		"/api/v1/health",
		"/api/v1/slots",
		"api/v1/metrics",
	}

	for _, apiRoute := range apiRoutes {
		_, err := fsys.Open(apiRoute)
		if err == nil {
			t.Errorf("expected API route %s to fail and NOT fallback to index.html, but it succeeded", apiRoute)
		}
	}
}
