package config

import (
	"os"

	"github.com/mrkiz-git/kanba-go/internal/logging"
)

const Version = "0.1.0"

const defaultJWTSecret = "dev-only-jwt-secret-change-in-production"
const defaultAdminPassword = "changeme"

type Config struct {
	Host         string
	Port         string
	StaticDir    string
	LogLevel     logging.Level
	LogFile      string
	DatabasePath string
	JWTSecret    string
	AdminEmail   string
	AdminPassword string
	AdminName    string
	SecureCookie bool
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

	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "data/kanba.db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = defaultJWTSecret
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@kanba.local"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "changeme"
	}

	adminName := os.Getenv("ADMIN_NAME")
	if adminName == "" {
		adminName = "System Admin"
	}

	insecureCookie := os.Getenv("KANBA_INSECURE_COOKIE") == "1" || os.Getenv("KANBA_INSECURE_COOKIE") == "true"
	secureCookie := !insecureCookie

	return Config{
		Host:          host,
		Port:          port,
		StaticDir:     staticDir,
		LogLevel:      logging.ParseLevel(os.Getenv("LOG_LEVEL")),
		LogFile:       os.Getenv("LOG_FILE"),
		DatabasePath:  databasePath,
		JWTSecret:     jwtSecret,
		AdminEmail:    adminEmail,
		AdminPassword: adminPassword,
		AdminName:     adminName,
		SecureCookie:  secureCookie,
	}
}

func (c Config) Addr() string {
	return c.Host + ":" + c.Port
}

func DefaultJWTSecret() string {
	return defaultJWTSecret
}

func DefaultAdminPassword() string {
	return defaultAdminPassword
}
