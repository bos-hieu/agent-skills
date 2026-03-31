---
name: pdf
description: Read PDF files with Go. Extracts text page by page, supports page ranges, keyword search, and summarizes content. Notes OCR requirement for scanned PDFs.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

Use `${CLAUDE_SKILL_DIR}` to reference the Go file. For scanned/image-only PDFs, inform the user OCR is needed (e.g. `ocrmypdf scanned.pdf searchable.pdf`).

## Usage

```bash
go run ${CLAUDE_SKILL_DIR}/read_pdf.go <file.pdf> [flags]
```

| Flag | Description |
|------|-------------|
| `--pages <range>` | Page range: `1-5`, `3`, `2,4,7` |
| `--search <term>` | Search keyword (case-insensitive), report page + line |
| `--lines <n>` | Max lines per page in preview (default 20) |
| `--summary` | Print only page count and first page content |
| `--full` | Print full text of every page (no line limit) |
