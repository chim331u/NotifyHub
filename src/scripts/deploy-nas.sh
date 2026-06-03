#!/usr/bin/env bash

# ==============================================================================
# NotifyHub NAS Deploy Automation Script
# ==============================================================================
# Automatizza la compilazione multi-piattaforma Docker locale (su Mac),
# il packaging tar, la copia multiplexata SSH e il caricamento sul NAS.
#
# Utilizzo:
#   Locale (Mac): ./scripts/deploy-nas.sh build
#   Deploy Remoto: ./scripts/deploy-nas.sh deploy
#   Sul NAS:      ./deploy-nas.sh run-nas
# ==============================================================================

set -euo pipefail

# Colori del testo per la formattazione
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # Nessun Colore

# Configurazione standard del NAS QNAP
NAS_IP="192.168.1.5"
NAS_USER="admin"
NAS_PORT="22"
NAS_PATH="/share/Storage/Docker/NotifyHub"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    echo "NotifyHub Deployment Automation Utility"
    echo ""
    echo "Uso:"
    echo "  $0 [options] <command>"
    echo ""
    echo "Comandi:"
    echo "  build             - Local: Compilazione Docker ARM32v7 locale e pacchetto tar"
    echo "  deploy            - Local to Remote: Compila l'immagine, la copia via SSH ed esegue il deploy sul NAS"
    echo "  run-nas           - QNAP: Carica il pacchetto tar ed avvia docker-compose"
    echo "  help              - Mostra questo aiuto"
    echo ""
    echo "Opzioni (per deploy/build):"
    echo "  --ip <ip>         - Indirizzo IP del NAS QNAP (default: ${NAS_IP})"
    echo "  --user <user>     - Nome utente SSH del NAS QNAP (default: ${NAS_USER})"
    echo "  --port <port>     - Porta SSH del NAS QNAP (default: ${NAS_PORT})"
    echo "  --path <path>     - Cartella di destinazione sul NAS QNAP (default: ${NAS_PATH})"
    echo ""
}

# 1. Compilazione Locale
local_build() {
    log_info "Avvio fase di compilazione incrociata Docker ARM32v7 locale..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker non è installato su questo host Mac."
        exit 1
    fi

    if ! docker buildx version &> /dev/null; then
        log_error "Docker Buildx non è disponibile. Richiesto per compilazione multi-piattaforma."
        exit 1
    fi

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
    OUT_DIR="${APP_ROOT}/build"
    TAR_OUT="${OUT_DIR}/notifyhub-nas-arm32.tar"

    mkdir -p "${OUT_DIR}"

    log_info "Verifica/Creazione dell'istanza buildx builder..."
    if ! docker buildx inspect notifyhub-builder &> /dev/null; then
        log_info "Creazione nuovo builder 'notifyhub-builder'..."
        docker buildx create --name notifyhub-builder --use
    else
        docker buildx use notifyhub-builder
    fi

    log_info "Compilazione dell'immagine Docker per linux/arm/v7 (ARM32v7)..."
    log_info "Context: ${APP_ROOT}"
    
    docker buildx build \
        --platform linux/arm/v7 \
        -f "${APP_ROOT}/Dockerfile.arm32" \
        -t notifyhub:latest \
        --load \
        "${APP_ROOT}"

    log_success "Immagine Docker compilata con successo: notifyhub:latest (ARM32v7)"

    log_info "Esportazione dell'immagine in archivio tarball: ${TAR_OUT}..."
    docker save notifyhub:latest -o "${TAR_OUT}"

    log_success "Pacchetto Tar creato con successo in: ${TAR_OUT}"
    
    if command -v du &> /dev/null; then
        log_info "Dimensione pacchetto tar: $(du -sh "${TAR_OUT}" | cut -f1)"
    fi
}

