package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-streamer/internal/domain"
)

// GenerateConcatManifest membuat file manifest playlist ffconcat untuk slot tertentu di cacheDir.
// Manifest ini digunakan oleh FFmpeg Concat Demuxer (-f concat -safe 0) untuk streaming multi-video berkesinambungan.
func GenerateConcatManifest(slotID int64, cacheDir string, videoPaths []string) (string, error) {
	if len(videoPaths) == 0 {
		return "", fmt.Errorf("video paths cannot be empty: %w", domain.ErrInvalidInput)
	}

	cacheDir = strings.TrimSpace(cacheDir)
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	manifestFilename := fmt.Sprintf("concat_slot_%d.txt", slotID)
	manifestPath := filepath.Join(cacheDir, manifestFilename)

	absManifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute manifest path: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("ffconcat version 1.0\n")

	validCount := 0
	for _, p := range videoPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		absVideoPath := p
		if !filepath.IsAbs(p) && !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "\\") {
			var absErr error
			absVideoPath, absErr = filepath.Abs(p)
			if absErr != nil {
				return "", fmt.Errorf("failed to resolve absolute path for %s: %w", p, absErr)
			}
		}

		escapedPath := escapeConcatPath(absVideoPath)
		builder.WriteString(fmt.Sprintf("file '%s'\n", escapedPath))
		validCount++
	}

	if validCount == 0 {
		return "", fmt.Errorf("no valid video paths provided: %w", domain.ErrInvalidInput)
	}

	if err := os.WriteFile(absManifestPath, []byte(builder.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write concat manifest: %w", err)
	}

	return absManifestPath, nil
}

// escapeConcatPath meng-escape karakter khusus di manifest ffconcat.
// Karakter kutip tunggal di-escape sebagai '\'' dan backslash diubah ke slash agar kompatibel dengan FFmpeg.
func escapeConcatPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, "'", `'\''`)
	return p
}
