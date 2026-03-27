#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$HOME/.openclaw-instances"
DEFAULT_BASE_PORT=18789

usage() {
  cat <<'EOF'
Usage: openclaw.sh <command> <name> [options]

Commands:
  create <name> [-p PORT]   Create a new OpenClaw container (auto-restart, detached)
  onboard <name>            Enter container interactively to run openclaw setup/onboard
  start <name>              Start gateway in background
  stop <name>               Stop the container
  restart <name>            Restart gateway
  logs <name>               Tail container logs
  shell <name>              Open a shell in the container
  remove <name>             Remove container and optionally its data
  dashboard <name>          Get tokenized dashboard URL
  list                      List all OpenClaw instances
  status                    Show running status of all instances

Options:
  -p PORT   Gateway port (default: auto-assigned starting from 18789)

Examples:
  openclaw.sh create alice              # creates openclaw-alice on auto port
  openclaw.sh create bob -p 18800      # creates openclaw-bob on port 18800
  openclaw.sh onboard alice             # interactive setup inside container
  openclaw.sh start alice               # start gateway in background
  openclaw.sh list                      # show all instances
EOF
  exit 1
}

get_container_name() {
  echo "openclaw-${1}"
}

get_instance_dir() {
  echo "${BASE_DIR}/${1}"
}

# Find next available port by scanning existing instances
next_available_port() {
  local port=$DEFAULT_BASE_PORT
  local used_ports
  used_ports=$(docker ps -a --filter "name=openclaw-" --format '{{.Ports}}' 2>/dev/null \
    | grep -oE '0\.0\.0\.0:[0-9]+' \
    | cut -d: -f2 \
    | sort -n \
    | uniq)

  while echo "$used_ports" | grep -qw "$port"; do
    port=$((port + 2))
  done
  echo "$port"
}

cmd_create() {
  local name="$1"; shift
  local port=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -p) port="$2"; shift 2 ;;
      *) echo "Unknown option: $1"; usage ;;
    esac
  done

  if [[ -z "$port" ]]; then
    port=$(next_available_port)
  fi

  local container
  container=$(get_container_name "$name")
  local instance_dir
  instance_dir=$(get_instance_dir "$name")
  local control_port=$((port + 1))

  # Check if container already exists
  if docker ps -a --format '{{.Names}}' | grep -qw "$container"; then
    echo "Error: Container '$container' already exists. Use 'remove' first or pick a different name."
    exit 1
  fi

  echo "Creating OpenClaw instance '$name'..."
  echo "  Container:    $container"
  echo "  Gateway port: $port"
  echo "  Control port: $control_port"
  echo "  Config dir:   $instance_dir/config"
  echo "  Workspace:    $instance_dir/workspace"

  mkdir -p "$instance_dir/config"
  mkdir -p "$instance_dir/workspace"

  docker run -dit \
    --name "$container" \
    --restart unless-stopped \
    -p "${port}:${port}" \
    -p "${control_port}:${control_port}" \
    -v "$instance_dir/config:/root/.openclaw" \
    -v "$instance_dir/workspace:/root/.openclaw/workspace" \
    -e "OPENCLAW_PORT=${port}" \
    node:24-bookworm bash

  # Save instance metadata
  cat > "$instance_dir/instance.env" <<ENVEOF
NAME=$name
CONTAINER=$container
PORT=$port
CONTROL_PORT=$control_port
CREATED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ENVEOF

  echo ""
  echo "Container created. Next steps:"
  echo "  1. openclaw.sh onboard $name    # interactive setup (run once)"
  echo "  2. openclaw.sh start $name      # start gateway"
  echo "  3. Open http://127.0.0.1:$port"
}

