#!/usr/bin/env bash
# Shared helpers for start.sh and stop.sh

pid_is_kanba() {
  local pid=$1
  kill -0 "$pid" 2>/dev/null || return 1

  local cmd args bin_name
  bin_name=$(basename "${BINARY:-kanba}")
  cmd=$(ps -p "$pid" -o comm= 2>/dev/null | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  if [[ "$(basename "$cmd")" == "$bin_name" ]]; then
    return 0
  fi

  args=$(ps -p "$pid" -o args= 2>/dev/null | sed 's/^[[:space:]]*//') || return 1
  [[ "$args" == "${bin_name}"* ]] || [[ "$args" == *"/${bin_name}"* ]] || [[ "$args" == "./"*"/${bin_name}"* ]]
}

listener_pids_on_port() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN -t 2>/dev/null || true
}

stop_kanba_on_port() {
  local port="$1"
  local stopped=0
  local pid

  while IFS= read -r pid; do
    [[ -z "$pid" ]] && continue
    if pid_is_kanba "$pid"; then
      kill "$pid" 2>/dev/null || true
      echo "Stopped local Kanba server (pid ${pid}) on port ${port}."
      stopped=1
    fi
  done < <(listener_pids_on_port "$port")

  if [[ "$stopped" -eq 1 ]]; then
    sleep 1
    return 0
  fi
  return 1
}

ensure_port_for_kanba() {
  local port="$1"
  local pid cmd

  pid=$(listener_pids_on_port "$port" | head -1)
  if [[ -z "$pid" ]]; then
    return 0
  fi

  if pid_is_kanba "$pid"; then
    echo "Port ${port} in use by Kanba (pid ${pid}); stopping it..."
    stop_kanba_on_port "$port"
    if [[ -n "${PID_FILE:-}" && -f "$PID_FILE" ]]; then
      rm -f "$PID_FILE"
    fi
    return 0
  fi

  cmd=$(ps -p "$pid" -o comm= 2>/dev/null | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  echo "Port ${port} is in use by ${cmd} (pid ${pid}). Stop it or set PORT to another value." >&2
  exit 1
}
