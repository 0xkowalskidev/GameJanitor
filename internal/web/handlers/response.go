package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

type envelope struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

func respondOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(envelope{Status: "ok", Data: data})
}

func respondCreated(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(envelope{Status: "ok", Data: data})
}

func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(envelope{Status: "error", Error: message})
}

// serviceErrorStatus maps service-layer error messages to HTTP status codes.
func serviceErrorStatus(err error) int {
	msg := err.Error()
	if strings.Contains(msg, "not found") {
		return http.StatusNotFound
	}
	if strings.Contains(msg, "must be stopped") || strings.Contains(msg, "still reference") {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
