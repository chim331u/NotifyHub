# Technical Specification: Unified Messaging Microservice (NotifyHub)

**Autore**: Product Manager Agent (Antigravity Team)  
**Stato**: In Attesa di Approvazione (Revisione 4)  
**Destinazione**: `production_artifacts/Technical_Specification.md`

---

## 1. Executive Summary

L'obiettivo di questa specifica tecnica è definire la progettazione e l'architettura di **NotifyHub**, un microservizio di messaggistica e notifiche centralizzato, leggero ed indipendente. 

Il servizio agirà come un **Gateway di Notifica Centralizzato** autonomo per tutta la rete locale dell'utente. Esporrà un'API REST protetta ad alte prestazioni, gestirà le chiavi di sicurezza in modo isolato ed invierà asincronamente notifiche su canali multipli (Telegram, Discord, Email/SMTP, ecc.), servendo simultaneamente più progetti (MediaButler, HouseLedger, ecc.).

Il file `Technical_Specification.md` funge da **Entry Point principale della nuova applicazione**, fungendo da riferimento assoluto per il comportamento, la documentazione e la definizione di tutti i componenti del sistema.

---

## 2. Requisiti & Architettura del Servizio

### 2.1 Requisiti Funzionali
*   **API di Invio Unificata**: Endpoint `POST /api/notify` protetto che accetta JSON semantico per l'invio asincrono multicanale.
*   **Supporto Canali Multipli**:
    *   **Telegram**: Notifiche in formato HTML via Bot API.
    *   **Discord**: Integrazione canali tramite Webhooks (Markdown).
    *   **SMTP (Email)**: Supporto e-mail per report riassuntivi o anomalie critiche.
*   **Resilienza & Rate-Limiting**: Coda in-memory persistente per scaglionare i messaggi e rispettare i limiti delle API esterne (es. i limiti di Telegram di 30 messaggi al secondo).
*   **Gestione Code**: Logica di ritentativo automatico (*exponential backoff retry*) in caso di fallimento temporaneo delle API esterne.

### 2.2 Requisiti Non Funzionali
*   **Footprint di Risorse Ultra-Basso (Go Engine)**:
    *   Utilizzo RAM < **10MB** in esercizio standard.
    *   Eseguibile compilato autocontenuto < **15MB**.
*   **Autenticazione Leggera**: Validazione obbligatoria dell'header `X-API-Key` con chiave precondivisa memorizzata nei segreti locali.
*   **Deployment Totalmente Isolato**: File `docker-compose.yml` e workspace completamente **separati** da qualsiasi altra applicazione per garantire l'isolamento dei processi e la massima portabilità.

---

## 3. Gestione della Versione (SemVer Automatizzato)

Il microservizio adotterà lo standard **Semantic Versioning (SemVer)** nel formato `major.minor.patch` (es. `1.2.7`). La gestione della versione sarà governata dalle seguenti regole per preservare l'integrità del ciclo di vita:

1.  **Stato della Versione**: La versione attiva è memorizzata in un file di testo piatto denominato `VERSION` presente nella radice del progetto (contenente ad esempio `1.0.0`).
2.  **Auto-Incremento `patch` su Git Push**:
    *   Viene configurato un Git Hook locale di tipo `pre-push` all'interno della cartella `.git/hooks/pre-push`.
    *   Ad ogni comando `git push`, l'hook si attiva automaticamente ed effettua le seguenti azioni:
        1. Legge il valore dal file `VERSION`.
        2. Incrementa di 1 l'ultimo numero (la `patch`, es. da `1.2.6` a `1.2.7`).
        3. Aggiorna il file `VERSION`.
        4. Esegue un commit silenzioso del file `VERSION` prima di procedere con l'effettivo push remoto.
3.  **Incremento `minor` per Modifiche Corpose**:
    *   In caso di modifiche sostanziali (aggiunta di canali, refactoring corposi), l'incremento di `minor` (secondo elemento, es. da `1.2.7` a `1.3.0`, azzerando la patch) avverrà lanciando uno script locale dedicato:
      ```bash
      ./scripts/bump-version.sh minor
      ```
    *   Questo script aggiorna il file `VERSION` ed effettua il commit automatico.
4.  **Incremento `major` solo Manuale**:
    *   Il primo elemento (`major`) potrà essere modificato **esclusivamente dall'utente** modificando manualmente il file `VERSION` (es. da `1.3.5` a `2.0.0`), a garanzia del pieno controllo sulle major release del servizio.

---

## 4. Architettura Documentale e di Tracciamento

Per garantire manutenibilità e chiarezza, il progetto implementerà una suite documentale rigorosa e standardizzata, strutturata come segue:

