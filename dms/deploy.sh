#!/usr/bin/env bash
# deploy.sh - deploys inh-dms on THIS server. You run it, by hand.
#
#   ./deploy.sh --envelope ~/envelope.asc            # real deployment (e-mail)
#   ./deploy.sh --envelope ~/envelope.asc --signal   # + Signal channel
#   ./deploy.sh --envelope passphrase=~/a.asc --envelope credentials=~/b.asc
#                                                    # several envelopes, each to its own recipients
#   ./deploy.sh --check                              # preflight only, changes nothing
#   ./deploy.sh --config-only --force-config         # rewrite config.json only
#   ./deploy.sh --envelope ... --no-compose          # no compose, plain docker commands
#   ./deploy.sh --envelope ~/env.asc --test-timings  # rehearsal, minute-scale intervals
#
# The script never touches Apache or postfix; it prints what you must add there.
set -euo pipefail

DATA_ROOT="${INH_DATA_DIR:-/opt/inh-dms}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_IDS=()
ENV_SRCS=()
ENV_DSTS=()
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
    --envelope)
      spec="${2:?--envelope needs a path (or id=path)}"
      case "$spec" in
        *=*) ENV_IDS+=("${spec%%=*}"); ENV_SRCS+=("${spec#*=}") ;;
        *)   ENV_IDS+=("default");     ENV_SRCS+=("$spec") ;;
      esac
      shift 2 ;;
    --signal)       WITH_SIGNAL=1; shift ;;
    --check)        CHECK_ONLY=1; shift ;;
    --force-config) FORCE_CONFIG=1; shift ;;
    --test-timings) TEST_TIMINGS=1; shift ;;
    --config-only)  CONFIG_ONLY=1; shift ;;
    --no-compose)   NO_COMPOSE=1; shift ;;
    --data-dir)     DATA_ROOT="${2:?--data-dir needs a path}"; shift 2 ;;
    -h|--help)      usage ;;
    *)              die "unknown flag: $1 (try --help)" ;;
  esac
done

# The rehearsal instance must be separate in EVERYTHING that can collide: data
# directory, container names, ports and the compose project. Otherwise an `up -d`
# or `rm -f` from a rehearsal would take down the live switch, and the proxy
# would start pointing at the rehearsal.
NAME_DMS="inh-dms"
NAME_SIGNAL="inh-signal"
PROJECT="inh-dms"
if [ "$TEST_TIMINGS" = 1 ]; then
  [ "$DATA_ROOT" = "/opt/inh-dms" ] && DATA_ROOT="/opt/inh-dms-test"
  NAME_DMS="inh-dms-test"
  NAME_SIGNAL="inh-signal-test"
  PROJECT="inh-dms-test"
  PORT_APP=8188
  PORT_SIGNAL=8180
fi
export INH_NAME_DMS="$NAME_DMS" INH_NAME_SIGNAL="$NAME_SIGNAL"
export INH_PORT_SIGNAL="$PORT_SIGNAL"
DATA="$DATA_ROOT/data"
CONFIG="$DATA/config.json"
# The image runs as 'nonroot' (uid 65532) but /data belongs to you and
# config.json is 0600, so the container must run under your uid to read it.
DOCKER_USER=(--user "$(id -u):$(id -g)")
export INH_UID="$(id -u)" INH_GID="$(id -g)"
COMPOSE=()
export INH_DATA_DIR="$DATA_ROOT"

# ---------------------------------------------------------------- preflight --
say "Preflight"
command -v docker >/dev/null || die "docker is not installed"
docker info >/dev/null 2>&1 || die "docker is not running, or you are not in the 'docker' group (log out and in after 'usermod -aG docker \$USER')"
ok "docker is running and reachable without sudo"

# Compose is convenient but not required: without it everything goes through
# plain docker commands (the same containers, just not managed by compose).
if [ "$NO_COMPOSE" = 1 ]; then
  COMPOSE=()
elif docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1 && docker-compose version 2>/dev/null | grep -qiE "version v?2\."; then
  COMPOSE=(docker-compose)          # standalone v2
else
  COMPOSE=()
  # compose v1 is end of life and cannot parse ${VAR:-default} in a compose
  # file, so plain docker commands beat quietly half-working support.
  if command -v docker-compose >/dev/null 2>&1; then
    warn "ignoring docker-compose v1 (end of life); using plain docker commands"
  fi
