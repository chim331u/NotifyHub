#!/usr/bin/env bash

# ==============================================================================
# NotifyHub NAS Deploy Automation Script
# ==============================================================================
# Automatizza la compilazione multi-piattaforma Docker locale (su Mac),
# il packaging tar e le istruzioni di caricamento sul NAS.
#
# Utilizzo:
#   Locale (Mac): ./scripts/deploy-nas.sh build
#   Sul NAS:      docker load -i notifyhub-nas-arm32.tar && docker compose -f docker-compose.nas.yml up -d
# ==============================================================================

set -euo pipefail

# Colori del testo per la formattazione
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # Nessun Colore

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
    echo "  $0 build          - Local: Compilazione Docker ARM32v7 locale e pacchetto tar"
    echo "  $0 help           - Mostra questo aiuto"
    echo ""
}

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

    echo -e "\n=============================================================================="
    log_success "FASE LOCALE COMPLETATA CON SUCCESSO!"
    echo -e "Procedi come segue per distribuire sul tuo NAS:"
    echo -e ""
    echo -e "1. Copia l'archivio tarball e il file docker-compose.nas.yml sul tuo NAS:"
    echo -e "   ${YELLOW}scp skills-codelab/app_build/build/notifyhub-nas-arm32.tar skills-codelab/app_build/docker-compose.nas.yml admin@<NAS_IP>:/share/Storage/Docker/NotifyHub/${NC}"
    echo -e ""
    echo -e "2. Connettiti via SSH sul NAS e vai alla directory:"
    echo -e "   ${YELLOW}ssh admin@<NAS_IP>${NC}"
    echo -e "   ${YELLOW}cd /share/Storage/Docker/NotifyHub/${NC}"
    echo -e ""
    echo -e "3. Carica l'immagine nel Docker del NAS ed avvia il servizio:"
    echo -e "   ${YELLOW}docker load -i notifyhub-nas-arm32.tar${NC}"
    echo -e "   ${YELLOW}docker compose -f docker-compose.nas.yml up -d${NC}"
    echo -e "==============================================================================\n"
}

# Rotte principali
if [ $# -lt 1 ]; then
    show_help
    exit 1
fi

case "$1" in
    build)
        local_build
        ;;
    help)
        show_help
        ;;
    *)
        log_error "Argomento sconosciuto: $1"
        show_help
        exit 1
        ;;
esac