# 2. Deployment Remoto (Mac -> NAS QNAP)
remote_deploy() {
    log_info "Avvio del deploy automatico remoto verso il NAS QNAP..."
    
    # 1. Compila l'immagine locally
    local_build

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
    TAR_OUT="${APP_ROOT}/build/notifyhub-nas-arm32.tar"
    COMPOSE_FILE="${APP_ROOT}/docker-compose.nas.yml"
    DEPLOY_SCRIPT="${APP_ROOT}/scripts/deploy-nas.sh"
    LOCAL_SECRETS="${APP_ROOT}/data/notifyhub-secrets.json"

    if [ ! -f "${TAR_OUT}" ]; then
        log_error "L'archivio locale per la build non esiste: ${TAR_OUT}"
        exit 1
    fi

    # Socket MUX per multiplexing SSH
    MUX_SOCKET="/tmp/ssh_mux_notifyhub_${NAS_IP}_${NAS_PORT}"

    # Funzione di pulizia per chiudere la sessione multiplexata all'uscita
    cleanup() {
        if [ -S "${MUX_SOCKET}" ]; then
            echo ""
            log_info "Chiusura della connessione master SSH..."
            ssh -p "${NAS_PORT}" -S "${MUX_SOCKET}" -O exit "${NAS_USER}@${NAS_IP}" 2>/dev/null || true
        fi
    }
    trap cleanup EXIT

    log_info "Stabilendo connessione SSH Master verso il NAS (${NAS_IP}:${NAS_PORT})..."
    log_warn "👉 Inserisci la password per l'utente '${NAS_USER}' del NAS (richiesta SOLO ORA):"
    ssh -p "${NAS_PORT}" -M -S "${MUX_SOCKET}" -fN "${NAS_USER}@${NAS_IP}"

    log_info "Creazione cartella di deploy sul NAS: ${NAS_PATH}..."
    ssh -S "${MUX_SOCKET}" -p "${NAS_PORT}" "${NAS_USER}@${NAS_IP}" "mkdir -p ${NAS_PATH}"

    log_info "Copia del pacchetto immagine e della configurazione sul NAS tramite SCP..."
    scp -o ControlPath="${MUX_SOCKET}" -P "${NAS_PORT}" "${TAR_OUT}" "${COMPOSE_FILE}" "${DEPLOY_SCRIPT}" "${NAS_USER}@${NAS_IP}:${NAS_PATH}/"
    log_success "File trasferiti con successo sul NAS."

    # Gestione intelligente dei segreti (secrets.json)
    log_info "Verifica della presenza del file dei segreti sul NAS..."
    if ! ssh -S "${MUX_SOCKET}" -p "${NAS_PORT}" "${NAS_USER}@${NAS_IP}" "[ -f ${NAS_PATH}/data/notifyhub-secrets.json ]"; then
        log_warn "File dei segreti non trovato sul NAS in ${NAS_PATH}/data/notifyhub-secrets.json."
        if [ -f "${LOCAL_SECRETS}" ]; then
            log_info "Trovato file segreti locale sul Mac. Caricamento in corso..."
            ssh -S "${MUX_SOCKET}" -p "${NAS_PORT}" "${NAS_USER}@${NAS_IP}" "mkdir -p ${NAS_PATH}/data"
            scp -o ControlPath="${MUX_SOCKET}" -P "${NAS_PORT}" "${LOCAL_SECRETS}" "${NAS_USER}@${NAS_IP}:${NAS_PATH}/data/"
            log_success "File dei segreti caricato con successo."
        else
            log_warn "Nessun file segreti trovato localmente in ${LOCAL_SECRETS}."
            log_warn "Il contenitore potrebbe avviarsi in modalità simulata o con configurazione vuota."
        fi
    else
        log_success "File dei segreti già presente sul NAS."
    fi

    log_info "Esecuzione remota dello script sul NAS..."
    ssh -S "${MUX_SOCKET}" -p "${NAS_PORT}" "${NAS_USER}@${NAS_IP}" << EOF
      # Aggiunge i percorsi di QNAP e Container Station al PATH
      for qpath in /share/*/.qpkg/container-station/bin /share/*/.qpkg/container-station/sbin /usr/local/bin /usr/local/sbin; do
        if [ -d "\$qpath" ]; then
          export PATH="\$qpath:\$PATH"
        fi
      done

      cd "${NAS_PATH}"
      # Ridefinisce i log all'interno del contesto shell remoto
      log_info() { echo -e "\033[0;34m[INFO]\033[0m \$1"; }
      log_success() { echo -e "\033[0;32m[SUCCESS]\033[0m \$1"; }
      log_warn() { echo -e "\033[1;33m[WARN]\033[0m \$1"; }
      log_error() { echo -e "\033[0;31m[ERROR]\033[0m \$1"; }
      
      # Lancia l'avvio locale sul NAS
      bash deploy-nas.sh run-nas --path "${NAS_PATH}"
EOF

    log_success "Pipeline di deploy automatico completata con successo!"
}