fi
if [ ${#COMPOSE[@]} -gt 0 ]; then ok "compose: ${COMPOSE[*]}"
else warn "no compose (or --no-compose); using plain docker commands"; fi

command -v openssl >/dev/null || die "openssl is missing (needed for hmac_secret)"
command -v curl    >/dev/null || die "curl is missing (needed for the post-start check)"

if (exec 3<>/dev/tcp/127.0.0.1/25) 2>/dev/null; then ok "postfix is listening on 127.0.0.1:25"
else warn "nothing is listening on 127.0.0.1:25; the switch will report a fault until that is fixed"; fi

for p in "$PORT_APP" $([ "$WITH_SIGNAL" = 1 ] && echo "$PORT_SIGNAL"); do
  if (exec 3<>/dev/tcp/127.0.0.1/"$p") 2>/dev/null; then
    if docker ps --format '{{.Names}}' | grep -qxE "$NAME_DMS|$NAME_SIGNAL"; then
      warn "port $p is held by this instance ($NAME_DMS/$NAME_SIGNAL); restarting it"
    else
      die "port $p is already taken (and not by this instance)"
    fi
  else ok "port $p is free"; fi
done

[ -f "$HERE/Dockerfile" ] || die "no Dockerfile next to the script"
ok "sources in place ($HERE)"

ok "instance: container $NAME_DMS, compose project $PROJECT, port $PORT_APP, data $DATA_ROOT"
if [ "$TEST_TIMINGS" = 1 ] && docker ps --format '{{.Names}}' | grep -qx "inh-dms"; then
  warn "a LIVE inh-dms is running alongside; the rehearsal will not touch it (different name, port and project)"
fi

if [ "$CHECK_ONLY" = 1 ]; then say "Preflight OK, nothing was changed."; exit 0; fi

# ----------------------------------------------------------------- directory --
say "Directory $DATA_ROOT"
if [ ! -d "$DATA" ]; then
  if mkdir -p "$DATA" 2>/dev/null; then :; else
    echo "   The directory has to be created as root. I will run:"
    echo "     sudo mkdir -p '$DATA' && sudo chown -R $USER: '$DATA_ROOT' && sudo chmod 700 '$DATA_ROOT'"
    read -r -p "   Continue? [y/N] " a; [ "$a" = "y" ] || die "aborted"
    sudo mkdir -p "$DATA" && sudo chown -R "$USER": "$DATA_ROOT" && sudo chmod 700 "$DATA_ROOT"
  fi
fi
chmod 700 "$DATA_ROOT" "$DATA" 2>/dev/null || true
ok "$DATA"

# ----------------------------------------------------------------- envelopes --
say "Envelopes (GPG ciphertext)"
if [ ${#ENV_IDS[@]} -gt 0 ]; then
  for i in "${!ENV_IDS[@]}"; do
    id="${ENV_IDS[$i]}"; src="${ENV_SRCS[$i]}"
    case "$id" in *[!a-zA-Z0-9_-]*) die "envelope id '$id' may only contain letters, digits, - and _";; esac
    [ -f "$src" ] || die "envelope '$src' does not exist"
    grep -q "BEGIN PGP MESSAGE" "$src" || die "'$src' does not look like an ASCII-armored PGP message"
    dst="$DATA/envelope-$id.asc"
    [ "$id" = "default" ] && dst="$DATA/envelope.asc"
    if [ "$src" -ef "$dst" ]; then
      ok "$id: already in place ($dst, $(wc -c <"$dst") B)"
    else
      install -m 600 "$src" "$dst"
      ok "$id → $dst ($(wc -c <"$dst") B)"
    fi
    ENV_DSTS+=("$dst")
  done
elif [ -f "$DATA/envelope.asc" ]; then
  ENV_IDS=("default"); ENV_DSTS=("$DATA/envelope.asc")
  ok "envelope.asc already in place (left alone)"
else
  die "no envelope; run with --envelope /path/envelope.asc (or --envelope id=path several times; make it offline, see the README)"
fi

# --------------------------------------------------------------------- config --
ask() { local q="$1" def="${2:-}" v=""; read -r -p "   $q${def:+ [$def]}: " v || true; printf '%s' "${v:-$def}"; }
json_safe() { case "$1" in *'"'*|*'\'*) die "the value '$1' contains \" or \\; put it into config.json by hand";; esac; }

say "Configuration"
if [ -f "$CONFIG" ] && [ "$FORCE_CONFIG" = 0 ]; then
  ok "config.json exists, leaving it alone (--force-config rewrites it)"
else
  [ -f "$CONFIG" ] && cp -a "$CONFIG" "$CONFIG.bak.$(date +%s)" && warn "old config backed up"
  echo "   (Enter keeps the default. Everything can be edited later in $CONFIG.)"
  DEF_BASE="https://dms.example.com"
  [ "$TEST_TIMINGS" = 1 ] && DEF_BASE="http://127.0.0.1:$PORT_APP"
  BASE_URL=$(ask "Public URL of the switch (https://...)" "$DEF_BASE")
  FROM=$(ask     "From address for e-mails" "dms@${BASE_URL#https://}")
  USER_MAIL=$(ask "Your e-mail (check-in, alerts)")
  for v in "$BASE_URL" "$FROM" "$USER_MAIL"; do json_safe "$v"; done
  [ -n "$USER_MAIL" ] || die "user_email is required"

  USER_SIG=""; SIGNAL_BLOCK=""
  if [ "$WITH_SIGNAL" = 1 ]; then
    SIG_FROM=$(ask "Signal number the switch sends from (E.164, +...)")
    USER_SIG=$(ask "Your Signal number (Enter to skip)")
    for v in "$SIG_FROM" "$USER_SIG"; do json_safe "$v"; done
    case "$SIG_FROM" in +*) ;; *) die "signal from_number must start with '+'";; esac
    SIGNAL_BLOCK=$(printf '  "signal": { "api_url": "http://127.0.0.1:%s", "from_number": "%s", "timeout": "20s" },' "$PORT_SIGNAL" "$SIG_FROM")
    [ -n "$USER_SIG" ] && SIGNAL_BLOCK="$SIGNAL_BLOCK\n$(printf '  "user_signal": "%s",' "$USER_SIG")"
  fi

  # --- recipients of each envelope ---
  ENVELOPES=""
  for i in "${!ENV_IDS[@]}"; do
    eid="${ENV_IDS[$i]}"; epath="${ENV_DSTS[$i]}"
    # The config is for the container, which has $DATA mounted as /data.
    cpath="/data/$(basename "$epath")"
    echo "   -- envelope: $eid ($cpath) --"
    ETO=""
    if [ "$TEST_TIMINGS" = 1 ]; then
      echo "      test: this envelope will go to you, $USER_MAIL"
      SIGJSON=""
      [ -n "$USER_SIG" ] && SIGJSON=$(printf ', "signal": "%s"' "$USER_SIG")
      ETO=$(printf '{ "name": "test", "email": "%s"%s }' "$USER_MAIL" "$SIGJSON")
    else
      RN=$(ask "  how many recipients for this envelope?" "1")
      for r in $(seq 1 "$RN"); do
        RNAME=$(ask "    $r. name")
        RMAIL=$(ask "    $r. e-mail")
        RSIG=""
        if [ "$WITH_SIGNAL" = 1 ]; then RSIG=$(ask "    $r. Signal number, Enter for none"); fi
        for v in "$RNAME" "$RMAIL" "$RSIG"; do json_safe "$v"; done
        if [ -z "$RMAIL" ] && [ -z "$RSIG" ]; then die "a recipient needs at least an e-mail or a Signal number"; fi
        SIGJSON=""
        [ -n "$RSIG" ] && SIGJSON=$(printf ', "signal": "%s"' "$RSIG")
        ONE=$(printf '{ "name": "%s", "email": "%s"%s }' "$RNAME" "$RMAIL" "$SIGJSON")
        [ -n "$ETO" ] && ETO="$ETO, "
        ETO="$ETO$ONE"
      done
    fi
    [ -z "$ETO" ] && die "envelope '$eid' has no recipients"
    [ -n "$ENVELOPES" ] && ENVELOPES="$ENVELOPES,\n"
    ONE=$(printf '    { "id": "%s", "path": "%s", "to": [ %s ] }' "$eid" "$cpath" "$ETO")
    ENVELOPES="$ENVELOPES$ONE"
  done
  N=$(ask "How many confirmers (people who can attest to the death)?" "2")
  CONFIRMERS=""
  for i in $(seq 1 "$N"); do
    echo "   -- confirmer $i --"
    CID=$(ask   "  id (no spaces, e.g. friend)")
    CNAME=$(ask "  name (used to address them in the e-mail)")
    CSIG=""
    if [ "$TEST_TIMINGS" = 1 ]; then
      # A rehearsal must never write to real people: in the waiting phase the
      # request repeats every reminder_interval, which here is a few minutes.
      CMAIL="$USER_MAIL"; CSIG="$USER_SIG"
      echo "     (test: this confirmer's requests go to you, $USER_MAIL)"
    else
      CMAIL=$(ask "  e-mail")
      [ "$WITH_SIGNAL" = 1 ] && CSIG=$(ask "  Signal number (Enter for none)")
    fi
    for v in "$CID" "$CNAME" "$CMAIL" "$CSIG"; do json_safe "$v"; done
    [ -n "$CID" ] && [ -n "$CMAIL" ] || die "a confirmer needs an id and an e-mail"
    [ -n "$CONFIRMERS" ] && CONFIRMERS+=",\n"
    CONFIRMERS+=$(printf '    { "id": "%s", "name": "%s", "email": "%s"%s }' \
      "$CID" "$CNAME" "$CMAIL" "$([ -n "$CSIG" ] && printf ', "signal": "%s"' "$CSIG")")
  done

  if [ "$TEST_TIMINGS" = 1 ]; then
    warn "TEST mode: minute-scale intervals; envelopes and confirmation requests all go to YOU ($USER_MAIL)"
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
    [ -n "$SIGNAL_BLOCK" ] && printf '%b\n' "$SIGNAL_BLOCK"
    printf '  "confirmers": [\n%b\n  ],\n' "$CONFIRMERS"
    printf '  "envelopes": [\n%b\n  ],\n' "$ENVELOPES"
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
  ok "wrote $CONFIG"
  warn "store hmac_secret in the KeePass database; without it the check-in links change"
fi

# ---------------------------------------------------------------- validation --
say "Validating the config"
if docker image inspect inh-dms >/dev/null 2>&1; then
  if OUT=$(docker run --rm "${DOCKER_USER[@]}" -v "$DATA":/data inh-dms --validate 2>&1); then ok "${OUT#*config OK: }"
  else die "the config did not validate: $OUT"; fi
else
  warn "the inh-dms image does not exist yet; the config is validated right after the build"
fi

if [ "$CONFIG_ONLY" = 1 ]; then say "Config done ($CONFIG). No container was started."; exit 0; fi

# A rehearsal must start from a known-good state: an old state.json, from a
# previous attempt say, would drop it straight into the waiting phase.
if [ "$TEST_TIMINGS" = 1 ] && [ -f "$DATA/state.json" ]; then
  rm -f "$DATA/state.json"
  warn "test: old state.json deleted; the rehearsal starts from scratch"
fi

# ------------------------------------------------------- kontajnery (2 cesty) --
# With compose, compose drives them; without it, plain docker commands do the same.
have_compose() { [ ${#COMPOSE[@]} -gt 0 ]; }

img_build() {
  if have_compose; then ( cd "$HERE" && "${COMPOSE[@]}" -p "$PROJECT" build dms )
  else docker build -t inh-dms "$HERE"; fi
}

svc_up() {
  # The signal-cli directory must be created by the deploying user. If Docker
  # made it, root would own it and `rm -rf` after a rehearsal would fail.
  [ "$WITH_SIGNAL" = 1 ] && mkdir -p "$DATA_ROOT/signal"
  if have_compose; then
    if [ "$WITH_SIGNAL" = 1 ]; then ( cd "$HERE" && "${COMPOSE[@]}" -p "$PROJECT" --profile signal up -d )
    else                            ( cd "$HERE" && "${COMPOSE[@]}" -p "$PROJECT" up -d dms ); fi
    return
  fi
  docker rm -f "$NAME_DMS" >/dev/null 2>&1 || true
  docker run -d --name "$NAME_DMS" --restart unless-stopped "${DOCKER_USER[@]}" \
    --network host -v "$DATA":/data inh-dms >/dev/null
  if [ "$WITH_SIGNAL" = 1 ]; then
    docker rm -f "$NAME_SIGNAL" >/dev/null 2>&1 || true
    docker run -d --name "$NAME_SIGNAL" --restart unless-stopped \
      -e MODE=native -e "AUTO_RECEIVE_SCHEDULE=0 4 * * *" \
      -p "127.0.0.1:$PORT_SIGNAL:8080" \
      -v "$DATA_ROOT/signal":/home/.local/share/signal-cli \
      bbernhard/signal-cli-rest-api:latest >/dev/null
  fi
}

svc_logs() {
  if have_compose; then ( cd "$HERE" && "${COMPOSE[@]}" -p "$PROJECT" logs --no-log-prefix dms 2>/dev/null || true )
  else docker logs "$NAME_DMS" 2>&1 || true; fi
}

if have_compose; then
  LOGS_CMD="${COMPOSE[*]} -p $PROJECT logs dms"; DOWN_CMD="${COMPOSE[*]} -p $PROJECT down"
else
  LOGS_CMD="docker logs $NAME_DMS"
  DOWN_CMD="docker rm -f $NAME_DMS"
  [ "$WITH_SIGNAL" = 1 ] && DOWN_CMD="docker rm -f $NAME_DMS $NAME_SIGNAL"
fi

# --------------------------------------------------------------------- start --
say "Build and start"
img_build
if OUT=$(docker run --rm "${DOCKER_USER[@]}" -v "$DATA":/data inh-dms --validate 2>&1); then ok "config: ${OUT#*config OK: }"
else die "the config did not validate: $OUT"; fi
svc_up
ok "containers are running"

# -------------------------------------------------------------------- kontrola --
say "Verification"
for i in $(seq 1 30); do
  BODY=$(curl -fsS --max-time 2 "http://127.0.0.1:$PORT_APP/" 2>/dev/null) && break || sleep 1
done
case "${BODY:-}" in *"inh DMS"*) ok "HTTP on 127.0.0.1:$PORT_APP answers";; *) die "the service does not answer; see: $LOGS_CMD";; esac

LOGS=$(svc_logs | tail -40)
case "$LOGS" in *"listening on"*) ok "startup line in the log";; *) warn "no 'listening' line in the log; check the logs";; esac
case "$LOGS" in *failed*) warn "the log contains 'failed'; see: $LOGS_CMD";; esac
if [ -f "$DATA/state.json" ]; then
  ok "state is being written ($DATA/state.json)"
  PHASE=$(sed -n 's/.*"phase"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$DATA/state.json" | head -1)
  case "$PHASE" in
    normal) ok "phase: normal (the switch considers you alive)" ;;
    "")     warn "could not read the phase from state.json" ;;
    *)      warn "phase: $PHASE. The switch is NOT at rest. Click the check-in link below to cancel it." ;;
  esac
