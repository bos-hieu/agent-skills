#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# ssh-manager.sh — SSH operations using a Go config helper for host resolution
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_GO="${SCRIPT_DIR}/config.go"

# ===========================================================================
# Usage
# ===========================================================================

usage() {
  cat <<'HELP'
Usage: ssh-manager.sh <command> [arguments...]

Commands:
  config  <args...>        Pass-through to the Go config helper
  list                     List all configured hosts
  status                   Check connectivity for every configured host

  shell   <host>           Open an interactive SSH session
  exec    <host> <cmd...>  Run a command on a remote host
  exec-tag <tag> <cmd...>  Run a command on all hosts with a given tag

  tunnel  <host> <mapping> Create an SSH tunnel (named or ad-hoc L:R:P)
  scp     <host> <src> <dst>  Copy files via SCP (prefix path with : for remote)
  rsync   <host> <src> <dst>  Sync files via rsync (prefix path with : for remote)

  check     <host> [port]  Test SSH connectivity (optionally check a port)
  check-tag <tag>          Test connectivity for all hosts with a given tag

  tail    <host> <file> [lines]  Tail a remote file (default 100 lines)
  info    <host>           Show system info for a host
  info-tag <tag>           Show system info for all hosts with a given tag

Examples:
  ssh-manager.sh shell web1
  ssh-manager.sh exec  web1 uptime
  ssh-manager.sh exec-tag production "df -h /"
  ssh-manager.sh tunnel db1 local_pg            # named tunnel from config
  ssh-manager.sh tunnel db1 5432:localhost:5432  # ad-hoc tunnel
  ssh-manager.sh scp web1 ./file.txt :/tmp/     # local -> remote
  ssh-manager.sh scp web1 :/var/log/app.log .   # remote -> local
  ssh-manager.sh rsync web1 ./src/ :/opt/app/src/
  ssh-manager.sh check web1
  ssh-manager.sh check web1 8080
  ssh-manager.sh check-tag production
  ssh-manager.sh tail web1 /var/log/syslog 50
  ssh-manager.sh info web1
  ssh-manager.sh info-tag staging
  ssh-manager.sh status
  ssh-manager.sh list
  ssh-manager.sh config --add myhost --host 10.0.0.1 --port 22 --user deploy
HELP
}

# ===========================================================================
# Host resolution helpers
# ===========================================================================

# get_host_info <name>
#   Calls the Go config helper and populates H_NAME, H_HOST, H_PORT, H_USER,
#   H_BASTION, plus any TUNNEL_* variables found in the output.
get_host_info() {
  local name="$1"
  local output
  output="$(go run "$CONFIG_GO" --get "$name" 2>&1)" || {
    echo "Error: could not resolve host '$name'" >&2
    echo "$output" >&2
    return 1
  }

  # Reset variables
  H_NAME="" H_HOST="" H_PORT="" H_USER="" H_BASTION=""

  # Parse KEY=VALUE lines
  while IFS= read -r line; do
    case "$line" in
      NAME=*)    H_NAME="${line#NAME=}" ;;
      HOST=*)    H_HOST="${line#HOST=}" ;;
      PORT=*)    H_PORT="${line#PORT=}" ;;
      USER=*)    H_USER="${line#USER=}" ;;
      BASTION=*) H_BASTION="${line#BASTION=}" ;;
      TUNNEL_*)
        # Export tunnel variables dynamically (e.g. TUNNEL_PG=5432:localhost:5432)
        local tkey tval
        tkey="${line%%=*}"
        tval="${line#*=}"
        eval "${tkey}=\"${tval}\""
        ;;
    esac
  done <<< "$output"

  # Defaults
  H_PORT="${H_PORT:-22}"
}

# build_jump_chain <name>
#   Recursively resolves bastion references and outputs a ProxyJump chain.
#   Detects circular references by tracking visited bastion names.
build_jump_chain() {
  local name="$1"
  shift
  # Remaining args are visited bastions (for recursion)
  local visited=("$@")

  get_host_info "$name"

  if [[ -z "$H_BASTION" ]]; then
    echo ""
    return
  fi

  # Circular reference detection
  for v in "${visited[@]+"${visited[@]}"}"; do
    if [[ "$v" == "$H_BASTION" ]]; then
      echo "Error: circular bastion reference detected: $H_BASTION" >&2
      return 1
    fi
  done

  visited+=("$H_BASTION")

  # Resolve the bastion itself (it may chain further)
  local parent_chain
  parent_chain="$(build_jump_chain "$H_BASTION" "${visited[@]}")" || return 1

  # Build this bastion's jump spec
  local bastion_spec
  get_host_info "$H_BASTION"
  if [[ -n "$H_USER" ]]; then
    bastion_spec="${H_USER}@${H_HOST}:${H_PORT}"
  else
    bastion_spec="${H_HOST}:${H_PORT}"
  fi

  if [[ -n "$parent_chain" ]]; then
    echo "${parent_chain},${bastion_spec}"
  else
    echo "$bastion_spec"
  fi

  # Restore the original host info so the caller still has it
  get_host_info "$name" >/dev/null 2>&1 || true
}

