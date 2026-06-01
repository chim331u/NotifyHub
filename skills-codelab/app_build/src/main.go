package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"notifyhub/config"
)

// ErrPermanent definisce un errore che non deve essere ritentato in quanto definitivo.
var ErrPermanent = errors.New("errore permanente")

// Notification rappresenta la struttura del messaggio in transito nella coda.
type Notification struct {
	Channel   string `json:"channel"`
	Message   string `json:"message"`
	Recipient string `json:"recipient"`
	Subject   string `json:"subject"`
}

// Global state/orchestrators (isolati e ben definiti)
type App struct {
	cfg     config.Config
	secrets config.Secrets
	queue   chan Notification
}

func main() {
	log.Println("[NotifyHub] Avvio del servizio in corso...")

	// 1. Carica Configurazione e Segreti
	cfg := config.LoadConfig()
	secrets, err := config.LoadSecrets(cfg.SecretPath)
	if err != nil {
		log.Printf("[WARNING] Impossibile caricare i segreti reali da %s: %v. Verrà usato un set di fallback fittizio per scopi di test.\n", cfg.SecretPath, err)
		secrets = getMockSecretsFallback()
	}

	// 2. Inizializza l'applicazione con una coda di dimensione 1000 per picchi di carico
	app := &App{
		cfg:     cfg,
		secrets: secrets,
		queue:   make(chan Notification, 1000),
	}

	// 3. Avvia i Worker di notifica in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const numWorkers = 3
	for i := 1; i <= numWorkers; i++ {
		go app.worker(ctx, i)
	}
	log.Printf("[NotifyHub] Avviati %d worker asincroni per la gestione delle notifiche.\n", numWorkers)

	// 4. Configura il Server HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/api/notify", app.handleNotify)
	mux.HandleFunc("/api/health", app.handleHealth)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful Shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("[NotifyHub] Ricevuto segnale di interruzione. Spegnimento in corso...")
		cancel() // Ferma i worker in background

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[ERROR] Errore durante lo spegnimento del server HTTP: %v\n", err)
		}
	}()

	log.Printf("[NotifyHub] Server in ascolto sulla porta %s in modalità '%s'\n", cfg.Port, cfg.Env)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("[FATAL] Errore all'avvio del server HTTP: %v\n", err)
	}

	log.Println("[NotifyHub] Servizio arrestato correttamente.")
}

// handleNotify gestisce le richieste in ingresso POST /api/notify.
func (app *App) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Metodo non consentito"}`, http.StatusMethodNotAllowed)
		return
	}

	// 1. Validazione API Key
	apiKey := r.Header.Get("X-API-Key")
	if !app.secrets.IsAuthorized(apiKey) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "API Key non valida o mancante"}`))
		return
	}

	// 2. Parsing del JSON Payload
	var note Notification
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&note); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON malformato"}`))
		return
	}

	// 3. Validazione Campi Richiesti
	note.Channel = strings.ToLower(strings.TrimSpace(note.Channel))
	if note.Channel == "" || strings.TrimSpace(note.Message) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Campi obbligatori mancanti: 'channel' e 'message' sono richiesti"}`))
		return
	}

	// Verifica supporto canale
	if note.Channel != "telegram" && note.Channel != "discord" && note.Channel != "email" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Canale non supportato. Scegli tra 'telegram', 'discord' o 'email'"}`))
		return
	}

	// 4. Accettazione e accodamento
	select {
	case app.queue <- note:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "queued", "message": "Messaggio accodato correttamente per l'invio asincrono"}`))
	default:
		// Se la coda è piena, risponde con 500
		log.Println("[ERROR] Coda saturata! Impossibile accodare nuovi messaggi.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Coda messaggi satura, riprova più tardi"}`))
	}
}

