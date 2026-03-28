---
name: ssh
description: Manage SSH connections to servers with bastion/jump host support, tunneling, file transfer, multi-host execution, and system monitoring. Triggers on "ssh", "remote server", "bastion", "jump host", "ssh tunnel", "scp", "rsync", "remote command".
allowed-tools: Bash(${CLAUDE_SKILL_DIR}/*), Bash(go run *), Bash(ssh *), Bash(scp *), Bash(rsync *)
---

When the user asks to connect to servers, run remote commands, transfer files, or manage SSH connections:

1. Auto-detect configured hosts from project and global config files.
2. Let the user pick which host or tag to target.
3. Run commands, transfer files, create tunnels, or check connectivity.
4. Use `${CLAUDE_SKILL_DIR}` to reference skill files.
5. **Never store SSH credentials — auth is handled by the user's SSH agent and ~/.ssh/config.**

## Host Configuration

Hosts are discovered from two sources (highest priority first):

1. **Project config** (`.claude/ssh-config.yaml`) — project-specific servers
2. **Global config** (`~/.claude/ssh-config.yaml`) — shared across all projects

When the same host name exists in both, the project config wins.

### Managing Hosts via CLI

```bash
# Add a host to project config (default)
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --add-host prod-api \
  --host 10.0.1.50 --port 22 --user deploy --bastion bastion-1 --tags prod,api

# Add to global config
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --add-host bastion-1 \
  --host bastion.example.com --user admin --tags bastion,prod --scope global

# Edit an existing host (only updates provided fields)
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --edit-host prod-api --user newuser
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --edit-host prod-api --tags prod,api,web

# Remove a host
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --remove-host staging-api
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --remove-host old-server --scope global

# List all hosts (shows source: project/global)
${CLAUDE_SKILL_DIR}/ssh-manager.sh list

# Show full config
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --show-config

# Manage named tunnels
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --add-tunnel prod-db \
  --tunnel-name postgres --tunnel-mapping "5432:localhost:5432"
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --remove-tunnel prod-db --tunnel-name postgres
```

### Config File Format

```yaml
# .claude/ssh-config.yaml (project) or ~/.claude/ssh-config.yaml (global)
hosts:
  bastion-1:
    host: bastion.example.com
    port: 22
    user: admin
    tags: [bastion, prod]

  prod-api-1:
    host: 10.0.1.50
    port: 22
    user: deploy
    bastion: bastion-1
    tags: [prod, api]

  prod-redis:
    host: 10.0.1.60
    user: deploy
    bastion: bastion-1
    tags: [prod, redis]
    tunnels:
      redis: "6379:localhost:6379"
      redis-insight: "8001:localhost:8001"
```

Config files are stored with `0600` permissions. No credentials are stored — authentication is handled by SSH agent/keys.

## Commands

| Command | Description | Example |
|---|---|---|
| `list` | List all configured hosts | `list` |
| `status` | Check connectivity of all hosts | `status` |
| `exec <host> <cmd>` | Run command on remote host | `exec prod-api "df -h"` |
| `exec-tag <tag> <cmd>` | Run command on all hosts with tag | `exec-tag prod "uptime"` |
| `tunnel <host> <name\|L:R:P>` | Start SSH tunnel (named or ad-hoc) | `tunnel prod-db postgres` |
| `scp <host> <src> <dest>` | Copy files (prefix remote path with `:`) | `scp prod-api :/var/log/app.log ./` |
| `rsync <host> <src> <dest>` | Rsync files (prefix remote path with `:`) | `rsync prod-api ./dist/ :/opt/app/` |
| `shell <host>` | Open interactive SSH session | `shell prod-api` |
| `check <host> [port]` | Test connectivity + optional port check | `check prod-redis 6379` |
| `check-tag <tag>` | Check all hosts with tag | `check-tag prod` |
| `tail <host> <file> [lines]` | Stream remote log file | `tail prod-api /var/log/app.log` |
| `info <host>` | Show CPU, memory, disk, uptime | `info prod-api` |
| `info-tag <tag>` | System info for all hosts with tag | `info-tag prod` |
| `config ...` | Manage config (delegates to Go helper) | `config --add-host ...` |

## Config Management Flags

| Flag | Description | Example |
|---|---|---|
| `--add-host <name>` | Add a host to config | `--add-host prod-api --host 10.0.1.50` |
| `--edit-host <name>` | Edit a host in config | `--edit-host prod-api --user newuser` |
| `--remove-host <name>` | Remove a host from config | `--remove-host staging` |
| `--list` | List all hosts | `--list` |
| `--get <name>` | Get host details (key=value) | `--get prod-api` |
| `--get-by-tag <tag>` | Get hosts by tag | `--get-by-tag prod` |
| `--show-config` | Show full config | `--show-config` |
| `--add-tunnel <host>` | Add named tunnel | `--add-tunnel prod-db --tunnel-name pg --tunnel-mapping 5432:localhost:5432` |
| `--remove-tunnel <host>` | Remove named tunnel | `--remove-tunnel prod-db --tunnel-name pg` |
| `--host <addr>` | Host address (for add/edit) | `--host 10.0.1.50` |
| `--port <n>` | SSH port (for add/edit, default 22) | `--port 2222` |
| `--user <user>` | SSH user (for add/edit) | `--user deploy` |
| `--bastion <name>` | Bastion host reference (for add/edit) | `--bastion bastion-1` |
| `--tags <t1,t2>` | Comma-separated tags (for add/edit) | `--tags prod,api` |
| `--scope <s>` | Target config: project or global | `--scope global` |

## Examples

### Remote Command Execution

```bash
# Run a single command
${CLAUDE_SKILL_DIR}/ssh-manager.sh exec prod-api "df -h"

# Run command on all production servers
${CLAUDE_SKILL_DIR}/ssh-manager.sh exec-tag prod "uptime"

# Check service status across all API servers
${CLAUDE_SKILL_DIR}/ssh-manager.sh exec-tag api "systemctl status myapp"
```

### SSH Tunnels

```bash
# Use a named tunnel (defined in config)
${CLAUDE_SKILL_DIR}/ssh-manager.sh tunnel prod-redis redis

# Ad-hoc tunnel
${CLAUDE_SKILL_DIR}/ssh-manager.sh tunnel prod-db 5432:localhost:5432

# Tunnel to access Redis through bastion
${CLAUDE_SKILL_DIR}/ssh-manager.sh tunnel prod-redis 6379:localhost:6379
```

### File Transfer

```bash
# Download a file from remote
${CLAUDE_SKILL_DIR}/ssh-manager.sh scp prod-api :/var/log/app.log ./

# Upload a file to remote
${CLAUDE_SKILL_DIR}/ssh-manager.sh scp prod-api ./config.yaml :/etc/myapp/

# Sync a directory
${CLAUDE_SKILL_DIR}/ssh-manager.sh rsync prod-api ./dist/ :/opt/app/dist/
```

### Connectivity & Monitoring

```bash
# Check all servers
${CLAUDE_SKILL_DIR}/ssh-manager.sh status

# Check a specific server and port
${CLAUDE_SKILL_DIR}/ssh-manager.sh check prod-redis 6379

# Check all production servers
${CLAUDE_SKILL_DIR}/ssh-manager.sh check-tag prod

# Get system info
${CLAUDE_SKILL_DIR}/ssh-manager.sh info prod-api

# Get system info for all production servers
${CLAUDE_SKILL_DIR}/ssh-manager.sh info-tag prod

# Tail a remote log file
${CLAUDE_SKILL_DIR}/ssh-manager.sh tail prod-api /var/log/app.log 200
```

### Interactive Shell

The `shell` command opens an interactive SSH session. The user must run it with the `!` prefix:

```bash
! ${CLAUDE_SKILL_DIR}/ssh-manager.sh shell prod-api
```

Similarly, `tail` is interactive (streams continuously):

```bash
! ${CLAUDE_SKILL_DIR}/ssh-manager.sh tail prod-api /var/log/app.log
```

## Bastion / Jump Host Support

When a host has `bastion: bastion-1`, the skill automatically adds `-J user@bastion:port` to all SSH/SCP/rsync commands. Multi-hop is supported — if the bastion itself references another bastion, the chain is resolved recursively with circular reference detection.

## Security Notes

- **No credentials are stored** — authentication is handled by the user's SSH agent, SSH keys, or `~/.ssh/config`.
- Config files contain only hostnames, ports, users, and tags — no passwords or key paths.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save host information to auto-memory or memory files.**
- Do not commit config files with sensitive hostnames — `.claude/ssh-config.yaml` should be gitignored.