# build_ssh_cmd <name>
#   Prints the base SSH command (without trailing command) for the given host.
build_ssh_cmd() {
  local name="$1"

  # Resolve host info first, then build jump chain (which may overwrite H_* vars)
  get_host_info "$name"
  local host="$H_HOST" port="$H_PORT" user="$H_USER"

  local jump_chain
  jump_chain="$(build_jump_chain "$name")"

  local -a cmd=(ssh)

  if [[ -n "$jump_chain" ]]; then
    cmd+=(-J "$jump_chain")
  fi

  cmd+=(-p "$port")

  if [[ -n "$user" ]]; then
    cmd+=("${user}@${host}")
  else
    cmd+=("$host")
  fi

  echo "${cmd[@]}"
}

# build_scp_cmd <name>
#   Prints the base SCP command (without src/dest) for the given host.
build_scp_cmd() {
  local name="$1"

  get_host_info "$name"
  local port="$H_PORT"

  local jump_chain
  jump_chain="$(build_jump_chain "$name")"

  # Restore host info after build_jump_chain
  get_host_info "$name"

  local -a cmd=(scp)

  if [[ -n "$jump_chain" ]]; then
    cmd+=(-J "$jump_chain")
  fi

  cmd+=(-P "$port")

  echo "${cmd[@]}"
}

# get_remote_prefix <name>
#   Returns USER@HOST or just HOST.
get_remote_prefix() {
  local name="$1"
  get_host_info "$name"

  if [[ -n "$H_USER" ]]; then
    echo "${H_USER}@${H_HOST}"
  else
    echo "$H_HOST"
  fi
}

# ===========================================================================
# Command implementations
# ===========================================================================

# cmd_exec <host> <command...>
cmd_exec() {
  local host="$1"; shift
  local ssh_cmd
  ssh_cmd="$(build_ssh_cmd "$host")"

  # shellcheck disable=SC2086
  $ssh_cmd "$@"
}

# cmd_exec_tag <tag> <command...>
cmd_exec_tag() {
  local tag="$1"; shift
  local cmd_args=("$@")

  local output
  output="$(go run "$CONFIG_GO" --get-by-tag "$tag" 2>&1)" || {
    echo "Error: could not resolve hosts for tag '$tag'" >&2
    echo "$output" >&2
    return 1
  }

  if [[ -z "$output" ]]; then
    echo "No hosts found with tag '$tag'" >&2
    return 1
  fi

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    local hname
    hname="$(echo "$line" | sed 's/NAME=\([^ ]*\).*/\1/')"

    if [[ -z "$hname" ]]; then
      continue
    fi

    local ssh_cmd
    ssh_cmd="$(build_ssh_cmd "$hname")"

    # Run and prefix every output line with [hostname]
    # shellcheck disable=SC2086
    $ssh_cmd "${cmd_args[@]}" 2>&1 | while IFS= read -r out_line; do
      echo "[$hname] $out_line"
    done
  done <<< "$output"
}

# cmd_shell <host>
cmd_shell() {
  local host="$1"
  local ssh_cmd
  ssh_cmd="$(build_ssh_cmd "$host")"

  # Replace current process with SSH for interactive session
  # shellcheck disable=SC2086
  exec $ssh_cmd
}

