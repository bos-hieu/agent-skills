# Agent Skills

A collection of reusable skills for AI coding agents. Each skill teaches your coding agent a specific workflow — install once and every future session benefits automatically.

## Available Skills

| Skill | Description |
|-------|-------------|
| [generating-claude-instructions](skills/generating-claude-instructions/SKILL.md) | Generate a CLAUDE.md file at the project root by deeply exploring the actual source code. The file must contain only verified facts — never guess or infer. |

## Installation

### Claude Code

**Install as a private plugin (recommended):**

1. Register this repository as a plugin marketplace:
   ```bash
   /plugin marketplace add bos-hieu/agent-skills
   ```

2. Install the plugin:
   ```bash
   /plugin install agent-skills@agent-skills-dev
   ```

3. Choose installation scope:
   ```bash
   # User-wide (default) — available in all your projects
   /plugin install agent-skills@agent-skills-dev --scope user

   # Project-only — shared with team via version control
   /plugin install agent-skills@agent-skills-dev --scope project
   ```

> **Private repos:** If this is a private repository, Claude Code uses your existing git credentials. As long as `git clone` works for you (via SSH key, GitHub PAT, or credential helper), plugin installation will work automatically.

### Cursor

In Cursor Agent chat:

```text
/add-plugin agent-skills
```

Or search for "agent-skills" in the Cursor plugin marketplace.

### Codex

Tell Codex:

```
Fetch and follow instructions from https://raw.githubusercontent.com/bos-hieu/agent-skills/refs/heads/main/.codex/INSTALL.md
```

### GitHub Copilot

Custom instructions are automatically picked up from `.github/copilot-instructions.md` when this repository is cloned. No additional setup is needed.

### Gemini CLI

```bash
gemini extensions install https://github.com/bos-hieu/agent-skills
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

```bash
/plugin marketplace update
/plugin update agent-skills
```

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
