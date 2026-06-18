package handler

import (
	"encoding/json"
	"net/http"

	"github.com/mrkiz-git/kanba-go/internal/config"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "ok",
		Version: config.Version,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