# cmd_tunnel <host> <name_or_mapping>
cmd_tunnel() {
  local host="$1"
  local mapping_arg="$2"
  local mapping=""

  if echo "$mapping_arg" | grep -q ':'; then
    # Ad-hoc mapping: L:R:P format
    mapping="$mapping_arg"
  else
    # Named tunnel — look it up from the host config
    local tunnel_var="TUNNEL_${mapping_arg}"
    get_host_info "$host"

    # Check if the tunnel variable was set during get_host_info
    mapping="${!tunnel_var:-}"

    if [[ -z "$mapping" ]]; then
      # Try uppercase version
      local tunnel_var_upper
      tunnel_var_upper="TUNNEL_$(echo "$mapping_arg" | tr '[:lower:]' '[:upper:]')"
      mapping="${!tunnel_var_upper:-}"
    fi

    if [[ -z "$mapping" ]]; then
      echo "Error: no tunnel named '$mapping_arg' found for host '$host'" >&2
      return 1
    fi
  fi

  # Resolve host info and jump chain
  get_host_info "$host"
  local h_host="$H_HOST" h_port="$H_PORT" h_user="$H_USER"

  local jump_chain
  jump_chain="$(build_jump_chain "$host")"

  # Restore host info
  get_host_info "$host"

  local -a cmd=(ssh)

  if [[ -n "$jump_chain" ]]; then
    cmd+=(-J "$jump_chain")
  fi

  cmd+=(-p "$h_port" -L "$mapping" -N -f)

  if [[ -n "$h_user" ]]; then
    cmd+=("${h_user}@${h_host}")
  else
    cmd+=("$h_host")
  fi

  "${cmd[@]}"

  # Extract local port from the mapping (first part before the first colon)
  local local_port
  local_port="$(echo "$mapping" | awk -F: '{print $1}')"

  echo "Tunnel established: localhost:${local_port} -> ${mapping}"
  echo "SSH target: ${h_user:+${h_user}@}${h_host}:${h_port}"
  if [[ -n "$jump_chain" ]]; then
    echo "Jump chain: ${jump_chain}"
  fi
}

# cmd_scp <host> <src> <dest>
cmd_scp() {
  local host="$1"
  local src="$2"
  local dest="$3"

  local scp_cmd
  scp_cmd="$(build_scp_cmd "$host")"

  local remote_prefix
  remote_prefix="$(get_remote_prefix "$host")"

  # If src starts with :, it is a remote path
  if [[ "$src" == :* ]]; then
    src="${remote_prefix}:${src#:}"
  fi

  # If dest starts with :, it is a remote path
  if [[ "$dest" == :* ]]; then
    dest="${remote_prefix}:${dest#:}"
  fi

  # shellcheck disable=SC2086
  $scp_cmd "$src" "$dest"
}

# cmd_rsync <host> <src> <dest>
cmd_rsync() {
  local host="$1"
  local src="$2"
  local dest="$3"

  get_host_info "$host"
  local h_host="$H_HOST" h_port="$H_PORT" h_user="$H_USER"

  local jump_chain
  jump_chain="$(build_jump_chain "$host")"

  # Restore host info
  get_host_info "$host"

  local remote_prefix
  remote_prefix="$(get_remote_prefix "$host")"

  # Build the SSH command for rsync's -e flag
  local ssh_rsh="ssh"
  if [[ -n "$jump_chain" ]]; then
    ssh_rsh="ssh -J ${jump_chain}"
  fi
  ssh_rsh="${ssh_rsh} -p ${h_port}"

  # If src starts with :, it is a remote path
  if [[ "$src" == :* ]]; then
    src="${remote_prefix}:${src#:}"
  fi

  # If dest starts with :, it is a remote path
  if [[ "$dest" == :* ]]; then
    dest="${remote_prefix}:${dest#:}"
  fi

  rsync -avz -e "$ssh_rsh" "$src" "$dest"
}

# cmd_check <host> [port]
cmd_check() {
  local host="$1"
  local port="${2:-}"

  local ssh_cmd
  ssh_cmd="$(build_ssh_cmd "$host")"

  # Test basic SSH connectivity
  # shellcheck disable=SC2086
  if $ssh_cmd -o ConnectTimeout=5 -o BatchMode=yes "echo ok" >/dev/null 2>&1; then
    echo "[$host] SSH: OK"
  else
    echo "[$host] SSH: FAILED"
    return 1
  fi

  # If a port was provided, check it via SSH
  if [[ -n "$port" ]]; then
    # shellcheck disable=SC2086
    if $ssh_cmd "nc -z localhost $port" >/dev/null 2>&1; then
      echo "[$host] Port $port: OPEN"
    else
      echo "[$host] Port $port: CLOSED"
    fi
  fi
}

# cmd_check_tag <tag>
cmd_check_tag() {
  local tag="$1"

  local output
  output="$(go run "$CONFIG_GO" --get-by-tag "$tag" 2>&1)" || {
    echo "Error: could not resolve hosts for tag '$tag'" >&2
    echo "$output" >&2
    return 1
  }

  if [[ -z "$output" ]]; then
    echo "No hosts found with tag '$tag'" >&2
    return 1
  fi

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    local hname
    hname="$(echo "$line" | sed 's/NAME=\([^ ]*\).*/\1/')"

    if [[ -z "$hname" ]]; then
      continue
    fi

    cmd_check "$hname" || true
  done <<< "$output"
}

