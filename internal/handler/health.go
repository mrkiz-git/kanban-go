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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Version: config.Version,
	})
}