cmd_onboard() {
  local name="$1"
  local container
  container=$(get_container_name "$name")
  local instance_dir
  instance_dir=$(get_instance_dir "$name")

  if [[ ! -f "$instance_dir/instance.env" ]]; then
    echo "Error: Instance '$name' not found. Run 'create' first."
    exit 1
  fi

  source "$instance_dir/instance.env"

  echo "Entering container '$container' for interactive onboard..."
  echo "Run these commands inside:"
  echo "  npm install -g openclaw@latest"
  echo "  openclaw setup"
  echo "  openclaw onboard"
  echo "  openclaw config set gateway.port ${PORT}"
  echo "  openclaw config set gateway.controlUi.allowedOrigins '[\"http://127.0.0.1:${PORT}\",\"http://localhost:${PORT}\"]' --strict-json"
  echo "  openclaw config set gateway.controlUi.dangerouslyDisableDeviceAuth true"
  echo "  exit"
  echo ""

  docker start "$container" >/dev/null 2>&1 || true
  docker exec -it "$container" bash
}

cmd_start() {
  local name="$1"
  local container
  container=$(get_container_name "$name")
  local instance_dir
  instance_dir=$(get_instance_dir "$name")

  if [[ ! -f "$instance_dir/instance.env" ]]; then
    echo "Error: Instance '$name' not found."
    exit 1
  fi

  source "$instance_dir/instance.env"

  # Write startup script that installs openclaw and runs gateway as PID 1
  cat > "$instance_dir/config/start-gateway.sh" <<STARTEOF
#!/bin/bash
npm install -g openclaw@latest >/dev/null 2>&1
exec openclaw gateway --bind lan --port ${PORT}
STARTEOF
  chmod +x "$instance_dir/config/start-gateway.sh"

  # Recreate container with gateway as main process so it auto-restarts with Docker
  docker rm -f "$container" >/dev/null 2>&1 || true

  docker run -d \
    --name "$container" \
    --restart unless-stopped \
    -p "${PORT}:${PORT}" \
    -p "${CONTROL_PORT}:${CONTROL_PORT}" \
    -v "$instance_dir/config:/root/.openclaw" \
    -v "$instance_dir/workspace:/root/.openclaw/workspace" \
    -e "OPENCLAW_PORT=${PORT}" \
    node:24-bookworm \
    bash /root/.openclaw/start-gateway.sh

  # Wait for gateway to be ready before fetching token
  echo "Waiting for gateway to start..."
  local retries=0
  while ! docker exec "$container" which openclaw >/dev/null 2>&1 && [[ $retries -lt 30 ]]; do
    sleep 1
    retries=$((retries + 1))
  done
  sleep 2

  local dashboard_url
  dashboard_url=$(docker exec "$container" openclaw dashboard --no-open 2>/dev/null | grep -oE 'http://[^ ]+')

  # Replace default port with actual port
  dashboard_url=$(echo "$dashboard_url" | sed "s|://127.0.0.1:[0-9]*|://127.0.0.1:${PORT}|")

  echo "Started '$name' gateway on port $PORT"
  if [[ -n "$dashboard_url" ]]; then
    echo "  UI: $dashboard_url"
  else
    echo "  UI: http://127.0.0.1:$PORT"
    echo "  (token not ready yet — run: openclaw.sh dashboard $name)"
  fi
}

cmd_dashboard() {
  local name="$1"
  local container
  container=$(get_container_name "$name")
  local instance_dir
  instance_dir=$(get_instance_dir "$name")

  if [[ ! -f "$instance_dir/instance.env" ]]; then
    echo "Error: Instance '$name' not found."
    exit 1
  fi

  source "$instance_dir/instance.env"

  local dashboard_url
  dashboard_url=$(docker exec "$container" openclaw dashboard --no-open 2>/dev/null | grep -oE 'http://[^ ]+')
  dashboard_url=$(echo "$dashboard_url" | sed "s|://127.0.0.1:[0-9]*|://127.0.0.1:${PORT}|")

  if [[ -n "$dashboard_url" ]]; then
    echo "$dashboard_url"
  else
    echo "Error: Could not get dashboard URL. Is the gateway running?"
    exit 1
  fi
}

cmd_stop() {
  local name="$1"
  local container
  container=$(get_container_name "$name")

  docker stop "$container"
  echo "Stopped '$name'"
}

cmd_restart() {
  local name="$1"
  cmd_stop "$name" 2>/dev/null || true
  cmd_start "$name"
}

cmd_logs() {
  local name="$1"
  local container
  container=$(get_container_name "$name")

  docker logs -f "$container"
}

