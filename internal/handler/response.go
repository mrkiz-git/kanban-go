package handler

import (
	"encoding/json"
	"net/http"
)

type APIErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details,omitempty"`
	} `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	writeJSON(w, status, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details any) {
	body := APIErrorBody{}
	body.Error.Code = code
	body.Error.Message = message
	if details != nil {
		body.Error.Details = details
	}
	writeJSON(w, status, body)
}

func WriteUnauthorized(w http.ResponseWriter) {
	writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
}

func WriteForbidden(w http.ResponseWriter) {
	writeAPIError(w, http.StatusForbidden, "forbidden", "Forbidden", nil)
}

func WriteAPIError(w http.ResponseWriter, status int, code, message string, details any) {
	writeAPIError(w, status, code, message, details)
}
