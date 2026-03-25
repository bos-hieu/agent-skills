# Installing Agent Skills for Codex

Enable agent-skills in Codex via native skill discovery. Just clone and symlink.

## Prerequisites

- Git

## Installation

1. **Clone the agent-skills repository:**
   ```bash
   git clone https://github.com/bos-hieu/agent-skills.git ~/.codex/agent-skills
   ```

2. **Create the skills symlink:**
   ```bash
   mkdir -p ~/.agents/skills
   ln -s ~/.codex/agent-skills/skills ~/.agents/skills/agent-skills
   ```

   **Windows (PowerShell):**
   ```powershell
   New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.agents\skills"
   cmd /c mklink /J "$env:USERPROFILE\.agents\skills\agent-skills" "$env:USERPROFILE\.codex\agent-skills\skills"
   ```

3. **Restart Codex** (quit and relaunch the CLI) to discover the skills.

## Verify

```bash
ls -la ~/.agents/skills/agent-skills
```

You should see a symlink (or junction on Windows) pointing to your agent-skills skills directory.

## Updating

```bash
cd ~/.codex/agent-skills && git pull
```

Skills update instantly through the symlink.

## Uninstalling

```bash
rm ~/.agents/skills/agent-skills
```

Optionally delete the clone: `rm -rf ~/.codex/agent-skills`.