else
  warn "no state.json yet; wait for the first tick"
fi

SECRET=$(sed -n 's/.*"hmac_secret": "\([^"]*\)".*/\1/p' "$CONFIG")
BASE_URL=$(sed -n 's/.*"public_base_url": "\([^"]*\)".*/\1/p' "$CONFIG")
TOKEN=$(printf 'checkin' | openssl dgst -sha256 -hmac "$SECRET" -r | cut -d' ' -f1)

say "What you still have to do"
cat <<TXT
1) A reverse proxy with TLS (needs root), pointing $BASE_URL at 127.0.0.1:$PORT_APP.

   Apache (a2enmod proxy proxy_http headers, then systemctl reload apache2):

     <VirtualHost *:443>
       ServerName ${BASE_URL#https://}
       ProxyPreserveHost On
       ProxyPass        / http://127.0.0.1:$PORT_APP/
       ProxyPassReverse / http://127.0.0.1:$PORT_APP/
       RequestHeader set X-Forwarded-Proto https
     </VirtualHost>

   or nginx (then systemctl reload nginx):

     server {
         listen 443 ssl;
         server_name ${BASE_URL#https://};
         location / {
             proxy_pass http://127.0.0.1:$PORT_APP;
             proxy_set_header Host              \$host;
             proxy_set_header X-Real-IP         \$remote_addr;
             proxy_set_header X-Forwarded-For   \$proxy_add_x_forwarded_for;
             proxy_set_header X-Forwarded-Proto https;
         }
     }

   (certbot supplies the TLS certificate either way)

2) Bookmark the check-in link and put it in the KeePass database. It is stable:

     $BASE_URL/checkin?token=$TOKEN

3) Check that the "[DMS] all good" e-mail arrived; it is sent right after startup.
   If it did not: $LOGS_CMD
TXT
[ "$WITH_SIGNAL" = 1 ] && cat <<TXT
4) Link Signal (once):
     xdg-open http://127.0.0.1:$PORT_SIGNAL/v1/qrcodelink?device_name=$NAME_DMS
     # scan it in Signal: Settings -> Linked devices -> +
     curl -s http://127.0.0.1:$PORT_SIGNAL/v1/accounts   # must list your from_number
TXT
[ "$TEST_TIMINGS" = 1 ] && cat <<TXT

NOTE: this is the TEST instance ($DATA_ROOT). Intervals are in minutes and the
envelopes come to you. When you are done: $DOWN_CMD && rm -rf $DATA_ROOT
TXT
say "Done."
