#!/usr/bin/env bash
# deploy.sh — nasadí inh-dms na TENTO server. Spúšťaš ho ty, ručne.
#
#   ./deploy.sh --envelope ~/envelope.asc            # ostré nasadenie (e-mail)
#   ./deploy.sh --envelope ~/envelope.asc --signal   # + Signal kanál
#   ./deploy.sh --check                              # len preflight, nič nemení
#   ./deploy.sh --config-only --force-config         # len prepíš config.json
#   ./deploy.sh --envelope … --no-compose            # bez compose, čisté docker príkazy
#   ./deploy.sh --envelope ~/env.asc --test-timings  # skúšobný beh v minútach
#
# Skript nikdy nesiaha na Apache ani na postfix — vypíše, čo tam máš doplniť.
set -euo pipefail

DATA_ROOT="${INH_DATA_DIR:-/opt/inh-dms}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENVELOPE_SRC=""
WITH_SIGNAL=0
CHECK_ONLY=0
CONFIG_ONLY=0
NO_COMPOSE=0
FORCE_CONFIG=0
TEST_TIMINGS=0
PORT_APP=8088
PORT_SIGNAL=8080

say()  { printf '\n\033[1m== %s\033[0m\n' "$*"; }
ok()   { printf '   \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '   \033[33m!\033[0m %s\n' "$*"; }
die()  { printf '\n\033[31mCHYBA:\033[0m %s\n' "$*" >&2; exit 1; }

usage() { sed -n '2,${/^#/!q;s/^# \{0,1\}//;p;}' "${BASH_SOURCE[0]}"; exit 0; }

while [ $# -gt 0 ]; do
  case "$1" in
    --envelope)     ENVELOPE_SRC="${2:?--envelope potrebuje cestu}"; shift 2 ;;
    --signal)       WITH_SIGNAL=1; shift ;;
    --check)        CHECK_ONLY=1; shift ;;
    --force-config) FORCE_CONFIG=1; shift ;;
    --test-timings) TEST_TIMINGS=1; shift ;;
    --config-only)  CONFIG_ONLY=1; shift ;;
    --no-compose)   NO_COMPOSE=1; shift ;;
    --data-dir)     DATA_ROOT="${2:?--data-dir potrebuje cestu}"; shift 2 ;;
    -h|--help)      usage ;;
    *)              die "neznámy prepínač: $1 (skús --help)" ;;
  esac
done

if [ "$TEST_TIMINGS" = 1 ] && [ "$DATA_ROOT" = "/opt/inh-dms" ]; then
  DATA_ROOT="/opt/inh-dms-test"   # nech skúšobný beh nesiaha na ostrý stav
fi
DATA="$DATA_ROOT/data"
CONFIG="$DATA/config.json"
# Image beží ako 'nonroot' (uid 65532), ale /data patrí tebe a config.json je
# 0600 — kontajner teda musí bežať pod tvojím uid, inak si ho neprečíta.
DOCKER_USER=(--user "$(id -u):$(id -g)")
export INH_UID="$(id -u)" INH_GID="$(id -g)"
COMPOSE=()
export INH_DATA_DIR="$DATA_ROOT"

# ---------------------------------------------------------------- preflight --
say "Preflight"
command -v docker >/dev/null || die "docker nie je nainštalovaný"
docker info >/dev/null 2>&1 || die "docker nebeží alebo nie si v skupine 'docker' (odhlás/prihlás sa po 'usermod -aG docker \$USER')"
ok "docker beží a mám k nemu prístup bez sudo"

# Compose je príjemné, ale nie nutné — bez neho ide všetko cez čisté docker
# príkazy (kontajnery sú tie isté, len ich neriadi compose).
if [ "$NO_COMPOSE" = 1 ]; then
  COMPOSE=()
elif docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1 && docker-compose version 2>/dev/null | grep -qiE "version v?2\."; then
  COMPOSE=(docker-compose)          # samostatné v2
else
  COMPOSE=()
  # compose v1 je EOL a nevie ani ${VAR:-default} v compose súbore — radšej
  # čisté docker príkazy než tichá polovičná podpora.
  if command -v docker-compose >/dev/null 2>&1; then
    warn "docker-compose v1 ignorujem (je EOL); idem cez čisté docker príkazy"
  fi