// handleHealth risponde 200 OK per monitoraggi e controlli di integrità.
func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "service": "NotifyHub"}`))
}

// worker consuma le notifiche dalla coda asincrona.
func (app *App) worker(ctx context.Context, id int) {
	log.Printf("[Worker #%d] Avviato con successo.\n", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Worker #%d] Arresto in corso...\n", id)
			return
		case note := <-app.queue:
			log.Printf("[Worker #%d] Inizio elaborazione messaggio per canale: %s\n", id, note.Channel)
			app.processNotificationWithRetry(note)
		}
	}
}

// processNotificationWithRetry gestisce l'invio effettivo con exponential backoff retry.
func (app *App) processNotificationWithRetry(note Notification) {
	const maxRetries = 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := app.sendNotification(note)
		if err == nil {
			log.Printf("[Success] Messaggio inviato correttamente sul canale '%s'\n", note.Channel)
			return
		}

		// Se l'errore è contrassegnato come permanente (4xx), interrompe immediatamente i tentativi
		if errors.Is(err, ErrPermanent) {
			log.Printf("[FATAL ERR] Errore permanente sul canale '%s': %v. Interruzione immediata dei tentativi.\n", note.Channel, err)
			return
		}

		log.Printf("[Retry #%d] Errore durante l'invio sul canale '%s': %v. Nuovo tentativo tra %v...\n", attempt, note.Channel, err, backoff)
		
		// Attesa con possibilità di interruzione
		time.Sleep(backoff)
		backoff *= 2 // Raddoppia il tempo di attesa ad ogni tentativo fallito
	}

	log.Printf("[FATAL ERR] Impossibile inviare la notifica dopo %d tentativi su canale '%s'\n", maxRetries, note.Channel)
}

// sendNotification smista la notifica verso i driver reali o mock in base alle credenziali configurate.
func (app *App) sendNotification(note Notification) error {
	switch note.Channel {
	case "telegram":
		token := app.secrets.Telegram.BotToken
		chatID := note.Recipient
		if chatID == "" {
			chatID = app.secrets.Telegram.DefaultChatID
		}

		// Se il token è fittizio, emula l'invio
		if token == "" || strings.HasPrefix(token, "MOCK_") {
			log.Printf("[MOCK TELEGRAM] Invio simulato a ChatID '%s': %s\n", chatID, note.Message)
			return nil
		}

		return app.sendRealTelegram(token, chatID, note.Message)

	case "discord":
		webhookURL := note.Recipient
		if webhookURL == "" {
			webhookURL = app.secrets.Discord.DefaultWebhookURL
		}

		// Se l'URL è fittizio, emula l'invio
		if webhookURL == "" || strings.Contains(webhookURL, "mock_") || strings.Contains(webhookURL, "localhost") {
			log.Printf("[MOCK DISCORD] Invio simulato a Webhook: %s\n", note.Message)
			return nil
		}

		return app.sendRealDiscord(webhookURL, note.Message)

	case "email":
		smtpSecrets := app.secrets.SMTP
		recipient := note.Recipient
		if recipient == "" {
			return errors.New("destinatario e-mail mancante")
		}

		// Se l'host SMTP è fittizio, emula l'invio
		if smtpSecrets.Host == "localhost" || smtpSecrets.Host == "" || strings.Contains(smtpSecrets.Host, "mock") {
			log.Printf("[MOCK EMAIL] Invio simulato a '%s' (Oggetto: '%s'): %s\n", recipient, note.Subject, note.Message)
			return nil
		}

		return app.sendRealEmail(smtpSecrets, recipient, note.Subject, note.Message)
	}

	return fmt.Errorf("canale non supportato: %s", note.Channel)
}

// sendRealTelegram implementa l'integrazione reale con Telegram Bot API.
func (app *App) sendRealTelegram(token, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	
	formData := url.Values{}
	formData.Set("chat_id", chatID)
	formData.Set("text", message)
	formData.Set("parse_mode", "HTML")

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		return fmt.Errorf("chiamata HTTP a Telegram fallita: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		// Rate limiting di Telegram. Se possibile legge il tempo di attesa consigliato
		retryAfterStr := resp.Header.Get("Retry-After")
		if retryAfterSec, err := strconv.Atoi(retryAfterStr); err == nil {
			log.Printf("[Telegram Rate Limit] Attesa obbligatoria di %d secondi richiesta da Telegram\n", retryAfterSec)
			time.Sleep(time.Duration(retryAfterSec) * time.Second)
		}
		return errors.New("rate limit di Telegram superato")
	}

	// Gli errori 4xx (client error) esclusi i rate limit 429 sono permanenti e non vanno ritentati
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return fmt.Errorf("%w: errore API Telegram (HTTP %d)", ErrPermanent, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("errore API Telegram: codice di stato HTTP %d", resp.StatusCode)
	}

	return nil
}

// sendRealDiscord implementa l'integrazione reale con Discord Webhook.
func (app *App) sendRealDiscord(webhookURL, message string) error {
	payload := map[string]string{
		"content": message,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("impossibile serializzare il payload Discord: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("chiamata HTTP a Discord fallita: %w", err)
	}
	defer resp.Body.Close()

	// Gli errori 4xx (client error) esclusi i rate limit 429 sono permanenti e non vanno ritentati
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
		return fmt.Errorf("%w: errore API Discord (HTTP %d)", ErrPermanent, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("errore Webhook Discord: codice di stato HTTP %d", resp.StatusCode)
	}

	return nil
}

// sendRealEmail implementa l'integrazione reale con il server SMTP.
func (app *App) sendRealEmail(s config.SMTPSecrets, recipient, subject, message string) error {
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	
	msg := []byte("To: " + recipient + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		message + "\r\n")

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	err := smtp.SendMail(addr, auth, s.Sender, []string{recipient}, msg)
	if err != nil {
		return fmt.Errorf("invio email SMTP fallito: %w", err)
	}

	return nil
}

// getMockSecretsFallback genera un set di segreti di fallback fittizi se il file reale manca.
func getMockSecretsFallback() config.Secrets {
	return config.Secrets{
		APIKeys: []string{"chiave_test_12345"},
		Telegram: config.TelegramSecrets{
			BotToken:      "MOCK_TELEGRAM_BOT_TOKEN",
			DefaultChatID: "MOCK_CHAT_ID",
		},
		Discord: config.DiscordSecrets{
			DefaultWebhookURL: "http://localhost/mock_webhook",
		},
		SMTP: config.SMTPSecrets{
			Host:     "localhost",
			Port:     1025,
			Username: "mock_user",
			Password: "mock_password",
			Sender:   "notifyhub@localhost",
		},
	}
}
