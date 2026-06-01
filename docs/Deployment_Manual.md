# Deployment Manual: Running NotifyHub

Questa guida illustra la configurazione, la build, il rilascio e la transizione da credenziali fittizie (utilizzate per i primi test) alle chiavi di produzione reali per **NotifyHub**.

---

## 1. Configurazione dei Segreti (`notifyhub-secrets.json`)

NotifyHub gestisce l'isolamento dei segreti caricando le credenziali da un file JSON locale montato in sola lettura (`ro`) nel container. 

Il file deve essere posizionato in `./data/notifyhub-secrets.json`.

### 1.1 Configurazione per Ambiente di Test (Valori Fittizi)
Per i primi test locali o di integrazione, puoi utilizzare valori dummy. NotifyHub rileverà le credenziali mock ed emulerà gli invii scrivendo le notifiche nei log di sistema.

```json
{
  "api_keys": ["chiave_test_12345", "chiave_applicazione_media_butler"],
  "telegram": {
    "bot_token": "MOCK_TELEGRAM_BOT_TOKEN",
    "default_chat_id": "MOCK_CHAT_ID"
  },
  "discord": {
    "default_webhook_url": "https://discord.com/api/webhooks/mock_webhook"
  },
  "smtp": {
    "host": "localhost",
    "port": 1025,
    "username": "mock_user",
    "password": "mock_password",
    "sender": "notifyhub@localhost"
  }
}
```

### 1.2 Transizione alla Produzione (Valori Reali)
Per connettere realmente il microservizio alle piattaforme esterne, sostituisci i valori fittizi con quelli reali forniti dai rispettivi provider:

```json
{
  "api_keys": ["genera_una_stringa_lunga_e_casuale_come_api_key_sicura"],
  "telegram": {
    "bot_token": "1234567890:ABCdefGhIJKlmNoPQRsTUVwxyZ", 
    "default_chat_id": "-100123456789"
  },
  "discord": {
    "default_webhook_url": "https://discord.com/api/webhooks/987654321/ZYXwvuTsrqpo"
  },
  "smtp": {
    "host": "smtp.gmail.com",
    "port": 587,
    "username": "tua-email-di-servizio@gmail.com",
    "password": "tua-password-di-applicazione-google",
    "sender": "notifiche-nas@tuodominio.it"
  }
}
```

---

## 2. Compilazione e Build del Binario (Go)

NotifyHub è scritto in puro Go (Go 1.21+) senza dipendenze esterne pesanti, consentendo build velocissime e ad alte prestazioni.

### 2.1 Compilazione Nativa (OS Corrente)
Esegui questo comando per compilare localmente:
```bash
go build -o notifyhub src/main.go
```
Il binario generato sarà autocontenuto ed inferiore a **15MB**.

### 2.2 Cross-Compilazione per ARM32v7 (NAS / Raspberry Pi)
Se desideri installare NotifyHub su macchine ARM a 32 bit, compila impostando i parametri d'ambiente:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o notifyhub-arm32 src/main.go
```

---

## 3. Containerizzazione tramite Docker e Docker Compose

Il rilascio tramite Docker garantisce l'isolamento dei processi e delle dipendenze.

### 3.1 Dockerfile Multi-Stage (`Dockerfile.arm32`)
Il Dockerfile è progettato per ottimizzare le dimensioni e la sicurezza del container finale:
*   **Stage 1 (Build)**: Utilizza l'immagine ufficiale `golang:1.21-alpine` per compilare il binario.
*   **Stage 2 (Run)**: Copia il binario compilato all'interno di un'immagine `alpine` pulita e priva di strumenti di sviluppo superflui per minimizzare l'attacco di sicurezza e mantenere la RAM ridotta (<10MB).

### 3.2 Avvio del Servizio con Docker Compose

A seconda del sistema di destinazione, sono disponibili due configurazioni pre-configurate di Docker Compose:

#### A. Ambiente Local / macOS (`docker-compose.yml`)
Usa la configurazione standard che monta la directory locale `./data/`:

```bash
docker-compose up -d --build
```

#### B. Ambiente NAS (`docker-compose.nas.yml` - Flusso Ottimizzato)
Per il NAS, al fine di evitare il trasferimento di tutti i file di codice sorgente, l'applicazione adotta lo stesso flusso ad alte prestazioni di **MediaButler**:

1. **Compilazione ed Esportazione Locale (su Mac)**:
   Esegui lo script di automazione per cross-compilare l'immagine ottimizzata per l'architettura ARM32v7 del NAS ed esportarla in un pacchetto `.tar`:
   ```bash
   ./skills-codelab/app_build/scripts/deploy-nas.sh build
   ```
   Lo script genererà l'archivio in `skills-codelab/app_build/build/notifyhub-nas-arm32.tar`.

2. **Copia del pacchetto e di docker-compose sul NAS**:
   Copia sul tuo NAS **esclusivamente** il file tarball e la configurazione `docker-compose.nas.yml`:
   ```bash
   scp skills-codelab/app_build/build/notifyhub-nas-arm32.tar skills-codelab/app_build/docker-compose.nas.yml admin@<NAS_IP>:/share/Storage/Docker/NotifyHub/
   ```

3. **Caricamento e Avvio sul NAS (tramite SSH)**:
   Collegati in SSH sul NAS ed esegui i comandi per caricare l'immagine ed avviare il container:
   ```bash
   cd /share/Storage/Docker/NotifyHub/
   docker load -i notifyhub-nas-arm32.tar
   docker compose -f docker-compose.nas.yml up -d
   ```
   *(Nota: l'immagine precompilata verrà iniettata direttamente nel daemon Docker del NAS, senza alcuna necessità di compilazione locale o presenza di file sorgente sul NAS).*

---

### 3.3 Comandi Operativi Comuni

Per visualizzare i log del servizio in tempo reale:
```bash
# Per macOS / Local
docker-compose logs -f

# Per NAS
docker-compose -f docker-compose.nas.yml logs -f
```

Per arrestare il microservizio:
```bash
# Per macOS / Local
docker-compose down

# Per NAS
docker-compose -f docker-compose.nas.yml down
```