fi
if [ ${#COMPOSE[@]} -gt 0 ]; then ok "compose: ${COMPOSE[*]}"
else warn "compose nie je (alebo --no-compose) — použijem čisté docker príkazy"; fi

command -v openssl >/dev/null || die "chýba openssl (treba na hmac_secret)"
command -v curl    >/dev/null || die "chýba curl (treba na overenie po štarte)"

if (exec 3<>/dev/tcp/127.0.0.1/25) 2>/dev/null; then ok "postfix počúva na 127.0.0.1:25"
else warn "na 127.0.0.1:25 nič nepočúva — DMS bude hlásiť poruchu, kým to nespravíš"; fi

for p in "$PORT_APP" $([ "$WITH_SIGNAL" = 1 ] && echo "$PORT_SIGNAL"); do
  if (exec 3<>/dev/tcp/127.0.0.1/"$p") 2>/dev/null; then
    if docker ps --format '{{.Names}}' | grep -qE '^inh-(dms|signal)$'; then
      warn "port $p drží bežiaci inh kontajner (reštartnem ho)"
    else
      die "port $p už niekto obsadil (a nie je to inh kontajner)"
    fi
  else ok "port $p je voľný"; fi
done

[ -f "$HERE/Dockerfile" ] || die "nenašiel som Dockerfile vedľa skriptu"
ok "zdrojáky na mieste ($HERE)"

if [ "$CHECK_ONLY" = 1 ]; then say "Preflight OK — nič som nemenil."; exit 0; fi

# ------------------------------------------------------------------- adresár --
say "Adresár $DATA_ROOT"
if [ ! -d "$DATA" ]; then
  if mkdir -p "$DATA" 2>/dev/null; then :; else
    echo "   Adresár treba vyrobiť ako root. Spustím:"
    echo "     sudo mkdir -p '$DATA' && sudo chown -R $USER: '$DATA_ROOT' && sudo chmod 700 '$DATA_ROOT'"
    read -r -p "   Pokračovať? [a/N] " a; [ "$a" = "a" ] || die "prerušené"
    sudo mkdir -p "$DATA" && sudo chown -R "$USER": "$DATA_ROOT" && sudo chmod 700 "$DATA_ROOT"
  fi
fi
chmod 700 "$DATA_ROOT" "$DATA" 2>/dev/null || true
ok "$DATA"

# -------------------------------------------------------------------- obálka --
say "Obálka (GPG ciphertext)"
if [ -n "$ENVELOPE_SRC" ]; then
  [ -f "$ENVELOPE_SRC" ] || die "obálka '$ENVELOPE_SRC' neexistuje"
  grep -q "BEGIN PGP MESSAGE" "$ENVELOPE_SRC" || die "'$ENVELOPE_SRC' nevyzerá ako ASCII-armored PGP správa"
  if [ "$ENVELOPE_SRC" -ef "$DATA/envelope.asc" ]; then
    ok "už na mieste ($DATA/envelope.asc, $(wc -c <"$DATA/envelope.asc") B)"
  else
    install -m 600 "$ENVELOPE_SRC" "$DATA/envelope.asc"
    ok "skopírovaná do $DATA/envelope.asc ($(wc -c <"$DATA/envelope.asc") B)"
  fi
elif [ -f "$DATA/envelope.asc" ]; then
  ok "už na mieste (nechávam tak)"
else
  die "chýba obálka — spusti s --envelope /cesta/envelope.asc (vyrob ju offline, viď README)"
fi

# --------------------------------------------------------------------- config --
ask() { local q="$1" def="${2:-}" v=""; read -r -p "   $q${def:+ [$def]}: " v || true; printf '%s' "${v:-$def}"; }
json_safe() { case "$1" in *'"'*|*'\'*) die "hodnota '$1' obsahuje \" alebo \\ — daj ju do config.json ručne";; esac; }

say "Konfigurácia"
if [ -f "$CONFIG" ] && [ "$FORCE_CONFIG" = 0 ]; then
  ok "config.json existuje — nechávam ho (prepíšeš cez --force-config)"
else
  [ -f "$CONFIG" ] && cp -a "$CONFIG" "$CONFIG.bak.$(date +%s)" && warn "starý config zálohovaný"
  echo "   (Enter = ponechať default. Hodnoty sa dajú kedykoľvek doeditovať v $CONFIG.)"
  BASE_URL=$(ask "Verejná URL DMS (https://…)" "https://dms.example.com")
  FROM=$(ask     "From adresa e-mailov" "dms@${BASE_URL#https://}")
  USER_MAIL=$(ask "Tvoj e-mail (check-in, alerty)")
  FRIEND_MAIL=$(ask "E-mail príjemcu obálky (technicky zdatná osoba)")
  for v in "$BASE_URL" "$FROM" "$USER_MAIL" "$FRIEND_MAIL"; do json_safe "$v"; done
  [ -n "$USER_MAIL" ] && [ -n "$FRIEND_MAIL" ] || die "user_email a friend_email sú povinné"

  USER_SIG=""; FRIEND_SIG=""; SIGNAL_BLOCK=""
  if [ "$WITH_SIGNAL" = 1 ]; then
    SIG_FROM=$(ask "Signal číslo, z ktorého DMS posiela (E.164, +…)")
    USER_SIG=$(ask "Tvoje Signal číslo (Enter = nepoužiť)")
    FRIEND_SIG=$(ask "Signal číslo príjemcu obálky (Enter = nepoužiť)")
    for v in "$SIG_FROM" "$USER_SIG" "$FRIEND_SIG"; do json_safe "$v"; done
    case "$SIG_FROM" in +*) ;; *) die "signal from_number musí začínať '+'";; esac
    SIGNAL_BLOCK=$(printf '  "signal": { "api_url": "http://127.0.0.1:%s", "from_number": "%s", "timeout": "20s" },' "$PORT_SIGNAL" "$SIG_FROM")
    [ -n "$USER_SIG" ]   && SIGNAL_BLOCK="$SIGNAL_BLOCK\n$(printf '  "user_signal": "%s",' "$USER_SIG")"
    [ -n "$FRIEND_SIG" ] && SIGNAL_BLOCK="$SIGNAL_BLOCK\n$(printf '  "friend_signal": "%s",' "$FRIEND_SIG")"
  fi

  N=$(ask "Koľko potvrdzovateľov (ľudí, ktorí vedia potvrdiť úmrtie)?" "2")
  CONFIRMERS=""
  for i in $(seq 1 "$N"); do
    echo "   -- potvrdzovateľ $i --"
    CID=$(ask   "  id (bez medzier, napr. friend)")
    CNAME=$(ask "  meno (do oslovenia v e-maile)")
    CSIG=""
    if [ "$TEST_TIMINGS" = 1 ]; then
      # Skúška nesmie nikdy napísať skutočným ľuďom: vo fáze čakania sa výzva
      # potvrdzovateľom opakuje každý reminder_interval, čo je tu pár minút.
      CMAIL="$USER_MAIL"; CSIG="$USER_SIG"
      echo "     (test: výzvy pre tohto potvrdzovateľa idú tebe, $USER_MAIL)"
    else
      CMAIL=$(ask "  e-mail")
      [ "$WITH_SIGNAL" = 1 ] && CSIG=$(ask "  Signal číslo (Enter = bez Signalu)")
    fi
    for v in "$CID" "$CNAME" "$CMAIL" "$CSIG"; do json_safe "$v"; done
    [ -n "$CID" ] && [ -n "$CMAIL" ] || die "id a e-mail potvrdzovateľa sú povinné"
    [ -n "$CONFIRMERS" ] && CONFIRMERS+=",\n"
    CONFIRMERS+=$(printf '    { "id": "%s", "name": "%s", "email": "%s"%s }' \
      "$CID" "$CNAME" "$CMAIL" "$([ -n "$CSIG" ] && printf ', "signal": "%s"' "$CSIG")")
  done

  if [ "$TEST_TIMINGS" = 1 ]; then
    warn "TEST režim: minútové intervaly; obálka aj výzvy potvrdzovateľom idú TEBE ($USER_MAIL)"
    FRIEND_MAIL="$USER_MAIL"; FRIEND_SIG="$USER_SIG"
    T_CHECKIN='"5m"'; T_REMIND='"2m"'; T_SILENCE='"10m"'; T_RELEASE='"5m"'
    T_HEALTH='"5m"';  T_WARN='"1m"';  T_ALERT='"5m"';     T_TICK='"30s"'
  else
    T_CHECKIN='"30d"'; T_REMIND='"7d"'; T_SILENCE='"60d"'; T_RELEASE='"7d"'
    T_HEALTH='"7d"';   T_WARN='"1d"';  T_ALERT='"1d"';     T_TICK='"1h"'
  fi

  SECRET=$(openssl rand -hex 32)
  umask 077
  {
    printf '{\n'
    printf '  "listen_addr": "127.0.0.1:%s",\n' "$PORT_APP"
    printf '  "public_base_url": "%s",\n' "$BASE_URL"
    printf '  "smtp_addr": "127.0.0.1:25",\n'
    printf '  "from_email": "%s",\n' "$FROM"
    printf '  "user_email": "%s",\n' "$USER_MAIL"
    printf '  "friend_email": "%s",\n' "$FRIEND_MAIL"
    [ -n "$SIGNAL_BLOCK" ] && printf '%b\n' "$SIGNAL_BLOCK"
    printf '  "confirmers": [\n%b\n  ],\n' "$CONFIRMERS"
    printf '  "envelope_path": "/data/envelope.asc",\n'
    printf '  "state_path": "/data/state.json",\n'
    printf '  "hmac_secret": "%s",\n' "$SECRET"
    printf '  "check_in_interval": %s,\n'    "$T_CHECKIN"
    printf '  "reminder_interval": %s,\n'    "$T_REMIND"
    printf '  "silence_threshold": %s,\n'    "$T_SILENCE"
    printf '  "release_delay": %s,\n'        "$T_RELEASE"
    printf '  "health_beat_interval": %s,\n' "$T_HEALTH"
    printf '  "warning_interval": %s,\n'     "$T_WARN"
    printf '  "alert_interval": %s,\n'       "$T_ALERT"
    printf '  "tick_interval": %s\n'         "$T_TICK"
    printf '}\n'
  } > "$CONFIG"
  chmod 600 "$CONFIG"
  ok "zapísaný $CONFIG"
  warn "hmac_secret ulož do KeePass DB — bez neho sa zmenia check-in odkazy"
fi

# ------------------------------------------------------------------ validácia --
say "Validácia configu"
if docker image inspect inh-dms >/dev/null 2>&1; then
  if OUT=$(docker run --rm "${DOCKER_USER[@]}" -v "$DATA":/data inh-dms --validate 2>&1); then ok "${OUT#*config OK: }"
  else die "config neprešiel: $OUT"; fi
else
  warn "image inh-dms zatiaľ neexistuje — config sa overí hneď po builde"
fi

if [ "$CONFIG_ONLY" = 1 ]; then say "Config hotový ($CONFIG). Kontajner som nespúšťal."; exit 0; fi

# Skúšobná inštancia musí štartovať z known-good stavu: starý state.json (napr.
# z predošlého pokusu) by ju rovno hodil do fázy „čakám na potvrdenie".
if [ "$TEST_TIMINGS" = 1 ] && [ -f "$DATA/state.json" ]; then
  rm -f "$DATA/state.json"
  warn "test: starý state.json zmazaný — skúška začína od nuly"
fi

# ------------------------------------------------------- kontajnery (2 cesty) --
# S compose ich riadi compose, bez neho ide to isté cez čisté docker príkazy.
have_compose() { [ ${#COMPOSE[@]} -gt 0 ]; }

img_build() {
  if have_compose; then ( cd "$HERE" && "${COMPOSE[@]}" build dms )
  else docker build -t inh-dms "$HERE"; fi
}

svc_up() {
  if have_compose; then
    if [ "$WITH_SIGNAL" = 1 ]; then ( cd "$HERE" && "${COMPOSE[@]}" --profile signal up -d )
    else                            ( cd "$HERE" && "${COMPOSE[@]}" up -d dms ); fi
    return
  fi
  docker rm -f inh-dms >/dev/null 2>&1 || true
  docker run -d --name inh-dms --restart unless-stopped "${DOCKER_USER[@]}" \
    --network host -v "$DATA":/data inh-dms >/dev/null
  if [ "$WITH_SIGNAL" = 1 ]; then
    mkdir -p "$DATA_ROOT/signal"
    docker rm -f inh-signal >/dev/null 2>&1 || true
    docker run -d --name inh-signal --restart unless-stopped \
      -e MODE=native -e "AUTO_RECEIVE_SCHEDULE=0 4 * * *" \
      -p "127.0.0.1:$PORT_SIGNAL:8080" \
      -v "$DATA_ROOT/signal":/home/.local/share/signal-cli \
      bbernhard/signal-cli-rest-api:latest >/dev/null
  fi
}

svc_logs() {
  if have_compose; then ( cd "$HERE" && "${COMPOSE[@]}" logs --no-log-prefix dms 2>/dev/null || true )
  else docker logs inh-dms 2>&1 || true; fi
}

if have_compose; then
  LOGS_CMD="${COMPOSE[*]} logs dms"; DOWN_CMD="${COMPOSE[*]} down"
else
  LOGS_CMD="docker logs inh-dms"
  DOWN_CMD="docker rm -f inh-dms"
  [ "$WITH_SIGNAL" = 1 ] && DOWN_CMD="docker rm -f inh-dms inh-signal"
fi

# ---------------------------------------------------------------------- štart --
say "Build & štart"
img_build
if OUT=$(docker run --rm "${DOCKER_USER[@]}" -v "$DATA":/data inh-dms --validate 2>&1); then ok "config: ${OUT#*config OK: }"
else die "config neprešiel: $OUT"; fi
svc_up
ok "kontajnery bežia"

# -------------------------------------------------------------------- kontrola --
say "Overenie"
for i in $(seq 1 30); do
  BODY=$(curl -fsS --max-time 2 "http://127.0.0.1:$PORT_APP/" 2>/dev/null) && break || sleep 1
done
case "${BODY:-}" in *"Služba beží"*) ok "HTTP na 127.0.0.1:$PORT_APP odpovedá";; *) die "služba neodpovedá — pozri: $LOGS_CMD";; esac

LOGS=$(svc_logs | tail -40)
case "$LOGS" in *"listening on"*) ok "štart v logu";; *) warn "v logu nevidím 'listening' — pozri logy";; esac
case "$LOGS" in *failed*) warn "v logu je 'failed' — pozri: $LOGS_CMD";; esac
if [ -f "$DATA/state.json" ]; then
  ok "stav sa zapisuje ($DATA/state.json)"
  PHASE=$(sed -n 's/.*"phase"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$DATA/state.json" | head -1)
  case "$PHASE" in
    normal) ok "fáza: normal (DMS ťa považuje za živého)" ;;
    "")     warn "fázu sa nepodarilo prečítať zo state.json" ;;
    *)      warn "fáza: $PHASE — DMS NIE je v pokoji! Klikni check-in odkaz nižšie, tým sa to zruší." ;;
  esac
