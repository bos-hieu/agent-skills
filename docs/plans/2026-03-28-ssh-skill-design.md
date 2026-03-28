# SSH Skill Design

## Overview

A skill for managing SSH connections to multiple servers, with support for bastion/jump hosts, tunneling, file transfer, and multi-host command execution.

## Architecture

- `skills/ssh/SKILL.md` — skill definition
- `skills/ssh/ssh-manager.sh` — bash script for all SSH operations
- `skills/ssh/config.go` — Go helper for YAML config CRUD only

## Config

**Locations:** `.claude/ssh-config.yaml` (project) and `~/.claude/ssh-config.yaml` (global). Merged at runtime, project overrides global.

**Format:**

```yaml
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
    port: 22
    user: deploy
    bastion: bastion-1
    tags: [prod, redis]
    tunnels:
      redis: 6379:localhost:6379
      redis-insight: 8001:localhost:8001
```

**No credentials stored.** Auth handled by user's SSH agent / ~/.ssh/config.

## Config Commands (Go helper)

- `--add-host <name>` with `--host`, `--port`, `--user`, `--bastion`, `--tags`
- `--edit-host <name>` — update fields
- `--remove-host <name>` — delete entry
- `--list` — list all hosts with source indicator
- `--get <name>` — output host as key=value for bash
- `--get-by-tag <tag>` — output matching hosts
- `--show-config` — full config
- `--add-tunnel <host> <name> <mapping>` — add named tunnel
- `--remove-tunnel <host> <name>` — remove tunnel
- `--scope global|project` — target config file

## SSH Operations (bash script)

| Command | Description |
|---------|-------------|
| `exec <host> <cmd>` | Run command on remote host |
| `exec-tag <tag> <cmd>` | Run command on all hosts with tag |
| `tunnel <host> <tunnel-name>` | Start named tunnel from config |
| `tunnel <host> <L:R:P>` | Ad-hoc tunnel |
| `scp <host> <src> <dest>` | Copy files (`:` prefix = remote) |
| `rsync <host> <src> <dest>` | Rsync files |
| `shell <host>` | Interactive SSH (user runs with `!`) |
| `check <host> [port]` | Test connectivity + optional port |
| `check-tag <tag>` | Check all hosts with tag |
| `tail <host> <file>` | Stream remote log |
| `info <host>` | CPU, memory, disk, uptime |
| `info-tag <tag>` | System info for all tagged hosts |
| `config ...` | Delegates to Go helper |
| `list` | List all hosts |
| `status` | Connectivity check all hosts |

## Bastion Handling

When host has `bastion: bastion-1`, automatically adds `-J user@bastion:port`. Multi-hop supported by chaining bastion references.

## Multi-host Output

```
[prod-api-1] output line here
[prod-api-2] output line here
```
