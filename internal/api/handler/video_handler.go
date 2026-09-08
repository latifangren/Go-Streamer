package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/domain"
	"go-streamer/internal/service"
)

// VideoHandler menangani endpoint media management file video dan chunked upload.
type VideoHandler struct {
	videoService *service.VideoService
	wsHub        *ws.Hub
	cacheDir     string
}

// NewVideoHandler membuat instans baru VideoHandler.
func NewVideoHandler(videoService *service.VideoService, wsHub *ws.Hub, cacheDir ...string) *VideoHandler {
	cd := "data/cache"
	if len(cacheDir) > 0 && cacheDir[0] != "" {
		cd = cacheDir[0]
	}
	return &VideoHandler{
		videoService: videoService,
		wsHub:        wsHub,
		cacheDir:     cd,
	}
}

// ListVideos menangani GET /api/v1/videos (opsional query param ?user_id=...).
func (h *VideoHandler) ListVideos(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	videos, err := h.videoService.ListVideos(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list videos: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, videos)
}

// UploadVideo menangani POST /api/v1/videos/upload (single multipart form upload).
func (h *VideoHandler) UploadVideo(w http.ResponseWriter, r *http.Request) {
	// Batasi ukuran memory buffer multipart hingga 32 MB
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		writeError(w, http.StatusBadRequest, "form field 'video' is required: "+err.Error())
		return
	}
	defer file.Close()

	userID := strings.TrimSpace(r.FormValue("user_id"))
	if userID == "" {
		userID = "usr_admin_default"
	}

	video, err := h.videoService.SaveUploadedVideo(r.Context(), userID, header.Filename, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save uploaded video: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("video_uploaded", video)
	}

	writeJSON(w, http.StatusCreated, video)
}

// UploadChunk menangani POST /api/v1/videos/upload-chunk untuk resumable upload bertahap.
func (h *VideoHandler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	uploadID := strings.TrimSpace(r.FormValue("upload_id"))
	if uploadID == "" {
		uploadID = strings.TrimSpace(r.URL.Query().Get("upload_id"))
	}
	uploadID = sanitizeUploadID(uploadID)
	if uploadID == "" {
		writeError(w, http.StatusBadRequest, "upload_id is required")
		return
	}

	chunkIndexStr := strings.TrimSpace(r.FormValue("chunk_index"))
	if chunkIndexStr == "" {
		chunkIndexStr = strings.TrimSpace(r.URL.Query().Get("chunk_index"))
	}
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		writeError(w, http.StatusBadRequest, "valid chunk_index (>= 0) is required")
		return
	}

	totalChunksStr := strings.TrimSpace(r.FormValue("total_chunks"))
	if totalChunksStr == "" {
		totalChunksStr = strings.TrimSpace(r.URL.Query().Get("total_chunks"))
	}
	totalChunks, _ := strconv.Atoi(totalChunksStr)

	// Ambil file chunk dari form
	file, _, err := r.FormFile("chunk")
	if err != nil {
		file, _, err = r.FormFile("file")
		if err != nil {
			file, _, err = r.FormFile("video")
			if err != nil {
				writeError(w, http.StatusBadRequest, "chunk file is required in field 'chunk' or 'file'")
				return
			}
		}
	}
	defer file.Close()

	uploadDir := filepath.Join(h.cacheDir, "uploads", uploadID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create upload directory: "+err.Error())
		return
	}

	chunkPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", chunkIndex))
	outFile, err := os.Create(chunkPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create chunk file: "+err.Error())
		return
	}

	written, err := io.Copy(outFile, file)
	_ = outFile.Close()
	if err != nil {
		_ = os.Remove(chunkPath)
		writeError(w, http.StatusInternalServerError, "failed to write chunk data: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "chunk uploaded successfully",
		"upload_id":    uploadID,
		"chunk_index":  chunkIndex,
		"total_chunks": totalChunks,
		"bytes_saved":  written,
	})
}

