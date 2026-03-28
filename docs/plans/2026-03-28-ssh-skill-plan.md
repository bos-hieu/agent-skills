# SSH Skill Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create an SSH skill that manages named server connections with bastion support, tunneling, file transfer, multi-host execution, and system monitoring.

**Architecture:** Bash script (`ssh-manager.sh`) handles all SSH operations. Go helper (`config.go`) handles YAML config CRUD. Config stored in `.claude/ssh-config.yaml` (project) and `~/.claude/ssh-config.yaml` (global).

**Tech Stack:** Bash (SSH/SCP/rsync operations), Go (YAML config management with `gopkg.in/yaml.v3`)

---

### Task 1: Go Config Helper — Full implementation

**Files:**
- Create: `skills/ssh/config.go`

Complete Go program with:
- HostEntry struct (host, port, user, bastion, tags, tunnels)
- ConfigFile struct with YAML marshaling
- loadConfig/saveConfig with 0600 permissions
- findProjectConfig (walks up to find .claude dir)
- globalConfigPath
- mergedHosts (project overrides global)
- CLI: --add-host, --edit-host, --remove-host, --list, --get, --get-by-tag, --show-config
- CLI: --add-tunnel, --remove-tunnel with --tunnel-name, --tunnel-mapping
- --scope global|project flag

### Task 2: Bash Script — Full implementation

**Files:**
- Create: `skills/ssh/ssh-manager.sh`

Complete bash script with:
- Core: usage, get_host_info (calls Go --get), build_ssh_cmd, build_scp_cmd, build_jump_chain (recursive bastion resolution with cycle detection)
- Commands: exec, exec-tag, shell, tunnel (named + ad-hoc), scp, rsync, check, check-tag, status, tail, info, info-tag, config (passthrough to Go), list (passthrough)
- Multi-host output prefixed with [hostname]
- Tunnel supports both named lookups and ad-hoc L:R:P format

### Task 3: SKILL.md

**Files:**
- Create: `skills/ssh/SKILL.md`

Skill definition following database skill pattern with frontmatter, config management docs, command reference, examples, and security notes.

### Task 4: Update README.md and version

**Files:**
- Modify: `README.md` — add SSH skill to table
- Modify: `.claude-plugin/plugin.json` — bump to 1.3.0
