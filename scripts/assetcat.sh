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
FRONTEND_HOST="${FRONTEND_HOST:-0.0.0.0}"
FRONTEND_PORT="${FRONTEND_PORT:-6173}"
BACKEND_PID_FILE="${BACKEND_PID_FILE:-$RUN_DIR/$APP_NAME-backend.pid}"
FRONTEND_PID_FILE="${FRONTEND_PID_FILE:-$RUN_DIR/$APP_NAME-frontend.pid}"
BACKEND_LOG_FILE="${BACKEND_LOG_FILE:-$LOG_DIR/$APP_NAME-backend.log}"
FRONTEND_LOG_FILE="${FRONTEND_LOG_FILE:-$LOG_DIR/$APP_NAME-frontend.log}"
BIN_PATH="${BIN_PATH:-$BIN_DIR/$APP_NAME}"

usage() {
  cat <<EOF
Usage: $0 <command>

Commands:
  build      Build Vue frontend and Go backend binary
  start      Start backend and frontend in background
  stop       Stop backend and frontend
  restart    Stop then start backend and frontend
  status     Show backend and frontend process status
  logs       Follow backend and frontend logs

Environment:
  ADDR           Backend listen address, default :9080
  FRONTEND_HOST  Frontend listen host, default 0.0.0.0
  FRONTEND_PORT  Frontend listen port, default 6173
  DATA_PATH      JSON data file, default data/assets.json
  WEB_DIR        Built frontend dir, default web/dist
  BIN_PATH       Backend binary path, default bin/assetcat
EOF
}

ensure_dirs() {
  mkdir -p "$BIN_DIR" "$RUN_DIR" "$LOG_DIR" "$(dirname "$DATA_PATH")"
}

pid_running() {
  local pid_file="$1"
  [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null
}

cleanup_stale_pid() {
  local pid_file="$1"
  if [[ -f "$pid_file" ]] && ! kill -0 "$(cat "$pid_file")" 2>/dev/null; then
    rm -f "$pid_file"
  fi
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
  echo "Built frontend: $WEB_DIR"
  echo "Built backend: $BIN_PATH"
}

start_backend() {
  cleanup_stale_pid "$BACKEND_PID_FILE"
  if pid_running "$BACKEND_PID_FILE"; then
    echo "Backend is already running with PID $(cat "$BACKEND_PID_FILE")"
    return 0
  fi
  if [[ ! -x "$BIN_PATH" || ! -d "$WEB_DIR" ]]; then
    build
  fi

  nohup setsid "$BIN_PATH" -addr "$ADDR" -data "$DATA_PATH" -web "$WEB_DIR" >>"$BACKEND_LOG_FILE" 2>&1 &
  local pid=$!
  echo "$pid" >"$BACKEND_PID_FILE"
  sleep 1
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$BACKEND_PID_FILE"
    echo "Backend failed to start. See $BACKEND_LOG_FILE" >&2
    return 1
  fi
  echo "Backend started with PID $pid"
  echo "Backend: http://127.0.0.1:${ADDR#:}"
}

start_frontend() {
  cleanup_stale_pid "$FRONTEND_PID_FILE"
  if pid_running "$FRONTEND_PID_FILE"; then
    echo "Frontend is already running with PID $(cat "$FRONTEND_PID_FILE")"
    return 0
  fi
  if [[ ! -d "$ROOT_DIR/web/node_modules" ]]; then
    (cd "$ROOT_DIR/web" && npm install)
  fi

  (
    cd "$ROOT_DIR/web"
    nohup setsid npm run dev -- --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" >>"$FRONTEND_LOG_FILE" 2>&1 &
    echo $! >"$FRONTEND_PID_FILE"
  )

  local pid
  pid="$(cat "$FRONTEND_PID_FILE")"
  sleep 1
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$FRONTEND_PID_FILE"
    echo "Frontend failed to start. See $FRONTEND_LOG_FILE" >&2
    return 1
  fi
  echo "Frontend started with PID $pid"
  echo "Frontend: http://127.0.0.1:$FRONTEND_PORT"
}

start() {
  ensure_dirs
  start_backend
  if ! start_frontend; then
    stop_backend
    return 1
  fi
}

stop_pid() {
  local name="$1"
  local pid_file="$2"

  if ! [[ -f "$pid_file" ]]; then
    echo "$name is not running"
    return 0
  fi

  local pid
  pid="$(cat "$pid_file")"
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$pid_file"
    echo "Removed stale $name PID file"
    return 0
  fi

  kill "-$pid" 2>/dev/null || kill "$pid"
  for _ in {1..20}; do
    if ! kill -0 "$pid" 2>/dev/null; then
      rm -f "$pid_file"
      echo "$name stopped"
      return 0
    fi
    sleep 0.2
  done

  kill -9 "-$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
  rm -f "$pid_file"
  echo "$name stopped with SIGKILL"
}

stop_backend() {
  stop_pid "Backend" "$BACKEND_PID_FILE"
}

stop_frontend() {
  stop_pid "Frontend" "$FRONTEND_PID_FILE"
}

stop() {
  stop_frontend
  stop_backend
}

status_one() {
  local name="$1"
  local pid_file="$2"
  if pid_running "$pid_file"; then
    echo "$name is running with PID $(cat "$pid_file")"
  else
    cleanup_stale_pid "$pid_file"
    echo "$name is not running"
  fi
}

status() {
  status_one "Backend" "$BACKEND_PID_FILE"
  status_one "Frontend" "$FRONTEND_PID_FILE"
}

logs() {
  ensure_dirs
  touch "$BACKEND_LOG_FILE" "$FRONTEND_LOG_FILE"
  tail -f "$BACKEND_LOG_FILE" "$FRONTEND_LOG_FILE"
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