# cmd_status
cmd_status() {
  local output
  output="$(go run "$CONFIG_GO" --list 2>&1)" || {
    echo "Error: could not list hosts" >&2
    echo "$output" >&2
    return 1
  }

  # Skip the first 2 header lines, extract the first column (host name)
  local line_num=0
  while IFS= read -r line; do
    line_num=$((line_num + 1))

    # Skip header lines (first 2)
    if [[ $line_num -le 2 ]]; then
      continue
    fi

    [[ -z "$line" ]] && continue

    local hname
    hname="$(echo "$line" | awk '{print $1}')"

    if [[ -z "$hname" ]]; then
      continue
    fi

    cmd_check "$hname" || true
  done <<< "$output"
}

# cmd_tail <host> <file> [lines]
cmd_tail() {
  local host="$1"
  local file="$2"
  local lines="${3:-100}"

  local ssh_cmd
  ssh_cmd="$(build_ssh_cmd "$host")"

  # shellcheck disable=SC2086
  $ssh_cmd "tail -n $lines -f $file"
}

# cmd_info <host>
cmd_info() {
  local host="$1"

  local ssh_cmd
  ssh_cmd="$(build_ssh_cmd "$host")"

  echo "=== [$host] System Info ==="

  # shellcheck disable=SC2086
  $ssh_cmd bash -s <<'REMOTE'
echo "--- Uptime ---"
uptime
echo ""
echo "--- CPU Cores ---"
nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "unknown"
echo ""
echo "--- Memory ---"
free -h 2>/dev/null || vm_stat 2>/dev/null || echo "unknown"
echo ""
echo "--- Disk (/) ---"
df -h /
echo ""
echo "--- OS Release ---"
cat /etc/os-release 2>/dev/null || sw_vers 2>/dev/null || echo "unknown"
REMOTE

  echo ""
}

# cmd_info_tag <tag>
cmd_info_tag() {
  local tag="$1"

  local output
  output="$(go run "$CONFIG_GO" --get-by-tag "$tag" 2>&1)" || {
    echo "Error: could not resolve hosts for tag '$tag'" >&2
    echo "$output" >&2
    return 1
  }

  if [[ -z "$output" ]]; then
    echo "No hosts found with tag '$tag'" >&2
    return 1
  fi

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    local hname
    hname="$(echo "$line" | sed 's/NAME=\([^ ]*\).*/\1/')"

    if [[ -z "$hname" ]]; then
      continue
    fi

    cmd_info "$hname"
  done <<< "$output"
}

# ===========================================================================
# Argument validation helper
# ===========================================================================

validate_args() {
  local required="$1"
  local actual="$2"
  local cmd_name="$3"

  if [[ "$actual" -lt "$required" ]]; then
    echo "Error: '$cmd_name' requires at least $required argument(s), got $actual" >&2
    echo "" >&2
    usage >&2
    exit 1
  fi
}

# ===========================================================================
# Main dispatcher
# ===========================================================================

if [[ $# -eq 0 ]]; then
  usage
  exit 0
fi

COMMAND="$1"
shift

case "$COMMAND" in
  config)
    go run "$CONFIG_GO" "$@"
    ;;
  list)
    go run "$CONFIG_GO" --list
    ;;
  status)
    cmd_status
    ;;
  exec)
    validate_args 2 $# "exec"
    cmd_exec "$@"
    ;;
  exec-tag)
    validate_args 2 $# "exec-tag"
    cmd_exec_tag "$@"
    ;;
  tunnel)
    validate_args 2 $# "tunnel"
    cmd_tunnel "$@"
    ;;
  scp)
    validate_args 3 $# "scp"
    cmd_scp "$@"
    ;;
  rsync)
    validate_args 3 $# "rsync"
    cmd_rsync "$@"
    ;;
  shell)
    validate_args 1 $# "shell"
    cmd_shell "$@"
    ;;
  check)
    validate_args 1 $# "check"
    cmd_check "$@"
    ;;
  check-tag)
    validate_args 1 $# "check-tag"
    cmd_check_tag "$@"
    ;;
  tail)
    validate_args 2 $# "tail"
    cmd_tail "$@"
    ;;
  info)
    validate_args 1 $# "info"
    cmd_info "$@"
    ;;
  info-tag)
    validate_args 1 $# "info-tag"
    cmd_info_tag "$@"
    ;;
  *)
    echo "Error: unknown command '$COMMAND'" >&2
    echo "" >&2
    usage >&2
    exit 1
    ;;
esac
