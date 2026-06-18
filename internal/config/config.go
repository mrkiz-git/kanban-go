package config

import (
	"os"

	"github.com/mrkiz-git/kanba-go/internal/logging"
)

const Version = "0.1.0"

type Config struct {
	Host      string
	Port      string
	StaticDir string
	LogLevel  logging.Level
	LogFile   string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "web/out"
	}

	return Config{
		Host:      host,
		Port:      port,
		StaticDir: staticDir,
		LogLevel:  logging.ParseLevel(os.Getenv("LOG_LEVEL")),
		LogFile:   os.Getenv("LOG_FILE"),
	}
}

func (c Config) Addr() string {
	return c.Host + ":" + c.Port
}