else
  warn "state.json ešte nie je — počkaj na prvý tik"
fi

SECRET=$(sed -n 's/.*"hmac_secret": "\([^"]*\)".*/\1/p' "$CONFIG")
BASE_URL=$(sed -n 's/.*"public_base_url": "\([^"]*\)".*/\1/p' "$CONFIG")
TOKEN=$(printf 'checkin' | openssl dgst -sha256 -hmac "$SECRET" -r | cut -d' ' -f1)

say "Čo ešte musíš spraviť ty"
cat <<TXT
1) Apache (potrebuje root) — vhost, ktorý pustí $BASE_URL na 127.0.0.1:$PORT_APP:

     a2enmod proxy proxy_http headers
     <VirtualHost *:443>
       ServerName ${BASE_URL#https://}
       ProxyPreserveHost On
       ProxyPass        / http://127.0.0.1:$PORT_APP/
       ProxyPassReverse / http://127.0.0.1:$PORT_APP/
       RequestHeader set X-Forwarded-Proto https
     </VirtualHost>
   (TLS doplní certbot; potom: systemctl reload apache2)

2) Ulož si check-in odkaz do záložiek a do KeePass DB — je stabilný:

     $BASE_URL/checkin?token=$TOKEN

3) Skontroluj, že ti prišiel e-mail „[DMS] v poriadku" (posiela sa hneď po štarte).
   Ak neprišiel: $LOGS_CMD
TXT
[ "$WITH_SIGNAL" = 1 ] && cat <<TXT
4) Linkni Signal (raz):
     xdg-open http://127.0.0.1:$PORT_SIGNAL/v1/qrcodelink?device_name=inh-dms
     # naskenuj v Signale: Nastavenia → Prepojené zariadenia → +
     curl -s http://127.0.0.1:$PORT_SIGNAL/v1/accounts   # musí obsahovať tvoje from_number
TXT
[ "$TEST_TIMINGS" = 1 ] && cat <<TXT

POZOR: toto je TEST inštancia ($DATA_ROOT), intervaly sú v minútach a obálka
sa pošle tebe. Keď doskúšaš: $DOWN_CMD && rm -rf $DATA_ROOT
TXT
say "Hotovo."
