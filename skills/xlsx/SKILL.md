---
name: xlsx
description: Read Excel .xlsx and macro-enabled .xlsm files with Go. Supports sheet selection, column filtering, CSV export, and keyword search.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to read an `.xlsx` or `.xlsm` (macro-enabled) file:

1. Use the bundled Go program to open the file.
2. List all sheet names first.
3. Show the first few rows of each sheet (or the specified sheet).
4. Summarize the headers/columns.
5. Apply any filters, exports, or searches as requested.
6. Use `${CLAUDE_SKILL_DIR}` to reference the Go file reliably.

## Flags / Options

| Flag | Description | Example |
|------|-------------|---------|
| `--sheet <name>` | Focus on a specific sheet | `--sheet Sheet1` |
| `--columns <c1,c2>` | Filter to specific columns only | `--columns Name,Age` |
| `--rows <n>` | Number of preview rows (default 5) | `--rows 10` |
| `--search <term>` | Search for a keyword across all cells | `--search "invoice"` |
| `--csv` | Export the sheet to CSV (stdout) | `--csv` |
| `--csv-file <path>` | Export the sheet to a CSV file | `--csv-file out.csv` |
| `--list-sheets` | Only list sheet names and exit | `--list-sheets` |

## Examples

```bash
# Basic read
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./data.xlsx

# Macro-enabled file (.xlsm) — same command, auto-detected
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./macro_report.xlsm

# Read specific sheet
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./data.xlsx --sheet "Sales"

# Filter columns
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./data.xlsx --sheet "Sales" --columns "Date,Amount,Status"

# Export to CSV
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./data.xlsx --sheet "Sales" --csv-file output.csv

# Search for keyword
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./data.xlsx --search "overdue"

# List sheets only
go run ${CLAUDE_SKILL_DIR}/read_xlsx.go ./data.xlsx --list-sheets
```

## Notes

- `.xlsm` (macro-enabled) files are supported via the excelize library; VBA macros themselves are not executed.
- Column filtering is case-insensitive.
- CSV export writes headers + all data rows (not just the preview).
- Keyword search is case-insensitive and reports sheet, row, column, and matched value.
