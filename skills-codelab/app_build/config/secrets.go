package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// TelegramSecrets contiene le credenziali isolate per Telegram.
type TelegramSecrets struct {
	BotToken      string `json:"bot_token"`
	DefaultChatID string `json:"default_chat_id"`
}

// DiscordSecrets contiene le credenziali isolate per Discord.
type DiscordSecrets struct {
	DefaultWebhookURL string `json:"default_webhook_url"`
}

// SMTPSecrets contiene le credenziali isolate per l'invio e-mail.
type SMTPSecrets struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Sender   string `json:"sender"`
}

// Secrets definisce il contenitore immutabile per tutti i segreti di NotifyHub.
type Secrets struct {
	APIKeys  []string        `json:"api_keys"`
	Telegram TelegramSecrets `json:"telegram"`
	Discord  DiscordSecrets  `json:"discord"`
	SMTP     SMTPSecrets     `json:"smtp"`
}

// LoadSecrets carica i segreti dal percorso del file specificato.
func LoadSecrets(path string) (Secrets, error) {
	var s Secrets

	file, err := os.Open(path)
	if err != nil {
		return s, fmt.Errorf("impossibile aprire il file dei segreti: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&s); err != nil {
		return s, fmt.Errorf("errore nel parsing del file JSON dei segreti: %w", err)
	}

	return s, nil
}

// IsAuthorized verifica se l'API Key fornita è presente nell'elenco delle chiavi autorizzate.
func (s Secrets) IsAuthorized(apiKey string) bool {
	if apiKey == "" {
		return false
	}
	for _, key := range s.APIKeys {
		if key == apiKey {
			return true
		}
	}
	return false
}
