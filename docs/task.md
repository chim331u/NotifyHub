# Registro dei Task e Stato Implementativo (NotifyHub)

## 2026-06-01 - Versione 1.0.5
- [x] Modifica della porta esposta su NAS da `30180` a `30111` (sia interna che esterna) in `docker-compose.nas.yml`.

## 2026-06-01 - Versione 1.0.4
- [x] Risoluzione del problema di creazione della rete virtuale su QNAP Container Station forzando `network_mode: "bridge"` all'interno di `docker-compose.nas.yml` per bypassare i bug di `qnetwork-tool`.

## 2026-06-01 - Versione 1.0.3
- [x] Implementazione dello script di build ed esportazione locale `deploy-nas.sh`.
- [x] Rimozione del blocco `build` da `docker-compose.nas.yml` per consentire l'avvio tramite immagine caricata da tarball.
- [x] Integrazione della cartella di compilazione `build/` in `.gitignore`.
- [x] Documentazione dettagliata del flusso compilazione locale + `docker load` nel manuale di deployment (`Deployment_Manual.md`).

## 2026-06-01 - Versione 1.0.2
- [x] Creazione di `docker-compose.nas.yml` con percorsi dei volumi configurati per l'archiviazione del NAS.
- [x] Aggiornamento del Manuale di Deployment (`Deployment_Manual.md`) con comandi e configurazioni dedicate per ambiente NAS.
- [x] Sincronizzazione dell'albero dei file in `Technical_Specification.md`.


## 2026-06-01 - Versione 1.0.0
- [x] Inizializzazione della struttura del progetto in `skills-codelab/app_build/`.
- [x] Configurazione del file `VERSION` a `1.0.0`.
- [x] Creazione del registro dei task interno `task.md` con ordinamento Bottom-Top.
- [x] Creazione dei manuali d'uso (`User_Manual.md`) e deployment (`Deployment_Manual.md`).
- [x] Configurazione di Docker (`docker-compose.yml`, `Dockerfile.arm32`).
- [x] Sviluppo degli script di auto-incremento (`bump-version.sh`, Git Hook `pre-push`).
- [x] Implementazione del motore Go con coda in-memory concorrente, autenticazione `X-API-Key` e integrazioni mock per canali esterni.
- [x] Scrittura dei test unitari (`src/main_test.go`) e validazione dell'integrazione con successo.
