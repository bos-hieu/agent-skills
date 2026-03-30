---
name: openclaw-docker-setup
description: Use when setting up OpenClaw in Docker, running OpenClaw gateway in a container, troubleshooting a Dockerized OpenClaw installation, or managing plugins in an OpenClaw instance. Triggers on "openclaw docker", "openclaw container", "run openclaw in docker", "openclaw setup", "openclaw plugin", or "install openclaw plugin".
---

Manage multiple named OpenClaw gateway instances via `openclaw.sh`.

**Script:** `~/.claude/skills/openclaw-docker-setup/openclaw.sh`

## Commands

```bash
openclaw.sh create <name> [-p PORT]   # create instance (auto-port or specific)
openclaw.sh onboard <name>            # interactive setup (required once)
openclaw.sh start|stop|restart <name> # manage gateway
openclaw.sh logs|shell <name>         # tail logs / open bash
openclaw.sh remove <name>             # remove container + optionally data
openclaw.sh list                      # list all instances
openclaw.sh status                    # show running status
```

## Plugin Commands

```bash
openclaw.sh plugin <name> install <package>              # from ClawHub/npm
openclaw.sh plugin <name> install-local ~/agent-skills   # from local clone
openclaw.sh plugin <name> list|status|doctor             # info commands
openclaw.sh plugin <name> update <id|--all>              # update plugins
openclaw.sh plugin <name> enable|disable <id>            # toggle plugins
openclaw.sh plugin <name> inspect <id>                   # plugin details
```

## Instance Layout

Each instance gets: container `openclaw-<name>`, config `~/.openclaw-instances/<name>/config`, workspace `~/.openclaw-instances/<name>/workspace`, ports auto-assigned from 18789 (gateway) and 18790 (control), incrementing by 2.

## First-Time Workflow

```bash
openclaw.sh create myinstance
openclaw.sh onboard myinstance   # interactive — run inside: npm install -g openclaw@latest && openclaw setup && openclaw onboard && exit
openclaw.sh start myinstance     # then open http://127.0.0.1:<PORT>
```

## Installing Local Plugins

OpenClaw rejects git URL specs. Clone locally, then copy into container:

```bash
git clone https://github.com/bos-hieu/agent-skills.git ~/agent-skills
openclaw.sh plugin alice install-local ~/agent-skills
# To update: cd ~/agent-skills && git pull && openclaw.sh plugin alice install-local ~/agent-skills
```

## Notes

- `onboard` is interactive (requires user input) — always uses `docker exec -it`
- `create` only makes the container; must `onboard` before `start`
- Gateway must be running before installing plugins
- Use `openclaw.sh status` to check ports before assigning manually
