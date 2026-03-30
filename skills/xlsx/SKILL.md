---
name: xlsx
description: Read Excel .xlsx and macro-enabled .xlsm files with Go. Supports sheet selection, column filtering, CSV export, and keyword search.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

Use `${CLAUDE_SKILL_DIR}` to reference the Go file. Supports both `.xlsx` and `.xlsm` (macro-enabled, VBA not executed).

## Usage

```bash
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go <file> [flags]
```

| Flag | Description |
|------|-------------|
| `--sheet <name>` | Focus on a specific sheet |
| `--columns <c1,c2>` | Filter to specific columns (case-insensitive) |
| `--rows <n>` | Preview rows (default 5) |
| `--search <term>` | Search keyword across all cells (case-insensitive) |
| `--csv` | Export sheet to CSV (stdout) |
| `--csv-file <path>` | Export sheet to CSV file |
| `--list-sheets` | List sheet names and exit |

CSV export writes headers + all data rows (not just preview).
