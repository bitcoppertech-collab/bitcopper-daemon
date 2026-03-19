#!/bin/sh
# ================================================================
#  Bitcopper Sentinel — Bootstrap v2.0
#  Copyright (c) 2026 Bitcopper Technologies LLC
#  Author: Pedro Ramos Morales — Chuquicamata, Chile
#  "In cuprum veritas."
#
#  Instalación en cualquier máquina del mundo:
#  curl -fsSL https://bitcopper.tech/sentinel.sh | sh
# ================================================================

set -e

RELEASES_URL="https://bitcopper.tech/releases"
SENTINEL_DIR="$HOME/.bitcopper-sentinel"
LOG="$SENTINEL_DIR/sentinel.log"
BINARY="$SENTINEL_DIR/bitcopper-daemon"
IDENTITY_FILE="$SENTINEL_DIR/identity.json"
VERSION="2.0.0"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

banner() {
  printf "\n${CYAN}${BOLD}"
  printf "  ██████╗ ██╗████████╗ ██████╗██╗   ██╗\n"
  printf "  ██╔══██╗██║╚══██╔══╝██╔════╝██║   ██║\n"
  printf "  ██████╔╝██║   ██║   ██║     ██║   ██║\n"
  printf "  ██╔══██╗██║   ██║   ██║     ██║   ██║\n"
  printf "  ██████╔╝██║   ██║   ╚██████╗╚██████╔╝\n"
  printf "  ╚═════╝ ╚═╝   ╚═╝    ╚═════╝ ╚═════╝\n${NC}"
  printf "\n${BOLD}  Sentinel v${VERSION} — Red Térmica Global\n${NC}"
  printf "${YELLOW}  In cuprum veritas.\n\n${NC}"
}

log()  { printf "${GREEN}[✓]${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}[!]${NC} %s\n" "$1"; }
err()  { printf "${RED}[✗]${NC} %s\n" "$1"; exit 1; }
info() { printf "${CYAN}[→]${NC} %s\n" "$1"; }

# ── Detectar plataforma y elegir binario correcto ─────────────
detect_binary() {
  OS=$(uname -s)
  ARCH=$(uname -m)
  info "Detectando plataforma: $OS / $ARCH"

  # Detectar Android (Termux)
  if [ -d "/data/data/com.termux" ] || [ -n "$TERMUX_VERSION" ]; then
    PLATFORM="android"
    case "$ARCH" in
      aarch64) BINARY_NAME="bitcopper-daemon-android-arm64" ;;
      armv7l)  BINARY_NAME="bitcopper-daemon-android-arm32" ;;
      x86_64)  BINARY_NAME="bitcopper-daemon-android-x86_64" ;;
      *)       BINARY_NAME="bitcopper-daemon-android-arm64" ;;
    esac
  else
    case "$OS" in
      Darwin)
        PLATFORM="macos"
        case "$ARCH" in
          arm64)  BINARY_NAME="bitcopper-daemon-mac-silicon" ;;
          x86_64) BINARY_NAME="bitcopper-daemon-mac-intel" ;;
          *)      err "Mac no soportado: $ARCH" ;;
        esac
        ;;
      Linux)
        case "$ARCH" in
          x86_64)        BINARY_NAME="bitcopper-daemon-linux"; PLATFORM="linux" ;;
          aarch64|arm64) BINARY_NAME="bitcopper-daemon-android-arm64"; PLATFORM="arm-linux" ;;
          armv7l|armv6l) BINARY_NAME="bitcopper-daemon-android-arm32"; PLATFORM="arm-linux" ;;
          *)             BINARY_NAME="bitcopper-daemon-linux"; PLATFORM="linux" ;;
        esac
        grep -q "Raspberry" /proc/device-tree/model 2>/dev/null && PLATFORM="raspberry-pi"
        ;;
      MINGW*|MSYS*|CYGWIN*)
        PLATFORM="windows"
        BINARY_NAME="bitcopper-daemon-windows.exe"
        BINARY="$SENTINEL_DIR/bitcopper-daemon.exe"
        ;;
      *)
        err "Sistema no soportado: $OS"
        ;;
    esac
  fi

  log "Plataforma: $PLATFORM | Binario: $BINARY_NAME"
}

# ── Descargar binario ─────────────────────────────────────────
descargar_binario() {
  mkdir -p "$SENTINEL_DIR"

  if [ -f "$BINARY" ]; then
    log "Daemon ya instalado en $BINARY"
    return
  fi

  info "Descargando $BINARY_NAME desde $RELEASES_URL..."
  DOWNLOAD_URL="${RELEASES_URL}/${BINARY_NAME}"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$BINARY" || err "Error descargando: $DOWNLOAD_URL"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$DOWNLOAD_URL" -O "$BINARY" || err "Error descargando: $DOWNLOAD_URL"
  else
    err "Instala curl o wget para continuar"
  fi

  chmod +x "$BINARY"
  log "Binario listo: $BINARY"
}

