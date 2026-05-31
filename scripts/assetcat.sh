#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="assetcat"
BIN_DIR="${BIN_DIR:-$ROOT_DIR/bin}"
RUN_DIR="${RUN_DIR:-$ROOT_DIR/run}"
LOG_DIR="${LOG_DIR:-$ROOT_DIR/logs}"
DATA_PATH="${DATA_PATH:-$ROOT_DIR/data/assets.json}"
WEB_DIR="${WEB_DIR:-$ROOT_DIR/web/dist}"
ADDR="${ADDR:-:9080}"
PID_FILE="${PID_FILE:-$RUN_DIR/$APP_NAME.pid}"
LOG_FILE="${LOG_FILE:-$LOG_DIR/$APP_NAME.log}"
BIN_PATH="${BIN_PATH:-$BIN_DIR/$APP_NAME}"

usage() {
  cat <<EOF
Usage: $0 <command>

Commands:
  build      Build Vue frontend and Go backend binary
  start      Start AssetCat in background
  stop       Stop AssetCat
  restart    Stop then start AssetCat
  status     Show process status
  logs       Follow service logs

Environment:
  ADDR       Backend listen address, default :9080
  DATA_PATH  JSON data file, default data/assets.json
  WEB_DIR    Built frontend dir, default web/dist
  BIN_PATH   Backend binary path, default bin/assetcat
EOF
}

ensure_dirs() {
  mkdir -p "$BIN_DIR" "$RUN_DIR" "$LOG_DIR" "$(dirname "$DATA_PATH")"
}

is_running() {
  [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

build_frontend() {
  if [[ ! -d "$ROOT_DIR/web/node_modules" ]]; then
    (cd "$ROOT_DIR/web" && npm install)
  fi
  (cd "$ROOT_DIR/web" && npm run build)
}

build_backend() {
  (cd "$ROOT_DIR" && go build -o "$BIN_PATH" ./cmd/asset-risk-server)
}

build() {
  ensure_dirs
  build_frontend
  build_backend
  echo "Built $BIN_PATH"
}

start() {
  ensure_dirs
  if is_running; then
    echo "AssetCat is already running with PID $(cat "$PID_FILE")"
    return 0
  fi
  if [[ ! -x "$BIN_PATH" || ! -d "$WEB_DIR" ]]; then
    build
  fi

  nohup "$BIN_PATH" -addr "$ADDR" -data "$DATA_PATH" -web "$WEB_DIR" >>"$LOG_FILE" 2>&1 &
  local pid=$!
  echo "$pid" >"$PID_FILE"
  sleep 1
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$PID_FILE"
    echo "AssetCat failed to start. See $LOG_FILE" >&2
    return 1
  fi
  echo "AssetCat started with PID $pid"
  echo "Backend and UI: http://127.0.0.1:${ADDR#:}"
}

stop() {
  if ! [[ -f "$PID_FILE" ]]; then
    echo "AssetCat is not running"
    return 0
  fi

  local pid
  pid="$(cat "$PID_FILE")"
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$PID_FILE"
    echo "Removed stale PID file"
    return 0
  fi

  kill "$pid"
  for _ in {1..20}; do
    if ! kill -0 "$pid" 2>/dev/null; then
      rm -f "$PID_FILE"
      echo "AssetCat stopped"
      return 0
    fi
    sleep 0.2
  done

  kill -9 "$pid" 2>/dev/null || true
  rm -f "$PID_FILE"
  echo "AssetCat stopped with SIGKILL"
}

status() {
  if is_running; then
    echo "AssetCat is running with PID $(cat "$PID_FILE")"
  else
    echo "AssetCat is not running"
  fi
}

logs() {
  ensure_dirs
  touch "$LOG_FILE"
  tail -f "$LOG_FILE"
}

case "${1:-}" in
  build)
    build
    ;;
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    stop
    start
    ;;
  status)
    status
    ;;
  logs)
    logs
    ;;
  ""|-h|--help|help)
    usage
    ;;
  *)
    echo "Unknown command: $1" >&2
    usage >&2
    exit 2
    ;;
esac