### 4.1 I 3 Pilastri Documentali
1.  **Technical_Specification.md** (Entry Point): Questo file. Descrive l'architettura tecnica globale, i requisiti, i protocolli di comunicazione e i criteri di sicurezza.
2.  **User_Manual.md**: Manuale utente destinato a chi consuma il servizio. Contiene gli esempi di payload JSON per `POST /api/notify` su Telegram, Discord e SMTP, i codici di errore HTTP restituiti e le modalità di utilizzo delle API Key.
3.  **Deployment_Manual.md**: Manuale operativo per amministratori di sistema. Descrive le modalità di build (cross-compilazione per ARM32v7), la configurazione delle variabili d'ambiente, la gestione dei volumi in Docker e la configurazione del file `docker-compose.yml` separato.

### 4.2 Tracciamento delle Implementazioni (`task.md`)
Verrà mantenuto un file `task.md` nella radice del progetto, aggiornato in tempo reale/automatico ad ogni modifica o aggiunta implementativa di rilievo.
*   **Ordinamento Bottom-Top (Cronologico Inverso)**: Per facilitare la lettura, le modifiche più recenti e le nuove implementazioni verranno accodate **in alto** (in cima al file), spingendo verso il basso lo storico dei task completati in precedenza.
*   **Struttura del file `task.md`**:
    ```markdown
    # Registro dei Task e Stato Implementativo (NotifyHub)

    ## [Ultima Data Modifica] - Versione [X.Y.Z]
    - [x] Descrizione dell'ultima modifica/funzionalità introdotta più di recente.
    - [x] Dettaglio dell'implementazione.

    ## [Data Precedente] - Versione [A.B.C]
    - [x] Descrizione della funzionalità precedente.
    ```

---

## 5. Deployment Isolato & Struttura delle Cartelle

`NotifyHub` risiederà in un workspace indipendente, con un file `docker-compose.yml` dedicato per consentire avvii, arresti e aggiornamenti isolati da qualsiasi altro progetto (come MediaButler o HouseLedger).

Il progetto adotterà una struttura a cartelle integrata che include anche la gestione degli agenti intelligenti ereditando i moduli operativi da `skills-codelab`.

### Struttura della directory di `NotifyHub`:
```
NotifyHub/
├── docs/                      # Cartella centralizzata per tutti i documenti *.md
│   ├── Technical_Specification.md # ENTRY POINT del progetto
│   ├── User_Manual.md            # Manuale utente e guide all'integrazione API
│   ├── Deployment_Manual.md      # Istruzioni di build, cross-compilation e Docker
│   └── task.md                  # Registro storico implementazioni (nuovo in alto)
├── skills-codelab/
│   ├── app_build/             # Directory con il codice sorgente compilabile ed eseguibile
│   │   ├── VERSION                  # File piatto contenente la versione SemVer (es. 1.0.0)
│   │   ├── docker-compose.yml       # Configurazione Docker isolata (macOS/Local)
│   │   ├── docker-compose.nas.yml   # Configurazione Docker isolata (NAS)
│   │   ├── Dockerfile.arm32         # Dockerfile multi-stage ottimizzato per ARM32v7 / NAS
│   │   ├── config/
│   │   │   ├── config.go
│   │   │   └── secrets.go           # Caricamento segreti locali in modo polimorfico
│   │   ├── scripts/
│   │   │   ├── bump-version.sh      # Script bash per incremento controllato di minor/major
│   │   │   └── deploy-nas.sh        # Script di automazione compilazione incrociata per NAS
│   │   ├── build/                 # Sottocartella per gli output di compilazione locali (esclusa da git)
│   │   └── src/
│   │       └── main.go              # Codice sorgente del microservizio in Go
│   └── .agents/               # Folder per la gestione degli agenti (come in skills-codelab)
│       ├── agents.md          # Definizione dei ruoli degli agenti autonomi
│       ├── skills/            # Istruzioni operative specifiche per fase
│       └── workflows/         # Flussi di esecuzione per la pipeline
└── .git/hooks/pre-push        # Hook git locale per incremento automatico di patch
```

### Proposta `docker-compose.yml` isolato:
```yaml
version: '3.8'

services:
  notifyhub:
    image: notifyhub:latest
    container_name: notifyhub-service
    build:
      context: .
      dockerfile: Dockerfile.arm32
    restart: unless-stopped
    ports:
      - "30180:30180" # Porta locale di ascolto di NotifyHub
    volumes:
      # Volume per persistere i dati in modo isolato
      - ./data:/app/data
      # File per la lettura locale e sicura delle API Key
      - ./data/notifyhub-secrets.json:/app/data/notifyhub-secrets.json:ro
    environment:
      - NOTIFYHUB_ENV=production
      - NOTIFYHUB_PORT=30180
```
