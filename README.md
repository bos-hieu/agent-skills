# Agent Skills

A collection of reusable skills for AI coding agents. Each skill teaches your coding agent a specific workflow — install once and every future session benefits automatically.

## Available Skills

| Skill | Description |
|-------|-------------|
| [database](skills/database/SKILL.md) | Query PostgreSQL, MySQL, SQLite, and MongoDB databases. Auto-detects connections from config files and env vars. |
| [pdf](skills/pdf/SKILL.md) | Extract text from PDF files with page ranges, keyword search, and content summarization. |
| [xlsx](skills/xlsx/SKILL.md) | Read Excel .xlsx/.xlsm files with sheet selection, column filtering, CSV export, and keyword search. |
| [browser-test](skills/browser-test/SKILL.md) | Run and debug Playwright browser tests (API, E2E, smoke) with smart filtering and failure reporting. |
| [go-issue-solver](skills/go-issue-solver/SKILL.md) | Analyze and solve Go codebase issues — search, diagnose, scaffold fixes, and run tests. |
| [generating-claude-instructions](skills/generating-claude-instructions/SKILL.md) | Generate a CLAUDE.md file by deeply exploring source code. Only verified facts, never guesses. |

## Installation

### Claude Code

This is a **private** repository. Claude Code supports installing plugins from private repos.

1. Register this repository as a private plugin marketplace:
   ```bash
   /plugin marketplace add https://github.com/bos-hieu/agent-skills.git
   ```

2. Install the plugin:
   ```bash
   /plugin install agent-skills
   ```

3. Reload to activate:
   ```bash
   /reload-plugins
   ```

> **How it works:** Claude Code reuses your local git credentials. As long as `git clone https://github.com/bos-hieu/agent-skills.git` works on your machine, plugin installation will work too. Team members need access to this repo.

### Cursor

In Cursor Agent chat:

```text
/add-plugin agent-skills
```

Or search for "agent-skills" in the Cursor plugin marketplace.

### Codex

Clone this private repo and tell Codex to follow the install instructions:

```bash
git clone https://github.com/bos-hieu/agent-skills.git ~/.codex/agent-skills
```

Then tell Codex:

```
Follow the instructions in ~/.codex/agent-skills/.codex/INSTALL.md
```

### GitHub Copilot

Custom instructions are automatically picked up from `.github/copilot-instructions.md` when this repository is cloned. No additional setup is needed.

### Gemini CLI

```bash
gemini extensions install https://github.com/bos-hieu/agent-skills.git
```

To update:

```bash
gemini extensions update agent-skills
```

## How It Works

Skills are markdown files that live in the `skills/` directory. Each skill has:

- **A `SKILL.md` file** — The complete process definition with frontmatter metadata
- **Supporting files** (optional) — References, examples, anti-patterns

When your coding agent encounters a task matching a skill's trigger, it follows the documented process automatically.

## Adding a New Skill

1. Create a new directory under `skills/` with a kebab-case name
2. Add a `SKILL.md` file with YAML frontmatter:

   ```markdown
   ---
   name: your-skill-name
   description: When to use this skill
   ---

   # Skill Title

   ## Overview
   ...

   ## When to Use
   ...

   ## Process
   ...
   ```

3. Submit a pull request

## Updating

### Claude Code

When a new version is released, the simplest way to upgrade is to uninstall and reinstall:

```bash
/plugin uninstall agent-skills
/plugin marketplace remove agent-skills-dev
/plugin marketplace add https://github.com/bos-hieu/agent-skills.git
/plugin install agent-skills
/reload-plugins
```

> **Why not `/plugin update`?** The update command checks the version in `plugin.json`. If the cached marketplace still has the old version, it reports "already at latest". A clean reinstall guarantees the latest code.

### Codex

```bash
cd ~/.codex/agent-skills && git pull
```

### Gemini CLI

```bash
gemini extensions update agent-skills
```

## License

MIT License — see [LICENSE](LICENSE) for details.
