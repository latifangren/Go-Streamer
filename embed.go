package streamer

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:web/dist
var WebFS embed.FS

// EmbeddedFS membungkus embed.FS dan mengimplementasikan http.FileSystem
// dengan dukungan Single Page Application (SPA) fallback ke index.html.
type EmbeddedFS struct {
	fs embed.FS
}

// Open membuka file dari sub-filesystem web/dist. Jika file tidak ditemukan
// dan request bukan untuk endpoint API (/api/...), maka otomatis fallback ke index.html.
func (e *EmbeddedFS) Open(name string) (http.File, error) {
	cleanName := path.Clean(name)
	trimmed := strings.TrimPrefix(cleanName, "/")

	isAPI := strings.HasPrefix(cleanName, "/api") || strings.HasPrefix(trimmed, "api")

	sub, err := fs.Sub(e.fs, "web/dist")
	if err != nil {
		return nil, err
	}
	httpSub := http.FS(sub)

	// Coba buka file yang diminta
	f, err := httpSub.Open(name)
	if err == nil {
		stat, statErr := f.Stat()
		if statErr == nil && stat.IsDir() {
			// Jika membuka direktori, buka index.html di dalamnya jika ada
			indexPath := path.Join(name, "index.html")
			indexFile, indexErr := httpSub.Open(indexPath)
			if indexErr == nil {
				_ = f.Close()
				return indexFile, nil
			}
		}
		return f, nil
	}

	// SPA Fallback: jika bukan request API dan file tidak ada, sajikan /index.html
	if !isAPI {
		indexFile, indexErr := httpSub.Open("/index.html")
		if indexErr == nil {
			return indexFile, nil
		}
	}

	return nil, err
}

// GetFileSystem mengembalikan http.FileSystem yang telah dikonfigurasi dengan SPA fallback.
func GetFileSystem() http.FileSystem {
	return &EmbeddedFS{fs: WebFS}
}
