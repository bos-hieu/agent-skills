---
name: ssh
description: Manage SSH connections to servers with bastion/jump host support, tunneling, file transfer, multi-host execution, and system monitoring. Triggers on "ssh", "remote server", "bastion", "jump host", "ssh tunnel", "scp", "rsync", "remote command".
allowed-tools: Bash(${CLAUDE_SKILL_DIR}/*), Bash(go run *), Bash(ssh *), Bash(scp *), Bash(rsync *)
---

Use `${CLAUDE_SKILL_DIR}` to reference skill files.
**No credentials stored — auth handled by user's SSH agent and ~/.ssh/config.**

## Host Configuration

Hosts discovered from (highest priority first):
1. Project config: `.claude/ssh-config.yaml`
2. Global config: `~/.claude/ssh-config.yaml`

```yaml
# .claude/ssh-config.yaml
hosts:
  bastion-1:
    host: bastion.example.com
    port: 22
    user: admin
    tags: [bastion, prod]
  prod-api-1:
    host: 10.0.1.50
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
```

### Managing Hosts

```bash
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --add-host <name> --host <addr> [--port N] [--user U] [--bastion B] [--tags t1,t2] [--scope global]
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --edit-host <name> [fields to update]
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --remove-host <name> [--scope global]
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --add-tunnel <host> --tunnel-name <name> --tunnel-mapping "L:R:P"
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --remove-tunnel <host> --tunnel-name <name>
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --show-config
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --get <name>          # get host details
${CLAUDE_SKILL_DIR}/ssh-manager.sh config --get-by-tag <tag>    # get hosts by tag
```

## Commands

| Command | Description |
|---|---|
| `list` | List all configured hosts |
| `status` | Check connectivity of all hosts |
| `exec <host> <cmd>` | Run command on remote host |
| `exec-tag <tag> <cmd>` | Run command on all hosts with tag |
| `tunnel <host> <name\|L:R:P>` | Start SSH tunnel (named or ad-hoc) |
| `scp <host> <src> <dest>` | Copy files (prefix remote path with `:`) |
| `rsync <host> <src> <dest>` | Rsync files (prefix remote path with `:`) |
| `shell <host>` | Open interactive SSH session |
| `check <host> [port]` | Test connectivity + optional port check |
| `check-tag <tag>` | Check all hosts with tag |
| `tail <host> <file> [lines]` | Stream remote log file |
| `info <host>` | Show CPU, memory, disk, uptime |
| `info-tag <tag>` | System info for all hosts with tag |

## Notes

- When a host has `bastion: <name>`, `-J user@bastion:port` is added automatically. Multi-hop supported with circular reference detection.
- `shell` and `tail` are interactive — user must run with `!` prefix.
- Config files created with `0600` permissions. Never save host info to auto-memory. Gitignore `.claude/ssh-config.yaml`.
