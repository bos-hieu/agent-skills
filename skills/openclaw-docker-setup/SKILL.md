---
name: openclaw-docker-setup
description: Use when setting up OpenClaw in Docker, running OpenClaw gateway in a container, troubleshooting a Dockerized OpenClaw installation, or managing plugins in an OpenClaw instance. Triggers on "openclaw docker", "openclaw container", "run openclaw in docker", "openclaw setup", "openclaw plugin", or "install openclaw plugin".
---

# OpenClaw Docker Setup

Manage multiple named OpenClaw gateway instances via `openclaw.sh`.

**Script location:** `~/.claude/skills/openclaw-docker-setup/openclaw.sh`

## Quick Reference

```bash
openclaw.sh create alice              # create instance on auto-assigned port
openclaw.sh create bob -p 18800      # create instance on specific port
openclaw.sh onboard alice             # interactive setup (run once per instance)
openclaw.sh start alice               # start gateway in background
openclaw.sh stop alice                # stop instance
openclaw.sh restart alice             # restart gateway
openclaw.sh logs alice                # tail logs
openclaw.sh shell alice               # open bash in container
openclaw.sh remove alice              # remove container + optionally data
openclaw.sh list                      # list all instances
openclaw.sh status                    # show running status of all instances

# Plugin management
openclaw.sh plugin alice install <package>              # install from ClawHub/npm
openclaw.sh plugin alice install-local ~/agent-skills   # install from local clone
openclaw.sh plugin alice list                           # list installed plugins
openclaw.sh plugin alice update <id>                    # update one plugin
openclaw.sh plugin alice update --all                   # update all plugins
openclaw.sh plugin alice enable <id>                    # enable a plugin
openclaw.sh plugin alice disable <id>                   # disable a plugin
openclaw.sh plugin alice status                         # plugin operational summary
openclaw.sh plugin alice doctor                         # plugin diagnostics
openclaw.sh plugin alice inspect <id>                   # show plugin details
```

## How It Works

Each instance gets:
- **Container**: `openclaw-<name>`
- **Config dir**: `~/.openclaw-instances/<name>/config`
- **Workspace**: `~/.openclaw-instances/<name>/workspace`
- **Ports**: auto-assigned starting from 18789 (gateway) and 18790 (control), incrementing by 2 for each new instance. Override with `-p`.

## First-Time Workflow

```bash
# 1. Create the instance
openclaw.sh create myinstance

# 2. Interactive onboard (required once)
openclaw.sh onboard myinstance
# Inside container, run:
#   npm install -g openclaw@latest
#   openclaw setup
#   openclaw onboard
#   openclaw config set gateway.controlUi.allowedOrigins '["http://127.0.0.1:PORT","http://localhost:PORT"]' --strict-json
#   exit

# 3. Start the gateway
openclaw.sh start myinstance

# 4. Open in browser
# http://127.0.0.1:<PORT>
```

## Subsequent Starts

```bash
openclaw.sh start myinstance
```

## Multiple Instances Example

```bash
openclaw.sh create work
openclaw.sh create personal -p 18800
openclaw.sh create testing -p 18810

openclaw.sh onboard work
openclaw.sh onboard personal
openclaw.sh onboard testing

openclaw.sh start work
openclaw.sh start personal
openclaw.sh start testing

openclaw.sh status
# NAME            CONTAINER            PORT     STATUS       URL
# work            openclaw-work        18789    running      http://127.0.0.1:18789
# personal        openclaw-personal    18800    running      http://127.0.0.1:18800
# testing         openclaw-testing     18810    running      http://127.0.0.1:18810
```

## Plugin Management

Plugins extend an OpenClaw instance with new capabilities (channels, model providers, tools, skills, etc.). They are managed with the `plugin` subcommand, which runs `openclaw plugins` inside the target container.

### Install from ClawHub or npm

```bash
openclaw.sh plugin alice install my-plugin          # ClawHub first, then npm
openclaw.sh plugin alice install clawhub:my-plugin  # ClawHub only
```

### Install from a local clone (git clone → docker cp)

OpenClaw rejects git URL specs, so the workflow for installing a bundle repo (e.g. this one) is to clone it locally and then copy it into the container:

```bash
# 1. Clone to local machine
git clone https://github.com/bos-hieu/agent-skills.git ~/agent-skills

# 2. Copy into container and install
openclaw.sh plugin alice install-local ~/agent-skills
```

`install-local` uses `docker cp` to copy the directory into `/tmp/openclaw-bundles/<name>/` inside the container, then runs `openclaw plugins install` on that path. OpenClaw auto-detects the bundle format from `.claude-plugin/plugin.json` or `.cursor-plugin/plugin.json`.

**To update:**
```bash
cd ~/agent-skills && git pull
openclaw.sh plugin alice install-local ~/agent-skills
```

### Other plugin commands

```bash
openclaw.sh plugin alice list                  # list installed plugins
openclaw.sh plugin alice update --all          # update all plugins
openclaw.sh plugin alice enable my-plugin      # enable a plugin
openclaw.sh plugin alice disable my-plugin     # disable a plugin
openclaw.sh plugin alice doctor                # diagnostics
```

**Notes:**
- The gateway must be running (`openclaw.sh start <name>`) before installing plugins.
- Config changes (enable/disable) take effect after an automatic gateway restart; if auto-restart is off, run `openclaw.sh restart <name>`.

## Common Mistakes

- Running `onboard` non-interactively -- it requires user input, always uses `docker exec -it`
- Forgetting to onboard -- `create` only makes the container; you must `onboard` before `start`
- Port conflicts -- use `openclaw.sh status` to check existing ports before assigning manually

## When Using This Skill

When the user asks to set up OpenClaw in Docker, run this script via Bash tool. The script handles all Docker commands. For first-time setup, the `onboard` step is interactive and must be run by the user themselves (suggest `! openclaw.sh onboard <name>`).
