package config

import (
	"os"
)

// Config definisce i parametri globali dell'applicazione, immutabili all'avvio.
type Config struct {
	Env        string
	Port       string
	SecretPath string
}

// LoadConfig carica la configurazione dalle variabili d'ambiente con fallback sicuri.
func LoadConfig() Config {
	env := os.Getenv("NOTIFYHUB_ENV")
	if env == "" {
		env = "development"
	}

	port := os.Getenv("NOTIFYHUB_PORT")
	if port == "" {
		port = "30180"
	}

	secretPath := os.Getenv("NOTIFYHUB_SECRET_PATH")
	if secretPath == "" {
		secretPath = "data/notifyhub-secrets.json"
	}

	return Config{
		Env:        env,
		Port:       port,
		SecretPath: secretPath,
	}
}
