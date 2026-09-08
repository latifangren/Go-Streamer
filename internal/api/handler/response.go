package handler

import (
	"encoding/json"
	"net/http"
)

// writeJSON menyajikan respon JSON dengan kode status HTTP tertentu.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError menyajikan respon JSON format standar error.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