cmd_shell() {
  local name="$1"
  local container
  container=$(get_container_name "$name")

  docker start "$container" >/dev/null 2>&1 || true
  docker exec -it "$container" bash
}

cmd_remove() {
  local name="$1"
  local container
  container=$(get_container_name "$name")
  local instance_dir
  instance_dir=$(get_instance_dir "$name")

  echo "Removing container '$container'..."
  docker rm -f "$container" >/dev/null 2>&1 || true

  if [[ -d "$instance_dir" ]]; then
    read -rp "Also remove data at $instance_dir? [y/N] " confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
      rm -rf "$instance_dir"
      echo "Data removed."
    else
      echo "Data kept at $instance_dir"
    fi
  fi

  echo "Instance '$name' removed."
}

cmd_list() {
  if [[ ! -d "$BASE_DIR" ]] || [[ -z "$(ls -A "$BASE_DIR" 2>/dev/null)" ]]; then
    echo "No OpenClaw instances found."
    return
  fi

  printf "%-15s %-20s %-8s %-8s %s\n" "NAME" "CONTAINER" "PORT" "CTRL" "CREATED"
  printf "%-15s %-20s %-8s %-8s %s\n" "----" "---------" "----" "----" "-------"

  for dir in "$BASE_DIR"/*/; do
    [[ -f "$dir/instance.env" ]] || continue
    (
      source "$dir/instance.env"
      printf "%-15s %-20s %-8s %-8s %s\n" "$NAME" "$CONTAINER" "$PORT" "$CONTROL_PORT" "$CREATED"
    )
  done
}

cmd_status() {
  if [[ ! -d "$BASE_DIR" ]] || [[ -z "$(ls -A "$BASE_DIR" 2>/dev/null)" ]]; then
    echo "No OpenClaw instances found."
    return
  fi

  printf "%-15s %-20s %-8s %-12s %s\n" "NAME" "CONTAINER" "PORT" "STATUS" "URL"
  printf "%-15s %-20s %-8s %-12s %s\n" "----" "---------" "----" "------" "---"

  for dir in "$BASE_DIR"/*/; do
    [[ -f "$dir/instance.env" ]] || continue
    (
      source "$dir/instance.env"
      local status
      status=$(docker inspect -f '{{.State.Status}}' "$CONTAINER" 2>/dev/null || echo "removed")
      local url="http://127.0.0.1:$PORT"
      printf "%-15s %-20s %-8s %-12s %s\n" "$NAME" "$CONTAINER" "$PORT" "$status" "$url"
    )
  done
}

# --- Main ---

[[ $# -lt 1 ]] && usage

command="$1"; shift

case "$command" in
  create)
    [[ $# -lt 1 ]] && { echo "Error: 'create' requires a name."; usage; }
    cmd_create "$@"
    ;;
  onboard)
    [[ $# -lt 1 ]] && { echo "Error: 'onboard' requires a name."; usage; }
    cmd_onboard "$1"
    ;;
  start)
    [[ $# -lt 1 ]] && { echo "Error: 'start' requires a name."; usage; }
    cmd_start "$1"
    ;;
  stop)
    [[ $# -lt 1 ]] && { echo "Error: 'stop' requires a name."; usage; }
    cmd_stop "$1"
    ;;
  restart)
    [[ $# -lt 1 ]] && { echo "Error: 'restart' requires a name."; usage; }
    cmd_restart "$1"
    ;;
  logs)
    [[ $# -lt 1 ]] && { echo "Error: 'logs' requires a name."; usage; }
    cmd_logs "$1"
    ;;
  shell)
    [[ $# -lt 1 ]] && { echo "Error: 'shell' requires a name."; usage; }
    cmd_shell "$1"
    ;;
  remove)
    [[ $# -lt 1 ]] && { echo "Error: 'remove' requires a name."; usage; }
    cmd_remove "$1"
    ;;
  list)
    cmd_list
    ;;
  status)
    cmd_status
    ;;
  dashboard)
    [[ $# -lt 1 ]] && { echo "Error: 'dashboard' requires a name."; usage; }
    cmd_dashboard "$1"
    ;;
  *)
    echo "Unknown command: $command"
    usage
    ;;
esac
