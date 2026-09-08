package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"go-streamer/internal/config"
	"go-streamer/internal/domain"
	"go-streamer/internal/repository/sqlite"
)

func setupTestServer(t *testing.T) (*Server, *sqlite.DB) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_api.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}

	cfg := &config.Config{
		Port:     8080,
		DataDir:  tempDir,
		VideoDir: filepath.Join(tempDir, "videos"),
		CacheDir: filepath.Join(tempDir, "cache"),
		Debug:    true,
	}

	platform := &domain.PlatformInfo{
		OS:          "linux",
		Arch:        "amd64",
		NumCPU:      4,
		Version:     "v1.0.0",
		FFmpegPath:  "",
		FFprobePath: "",
	}

	server := NewServer(cfg, db, platform)
	return server, db
}

func TestAPIRoutes(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	defer server.Close()

	router := server.Router()

	// 1. GET /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET / returned %d", w.Code)
	}

	// 2. GET /api/v1/health
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/health returned %d", w.Code)
	}

	// 3. GET /api/v1/system/info
	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/system/info returned %d", w.Code)
	}

	// 4. GET /api/v1/system/metrics
	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/metrics", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/system/metrics returned %d", w.Code)
	}

	// 5. POST /api/v1/system/killswitch
	req = httptest.NewRequest(http.MethodPost, "/api/v1/system/killswitch", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST /api/v1/system/killswitch returned %d", w.Code)
	}

	// 6. GET /api/v1/slots
	req = httptest.NewRequest(http.MethodGet, "/api/v1/slots", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/slots returned %d", w.Code)
	}
	var slots []*domain.StreamSlot
	if err := json.NewDecoder(w.Body).Decode(&slots); err != nil || len(slots) < 2 {
		t.Errorf("expected at least 2 default slots, got %d", len(slots))
	}

	// 7. GET /api/v1/slots/1
	req = httptest.NewRequest(http.MethodGet, "/api/v1/slots/1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/slots/1 returned %d", w.Code)
	}

	// 8. PUT /api/v1/slots/1
	updateSlotPayload := `{"name": "Slot 1 Updated", "target_platform": "youtube", "rtmp_url": "rtmp://localhost/live", "mode": "copy"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/slots/1", strings.NewReader(updateSlotPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /api/v1/slots/1 returned %d", w.Code)
	}
	var updatedSlot domain.StreamSlot
	_ = json.NewDecoder(w.Body).Decode(&updatedSlot)
	if updatedSlot.Name != "Slot 1 Updated" {
		t.Errorf("expected updated slot name 'Slot 1 Updated', got '%s'", updatedSlot.Name)
	}
	if !updatedSlot.AutoRestart {
		t.Errorf("expected partial update to preserve AutoRestart=true")
	}

	// 9. POST /api/v1/videos/upload (Multipart upload)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, err := mw.CreateFormFile("video", "sample_test.mp4")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("dummy video binary stream"))
	_ = mw.WriteField("user_id", "usr_admin_default")
	_ = mw.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/v1/videos/upload", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("POST /api/v1/videos/upload returned %d: %s", w.Code, w.Body.String())
	}
	var uploadedVideo domain.Video
	_ = json.NewDecoder(w.Body).Decode(&uploadedVideo)

	// 10. GET /api/v1/videos
	req = httptest.NewRequest(http.MethodGet, "/api/v1/videos", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/videos returned %d", w.Code)
	}

	// 11. GET /api/v1/videos/storage
	req = httptest.NewRequest(http.MethodGet, "/api/v1/videos/storage", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/videos/storage returned %d", w.Code)
	}

	// 12. PUT /api/v1/videos/{id}/rename (testing "filename" fallback and domain.Video return)
	if uploadedVideo.ID != "" {
		renamePayload := `{"filename": "renamed_video.mp4"}`
		req = httptest.NewRequest(http.MethodPut, "/api/v1/videos/"+uploadedVideo.ID+"/rename", strings.NewReader(renamePayload))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("PUT /api/v1/videos/rename returned %d", w.Code)
		}
		var renamedVid domain.Video
		if err := json.NewDecoder(w.Body).Decode(&renamedVid); err != nil {
			t.Errorf("failed to decode renamed video response: %v", err)
		} else if renamedVid.OriginalName != "renamed_video.mp4" {
			t.Errorf("expected renamed video OriginalName 'renamed_video.mp4', got '%s'", renamedVid.OriginalName)
		}
	}

	// 13. POST /api/v1/schedules
	schedPayload := `{"title": "Prime Time Stream", "slot_id": 1, "cron_expr": "0 12 * * *", "duration_minutes": 30, "overlap_guard_policy": "yield_priority", "is_enabled": true}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(schedPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("POST /api/v1/schedules returned %d: %s", w.Code, w.Body.String())
	}
	var createdSched domain.Schedule
	_ = json.NewDecoder(w.Body).Decode(&createdSched)
	if createdSched.Title != "Prime Time Stream" {
		t.Errorf("expected schedule title 'Prime Time Stream', got '%s'", createdSched.Title)
	}

	// 14. GET /api/v1/schedules
	req = httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/schedules returned %d", w.Code)
	}

	// 15. GET /api/v1/schedules/overlap-status
	req = httptest.NewRequest(http.MethodGet, "/api/v1/schedules/overlap-status", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/schedules/overlap-status returned %d", w.Code)
	}

	// 16. DELETE /api/v1/schedules/{id}
	if createdSched.ID != "" {
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/"+createdSched.ID, nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("DELETE /api/v1/schedules returned %d", w.Code)
		}
	}

	// 17. GET /api/v1/tunnel/status
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tunnel/status", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/tunnel/status returned %d", w.Code)
	}

	// 18. GET /api/v1/alerts/settings
	req = httptest.NewRequest(http.MethodGet, "/api/v1/alerts/settings", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/alerts/settings returned %d", w.Code)
	}

	// 19. PUT /api/v1/alerts/settings
	alertUpdatePayload := `{"telegram_enabled": true, "telegram_bot_token": "my_secret_token", "discord_enabled": true, "discord_webhook_url": "https://discord.com/api/webhooks/my_secret", "trigger_on_crash": true, "trigger_on_thermal": true, "thermal_threshold_c": 50.0}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/alerts/settings", strings.NewReader(alertUpdatePayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /api/v1/alerts/settings returned %d", w.Code)
	}
	var alertResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&alertResp); err != nil {
		t.Errorf("failed to decode alert response: %v", err)
	} else {
		if alertResp["has_telegram_token"] != true || alertResp["telegram_bot_token"] != "********" {
			t.Errorf("expected masked telegram token, got %v", alertResp["telegram_bot_token"])
		}
		if alertResp["has_discord_webhook"] != true || alertResp["discord_webhook_url"] != "********" {
			t.Errorf("expected masked discord webhook, got %v", alertResp["discord_webhook_url"])
		}
	}

	// 19b. PUT /api/v1/alerts/settings (test clearing tokens with "")
	clearAlertPayload := `{"telegram_enabled": false, "telegram_bot_token": "", "discord_enabled": false, "discord_webhook_url": ""}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/alerts/settings", strings.NewReader(clearAlertPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /api/v1/alerts/settings (clearing) returned %d", w.Code)
	}
	var clearAlertResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&clearAlertResp); err != nil {
		t.Errorf("failed to decode alert response: %v", err)
	} else {
		if clearAlertResp["has_telegram_token"] != false || clearAlertResp["telegram_bot_token"] != nil {
			t.Errorf("expected empty telegram token after clear, got %v", clearAlertResp["telegram_bot_token"])
		}
		if clearAlertResp["has_discord_webhook"] != false || clearAlertResp["discord_webhook_url"] != nil {
			t.Errorf("expected empty discord webhook after clear, got %v", clearAlertResp["discord_webhook_url"])
		}
	}

	// 20. GET /api/v1/codec/jobs
	req = httptest.NewRequest(http.MethodGet, "/api/v1/codec/jobs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/codec/jobs returned %d", w.Code)
	}

	// 21. DELETE /api/v1/videos/{id}
	if uploadedVideo.ID != "" {
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/videos/"+uploadedVideo.ID, nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("DELETE /api/v1/videos returned %d", w.Code)
		}
	}

	// 22. POST /api/v1/videos/upload-chunk (Chunk 0 dan Chunk 1)
	uploadID := "test-chunk-upload-123"

	// Chunk 0
	var chunk0Buf bytes.Buffer
	mw0 := multipart.NewWriter(&chunk0Buf)
	part0, _ := mw0.CreateFormFile("chunk", "blob")
	_, _ = part0.Write([]byte("CHUNK_PART_0_DATA_"))
	_ = mw0.WriteField("upload_id", uploadID)
	_ = mw0.WriteField("chunk_index", "0")
	_ = mw0.WriteField("total_chunks", "2")
	_ = mw0.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/v1/videos/upload-chunk", &chunk0Buf)
	req.Header.Set("Content-Type", mw0.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST /api/v1/videos/upload-chunk (0) returned %d: %s", w.Code, w.Body.String())
	}

	// Chunk 1
	var chunk1Buf bytes.Buffer
	mw1 := multipart.NewWriter(&chunk1Buf)
	part1, _ := mw1.CreateFormFile("chunk", "blob")
	_, _ = part1.Write([]byte("CHUNK_PART_1_DATA_END"))
	_ = mw1.WriteField("upload_id", uploadID)
	_ = mw1.WriteField("chunk_index", "1")
	_ = mw1.WriteField("total_chunks", "2")
	_ = mw1.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/v1/videos/upload-chunk", &chunk1Buf)
	req.Header.Set("Content-Type", mw1.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST /api/v1/videos/upload-chunk (1) returned %d: %s", w.Code, w.Body.String())
	}

	// 23. POST /api/v1/videos/upload-complete
	completePayload := `{"upload_id": "test-chunk-upload-123", "filename": "assembled_video.mp4", "user_id": "usr_admin_default"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/videos/upload-complete", strings.NewReader(completePayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("POST /api/v1/videos/upload-complete returned %d: %s", w.Code, w.Body.String())
	}
	var assembled domain.Video
	_ = json.NewDecoder(w.Body).Decode(&assembled)
	if assembled.OriginalName != "assembled_video.mp4" {
		t.Errorf("expected assembled original_name 'assembled_video.mp4', got %s", assembled.OriginalName)
	}

	// Cleanup assembled video
	if assembled.ID != "" {
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/videos/"+assembled.ID, nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("DELETE assembled video returned %d", w.Code)
		}
	}

	// 24. Test WebSocket Endpoint di /api/v1/ws
	httpTestServer := httptest.NewServer(router)
	defer httpTestServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpTestServer.URL, "http") + "/api/v1/ws"
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial /api/v1/ws: %v", err)
	}
	defer wsConn.Close()
}
