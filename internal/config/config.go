package config

import "os"

const Version = "0.1.0"

type Config struct {
	Host string
	Port string
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

	return Config{
		Host: host,
		Port: port,
	}
}

func (c Config) Addr() string {
	return c.Host + ":" + c.Port
}