# ── Generar identidad criptográfica del nodo ──────────────────
generar_identidad() {
  if [ -f "$IDENTITY_FILE" ]; then
    log "Identidad existente cargada desde $IDENTITY_FILE"
    # Validar que el archivo no está corrupto
    WALLET=$(grep '"wallet"' "$IDENTITY_FILE" 2>/dev/null | cut -d'"' -f4)
    DEVICE_ID=$(grep '"device_id"' "$IDENTITY_FILE" 2>/dev/null | cut -d'"' -f4)
    if [ -z "$WALLET" ] || [ -z "$DEVICE_ID" ]; then
      warn "Archivo de identidad corrupto — regenerando..."
      rm -f "$IDENTITY_FILE"
    else
      log "Wallet recuperada: ${WALLET:0:32}..."
      return
    fi
  fi

  info "Generando identidad criptográfica única (primera vez)..."

  # IMPORTANTE: Seed basado SOLO en hardware — sin date +%s
  # Esto garantiza que la wallet sea siempre la misma para este dispositivo
  HW_SEED="$(hostname)$(uname -srm)"

  # Enriquecer seed con MAC address si está disponible (más único por dispositivo)
  if command -v ip >/dev/null 2>&1; then
    MAC=$(ip link 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1)
    HW_SEED="${HW_SEED}${MAC}"
  elif command -v ifconfig >/dev/null 2>&1; then
    MAC=$(ifconfig 2>/dev/null | grep -oE '([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}' | head -1)
    HW_SEED="${HW_SEED}${MAC}"
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    HASH_BIN="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    HASH_BIN="shasum -a 256"
  else
    err "Necesito sha256sum o shasum instalado"
  fi

  # Device ID y Wallet derivados deterministamente del hardware
  DEVICE_ID="CU-$(printf '%s' "$HW_SEED" | $HASH_BIN | cut -c1-64)"
  WALLET="BTCU-$(printf '%s' "bitcopper-wallet-${DEVICE_ID}" | $HASH_BIN | cut -c1-64)"

  cat > "$IDENTITY_FILE" << IDEOF
{
  "device_id": "${DEVICE_ID}",
  "wallet": "${WALLET}",
  "platform": "${PLATFORM}",
  "binary": "${BINARY_NAME}",
  "installed_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "sentinel_version": "${VERSION}"
}
IDEOF

  log "Identidad creada y sellada permanentemente."
  log "Wallet:    ${WALLET:0:32}..."
  log "Device ID: ${DEVICE_ID:0:32}..."
}

# ── Eliminar instancias fantasma (wallets duplicadas) ─────────
limpiar_fantasmas() {
  info "Verificando instancias duplicadas..."
  RUNNING=$(pgrep -f "bitcopper-daemon" 2>/dev/null | wc -l | tr -d ' ')
  if [ "$RUNNING" -gt 1 ]; then
    warn "Detectadas $RUNNING instancias del daemon — eliminando duplicados..."
    pkill -f "bitcopper-daemon" 2>/dev/null || true
    sleep 2
    log "Instancias fantasma eliminadas"
  elif [ "$RUNNING" -eq 1 ]; then
    warn "Daemon ya corriendo — reiniciando con identidad correcta..."
    pkill -f "bitcopper-daemon" 2>/dev/null || true
    sleep 1
  fi
}

# ── Instalar como servicio permanente ─────────────────────────
instalar_servicio() {
  info "Instalando servicio permanente..."

  WALLET=$(grep '"wallet"' "$IDENTITY_FILE" | cut -d'"' -f4)
  DEVICE_ID=$(grep '"device_id"' "$IDENTITY_FILE" | cut -d'"' -f4)

  case "$PLATFORM" in
    macos)
      PLIST="$HOME/Library/LaunchAgents/tech.bitcopper.sentinel.plist"
      cat > "$PLIST" << PLISTEOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>tech.bitcopper.sentinel</string>
  <key>ProgramArguments</key>
  <array><string>${BINARY}</string></array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>BITCOPPER_WALLET</key><string>${WALLET}</string>
    <key>BITCOPPER_DEVICE_ID</key><string>${DEVICE_ID}</string>
  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>${LOG}</string>
  <key>StandardErrorPath</key><string>${LOG}</string>
</dict>
</plist>
PLISTEOF
      launchctl unload "$PLIST" 2>/dev/null || true
      launchctl load "$PLIST" 2>/dev/null \
        && log "Servicio macOS activo — arranca automáticamente con el sistema" \
        || warn "Ejecuta manualmente: BITCOPPER_WALLET=$WALLET $BINARY"
      ;;

    linux|raspberry-pi|arm-linux)
      if command -v systemctl >/dev/null 2>&1; then
        sudo tee /etc/systemd/system/bitcopper-sentinel.service > /dev/null << SVCEOF
[Unit]
Description=Bitcopper Sentinel — Red Termica Global
After=network.target