// UploadComplete menangani POST /api/v1/videos/upload-complete untuk menggabungkan seluruh chunk.
func (h *VideoHandler) UploadComplete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UploadID    string `json:"upload_id"`
		Filename    string `json:"filename"`
		UserID      string `json:"user_id"`
		TotalChunks int    `json:"total_chunks"`
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.UploadID = r.FormValue("upload_id")
		req.Filename = r.FormValue("filename")
		req.UserID = r.FormValue("user_id")
		req.TotalChunks, _ = strconv.Atoi(r.FormValue("total_chunks"))
	}

	if req.UploadID == "" {
		req.UploadID = r.URL.Query().Get("upload_id")
	}
	if req.Filename == "" {
		req.Filename = r.URL.Query().Get("filename")
	}

	uploadID := sanitizeUploadID(strings.TrimSpace(req.UploadID))
	filename := strings.TrimSpace(req.Filename)
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = "usr_admin_default"
	}

	if uploadID == "" {
		writeError(w, http.StatusBadRequest, "upload_id is required")
		return
	}
	if filename == "" {
		writeError(w, http.StatusBadRequest, "filename is required")
		return
	}

	uploadDir := filepath.Join(h.cacheDir, "uploads", uploadID)
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "upload session not found")
		return
	}

	totalChunks := req.TotalChunks
	if totalChunks <= 0 {
		files, err := os.ReadDir(uploadDir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read upload directory: "+err.Error())
			return
		}
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "chunk_") {
				totalChunks++
			}
		}
	}

	if totalChunks == 0 {
		writeError(w, http.StatusBadRequest, "no chunks found for upload_id")
		return
	}

	// Gabungkan seluruh chunk secara berurutan ke merged.tmp
	mergedPath := filepath.Join(uploadDir, "merged.tmp")
	mergedFile, err := os.Create(mergedPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create merged file: "+err.Error())
		return
	}

	for i := 0; i < totalChunks; i++ {
		cPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", i))
		cFile, err := os.Open(cPath)
		if err != nil {
			_ = mergedFile.Close()
			_ = os.Remove(mergedPath)
			writeError(w, http.StatusBadRequest, fmt.Sprintf("missing chunk %d: %v", i, err))
			return
		}
		_, err = io.Copy(mergedFile, cFile)
		_ = cFile.Close()
		if err != nil {
			_ = mergedFile.Close()
			_ = os.Remove(mergedPath)
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read chunk %d: %v", i, err))
			return
		}
	}

	// Rewind mergedFile ke posisi awal
	if _, err := mergedFile.Seek(0, 0); err != nil {
		_ = mergedFile.Close()
		_ = os.Remove(mergedPath)
		writeError(w, http.StatusInternalServerError, "failed to seek merged file: "+err.Error())
		return
	}

	video, err := h.videoService.SaveUploadedVideo(r.Context(), userID, filename, mergedFile)
	_ = mergedFile.Close()

	// Hapus folder upload temporary
	_ = os.RemoveAll(uploadDir)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process and save merged video: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("video_uploaded", video)
	}

	writeJSON(w, http.StatusCreated, video)
}

// DeleteVideo menangani DELETE /api/v1/videos/{id}.
func (h *VideoHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "video id is required")
		return
	}

	if err := h.videoService.DeleteVideo(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "video not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete video: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("video_deleted", map[string]string{"id": id})
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "video deleted successfully",
		"id":      id,
	})
}

// RenameVideo menangani PUT /api/v1/videos/{id}/rename.
func (h *VideoHandler) RenameVideo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "video id is required")
		return
	}

	var req struct {
		Name     string `json:"name"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body: "+err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Filename = strings.TrimSpace(req.Filename)
	if req.Name == "" && req.Filename != "" {
		req.Name = req.Filename
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "new name cannot be empty")
		return
	}

	if err := h.videoService.RenameVideo(r.Context(), id, req.Name); err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "video not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to rename video: "+err.Error())
		return
	}

	v, err := h.videoService.GetVideo(r.Context(), id)
	if err == nil {
		writeJSON(w, http.StatusOK, v)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "video renamed successfully",
	})
}

// GetStorageStats menangani GET /api/v1/videos/storage.
func (h *VideoHandler) GetStorageStats(w http.ResponseWriter, r *http.Request) {
	totalBytes, count, err := h.videoService.GetStorageStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get storage stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_bytes": totalBytes,
		"video_count": count,
	})
}

func sanitizeUploadID(id string) string {
	id = filepath.Base(id)
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