# 3. Avvio sul NAS QNAP (Eseguito all'interno del NAS)
qnap_nas_run() {
    log_info "Fase di caricamento e avvio sul NAS QNAP..."
    
    # Aggiungi i percorsi di QNAP e Container Station al PATH nel caso non siano già presenti
    for qpath in /share/*/.qpkg/container-station/bin /share/*/.qpkg/container-station/sbin /usr/local/bin /usr/local/sbin; do
        if [ -d "$qpath" ]; then
            export PATH="$qpath:$PATH"
        fi
    done

    # A. Verifica requisiti sul NAS
    if ! command -v docker &> /dev/null; then
        log_error "Docker non è installato su questo NAS QNAP. Installa Container Station."
        exit 1
    fi

    # Crea la cartella dati se non esiste
    mkdir -p "${NAS_PATH}/data"

    # B. Carica l'immagine dal pacchetto tarball
    TAR_PATH="./notifyhub-nas-arm32.tar"
    if [ ! -f "${TAR_PATH}" ]; then
        TAR_PATH="${NAS_PATH}/notifyhub-nas-arm32.tar"
    fi

    if [ -f "${TAR_PATH}" ]; then
        log_info "Caricamento dell'immagine Docker dall'archivio: ${TAR_PATH}..."
        docker load -i "${TAR_PATH}"
        log_success "Immagine caricata con successo in Container Station!"
    else
        log_warn "Archivio tar '${TAR_PATH}' non trovato. Si assume che 'notifyhub:latest' sia già cacheata."
    fi

    # C. Trova il tool compose
    COMPOSE_CMD="docker compose"
    if ! docker compose version &> /dev/null; then
        if command -v docker-compose &> /dev/null; then
            COMPOSE_CMD="docker-compose"
        else
            log_error "Docker Compose non trovato. Installa docker-compose sul NAS."
            exit 1
        fi
    fi

    # D. Avvio del servizio tramite Compose
    COMPOSE_FILE="./docker-compose.nas.yml"
    if [ ! -f "${COMPOSE_FILE}" ]; then
        COMPOSE_FILE="${NAS_PATH}/docker-compose.nas.yml"
    fi

    if [ ! -f "${COMPOSE_FILE}" ]; then
        log_error "File di configurazione 'docker-compose.nas.yml' non trovato."
        exit 1
    fi

    log_info "Avvio del contenitore NotifyHub tramite ${COMPOSE_CMD}..."
    ${COMPOSE_CMD} -f "${COMPOSE_FILE}" up -d

    log_success "NotifyHub è attivo ed in esecuzione in background!"
    echo -e "\n=============================================================================="
    log_success "DEPLOYMENT SUL NAS COMPLETATO!"
    echo -e "Il servizio è attivo ai seguenti indirizzi:"
    echo -e "   - ${BLUE}API Invio Notifiche:${NC} http://<NAS_IP>:30111/api/notify"
    echo -e "   - ${BLUE}Health Check API:${NC}   http://<NAS_IP>:30111/api/health"
    echo -e "==============================================================================\n"
}

# Routing principale degli argomenti
COMMAND=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --ip)
            NAS_IP="$2"
            shift 2
            ;;
        --user)
            NAS_USER="$2"
            shift 2
            ;;
        --port)
            NAS_PORT="$2"
            shift 2
            ;;
        --path)
            NAS_PATH="$2"
            shift 2
            ;;
        build|run-nas|deploy|help)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Argomento sconosciuto: $1"
            show_help
            exit 1
            ;;
    esac
done

if [ -z "${COMMAND}" ] || [ "${COMMAND}" = "help" ]; then
    show_help
    exit 0
fi

case "${COMMAND}" in
    build)
        local_build
        ;;
    deploy)
        remote_deploy
        ;;
    run-nas)
        qnap_nas_run
        ;;
esac