[Service]
ExecStart=${BINARY}
Restart=always
RestartSec=5
User=${USER}
Environment=BITCOPPER_WALLET=${WALLET}
Environment=BITCOPPER_DEVICE_ID=${DEVICE_ID}

[Install]
WantedBy=multi-user.target
SVCEOF
        sudo systemctl daemon-reload
        sudo systemctl enable bitcopper-sentinel 2>/dev/null
        sudo systemctl start bitcopper-sentinel
        log "Servicio systemd activo"
      else
        nohup env BITCOPPER_WALLET="$WALLET" BITCOPPER_DEVICE_ID="$DEVICE_ID" \
          "$BINARY" > "$LOG" 2>&1 &
        log "Daemon iniciado en background (PID: $!)"
      fi
      ;;

    android)
      # Matar instancias previas para evitar wallets fantasma
      pkill -f "bitcopper-daemon" 2>/dev/null || true
      sleep 1

      nohup env BITCOPPER_WALLET="$WALLET" BITCOPPER_DEVICE_ID="$DEVICE_ID" \
        "$BINARY" > "$LOG" 2>&1 &
      DAEMON_PID=$!
      log "Daemon Android iniciado (PID: $DAEMON_PID)"

      # Persistencia entre reinicios via ~/.bashrc y ~/.profile
      AUTOSTART_LINE="# Bitcopper Sentinel — autostart"
      DAEMON_CMD="pgrep -f bitcopper-daemon >/dev/null 2>&1 || nohup env BITCOPPER_WALLET=\"$WALLET\" BITCOPPER_DEVICE_ID=\"$DEVICE_ID\" \"$BINARY\" >> \"$LOG\" 2>&1 &"

      for RC_FILE in "$HOME/.bashrc" "$HOME/.profile"; do
        if ! grep -q "Bitcopper Sentinel" "$RC_FILE" 2>/dev/null; then
          printf "\n%s\n%s\n" "$AUTOSTART_LINE" "$DAEMON_CMD" >> "$RC_FILE"
          log "Auto-inicio registrado en $RC_FILE"
        else
          # Actualizar wallet en línea existente
          sed -i "s|BITCOPPER_WALLET=\"[^\"]*\"|BITCOPPER_WALLET=\"$WALLET\"|g" "$RC_FILE" 2>/dev/null || true
          log "Auto-inicio ya registrado en $RC_FILE (wallet actualizada)"
        fi
      done

      warn "Android TV: al reiniciar, abre Termux para que inicie automáticamente."
      warn "Para inicio sin abrir Termux manualmente, usa Termux:Boot app."
      
      # Instrucciones para Termux:Boot (inicio real sin abrir app)
      TERMUX_BOOT_DIR="$HOME/.termux/boot"
      if [ -d "$TERMUX_BOOT_DIR" ] || mkdir -p "$TERMUX_BOOT_DIR" 2>/dev/null; then
        cat > "$TERMUX_BOOT_DIR/bitcopper-sentinel.sh" << BOOTEOF
#!/data/data/com.termux/files/usr/bin/sh
# Bitcopper Sentinel — Termux:Boot autostart
sleep 10
pgrep -f bitcopper-daemon >/dev/null 2>&1 || \\
  nohup env BITCOPPER_WALLET="$WALLET" BITCOPPER_DEVICE_ID="$DEVICE_ID" \\
  "$BINARY" >> "$LOG" 2>&1 &
BOOTEOF
        chmod +x "$TERMUX_BOOT_DIR/bitcopper-sentinel.sh"
        log "Termux:Boot configurado — minería inicia al encender el dispositivo 🔥"
      fi
      ;;

    windows)
      warn "Windows: ejecuta manualmente $BINARY"
      ;;
  esac
}

# ── Main ──────────────────────────────────────────────────────
main() {
  banner
  detect_binary
  descargar_binario
  generar_identidad
  limpiar_fantasmas
  instalar_servicio

  WALLET=$(grep '"wallet"' "$IDENTITY_FILE" | cut -d'"' -f4)
  DEVICE_ID=$(grep '"device_id"' "$IDENTITY_FILE" | cut -d'"' -f4)

  printf "\n${GREEN}${BOLD}"
  printf "══════════════════════════════════════════════\n"
  printf "  Bitcopper Sentinel v%s instalado 🔥\n" "$VERSION"
  printf "══════════════════════════════════════════════\n${NC}"
  printf "\n"
  printf "  Plataforma : %s\n" "$PLATFORM"
  printf "  Wallet     : %s...\n" "${WALLET:0:32}"
  printf "  Device ID  : %s...\n" "${DEVICE_ID:0:32}"
  printf "  Log        : %s\n" "$LOG"
  printf "\n${CYAN}  Monitorea tu nodo:\n${NC}"
  printf "  tail -f %s\n" "$LOG"
  printf "\n${CYAN}  Tu calor contribuye a la red térmica global.\n${NC}"
  printf "${YELLOW}  In cuprum veritas.\n\n${NC}"
}

main
