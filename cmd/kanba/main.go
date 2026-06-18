package main

import (
	"log"

	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.New(cfg)

	log.Printf("kanba %s listening on %s", config.Version, cfg.Addr())
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
