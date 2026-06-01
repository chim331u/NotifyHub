# User Manual: NotifyHub API Integration

Benvenuto nel Manuale Utente di **NotifyHub**, il gateway di messaggistica unificato ad alte prestazioni. Questa guida illustra come integrare ed utilizzare le API di notifica nei tuoi servizi.

---

## 1. Sicurezza & Autenticazione

Tutte le richieste HTTP inviate a NotifyHub devono essere autenticate inserendo la chiave segreta nell'header personalizzato **`X-API-Key`**:

```http
X-API-Key: la_tua_api_key_segreta
```

Le richieste sprovviste di tale header o con chiavi non autorizzate riceveranno una risposta `401 Unauthorized`.

---

## 2. Endpoint Unificato: `/api/notify`

*   **Metodo**: `POST`
*   **Path**: `/api/notify`
*   **Content-Type**: `application/json`

### Struttura Comune del Payload JSON:
```json
{
  "channel": "telegram | discord | email",
  "message": "Il testo del messaggio da inviare",
  "recipient": "Opzionale: chat_id, webhook_url o indirizzo email specifico",
  "subject": "Richiesto solo per il canale email: Oggetto della mail"
}
```

> [!NOTE]
> Se il campo `recipient` viene omesso, il servizio utilizzerà i canali e i destinatari di default definiti nel file di configurazione dei segreti sul server (`notifyhub-secrets.json`).

---

## 3. Canali di Notifica Specifici

### 3.1 Telegram
*   **Channel**: `"telegram"`
*   **Formato supportato**: **HTML** (standard Telegram). Puoi utilizzare tag come `<b>`, `<i>`, `<code>`, `<a href="...">` ecc.
*   **Esempio di Payload**:
    ```json
    {
      "channel": "telegram",
      "message": "🚨 <b>Allerta Critica</b>: Rilevato picco di carico sulla CPU!\n<i>Dettaglio:</i> <code>98% RAM saturata</code>\n<a href='http://nas.local/dashboard'>Apri Dashboard</a>",
      "recipient": "123456789"
    }
    ```

### 3.2 Discord
*   **Channel**: `"discord"`
*   **Formato supportato**: **Markdown** standard di Discord.
*   **Esempio di Payload**:
    ```json
    {
      "channel": "discord",
      "message": "### 🔔 Aggiornamento Backup\nIl backup di **HouseLedger** è stato completato con successo. \n- *Durata*: 2m 43s\n- *Dimensione*: `1.4 GB`",
      "recipient": "https://discord.com/api/webhooks/123456/abcdef"
    }
    ```

### 3.3 Email (SMTP)
*   **Channel**: `"email"`
*   **Subject**: Richiesto per l'invio e-mail.
*   **Formato supportato**: Testo piatto o HTML (in base a quanto configurato nel server).
*   **Esempio di Payload**:
    ```json
    {
      "channel": "email",
      "subject": "[NotifyHub] Report Mensile MediaButler",
      "message": "Gentile utente, ecco il report mensile dell'utilizzo di MediaButler...\n...",
      "recipient": "destinatario@esempio.com"
    }
    ```

---

## 4. Codici di Risposta HTTP

NotifyHub risponde con i seguenti codici di stato standardizzati:

| Codice HTTP | Descrizione | Causa |
| :--- | :--- | :--- |
| **`200 OK`** | Successo | Il messaggio è stato preso in carico ed inserito nella coda di invio. |
| **`400 Bad Request`** | Payload Errato | Il corpo JSON è malformato o mancano campi obbligatori (es. `channel`, `message`). |
| **`401 Unauthorized`** | Non Autenticato | Header `X-API-Key` mancante o non corrispondente alle chiavi configurate. |
| **`429 Too Many Requests`** | Limite Raggiunto | Il client sta inviando troppi messaggi ed ha superato la soglia di sicurezza temporanea. |
| **`500 Internal Error`** | Errore Interno | Un problema interno al server Go (es. coda piena o problemi di allocazione). |
