package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"notifyhub/config"
)

func TestHandleHealth(t *testing.T) {
	app := &App{}
	req, err := http.NewRequest("GET", "/api/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.handleHealth)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Stato restituito errato: atteso %v, ottenuto %v", http.StatusOK, status)
	}

	expected := `{"status": "healthy", "service": "NotifyHub"}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("Corpo della risposta errato: atteso %v, ottenuto %v", expected, rr.Body.String())
	}
}

func TestHandleNotifyUnauthorized(t *testing.T) {
	app := &App{
		secrets: config.Secrets{
			APIKeys: []string{"chiave_valida"},
		},
	}

	req, err := http.NewRequest("POST", "/api/notify", nil)
	if err != nil {
		t.Fatal(err)
	}
	// API Key mancante o non corretta
	req.Header.Set("X-API-Key", "chiave_errata")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.handleNotify)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Atteso StatusUnauthorized (401), ottenuto %v", status)
	}
}

func TestHandleNotifySuccessAndQueue(t *testing.T) {
	app := &App{
		secrets: config.Secrets{
			APIKeys: []string{"chiave_valida"},
		},
		queue: make(chan Notification, 10),
	}

	payload := Notification{
		Channel:   "telegram",
		Message:   "Test message",
		Recipient: "12345",
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "/api/notify", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-API-Key", "chiave_valida")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.handleNotify)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Atteso StatusOK (200), ottenuto %v", status)
	}

	// Verifica che l'elemento sia stato inserito nella coda
	if len(app.queue) != 1 {
		t.Errorf("La coda dovrebbe contenere esattamente 1 elemento, ne contiene %d", len(app.queue))
	}

	queuedNote := <-app.queue
	if queuedNote.Channel != "telegram" || queuedNote.Message != "Test message" {
		t.Errorf("Notifica accodata corrotta o errata: %v", queuedNote)
	}
}
