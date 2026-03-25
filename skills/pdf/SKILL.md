---
name: pdf
description: Read PDF files with Go. Extracts text page by page, supports page ranges, keyword search, and summarizes content. Notes OCR requirement for scanned PDFs.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to read a PDF file:

1. Use the bundled Go program to extract text from the PDF.
2. Read page by page (or the requested page range).
3. Summarize the document clearly.
4. If the user asks a specific question, answer based on the extracted text.
5. For scanned/image-only PDFs, inform the user that OCR is required and suggest tools.
6. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.

## Flags / Options

| Flag | Description | Example |
|------|-------------|---------|
| `--pages <range>` | Page range to read, e.g. `1-5`, `3`, `2,4,7` | `--pages 1-10` |
| `--search <term>` | Search for a keyword (case-insensitive), report page + line | `--search "contract"` |
| `--lines <n>` | Max lines to show per page in preview (default 20) | `--lines 30` |
| `--summary` | Print only page count and first page content | `--summary` |
| `--full` | Print full text of every page (no line limit) | `--full` |

## Examples

```bash
# Basic read (first 20 lines per page)
go run ${CLAUDE_SKILL_DIR}/read_pdf.go ./report.pdf

# Read pages 1 to 5 only
go run ${CLAUDE_SKILL_DIR}/read_pdf.go ./report.pdf --pages 1-5

# Read specific pages
go run ${CLAUDE_SKILL_DIR}/read_pdf.go ./report.pdf --pages 2,4,7

# Search for a keyword
go run ${CLAUDE_SKILL_DIR}/read_pdf.go ./contract.pdf --search "payment terms"

# Print full text without line limit
go run ${CLAUDE_SKILL_DIR}/read_pdf.go ./report.pdf --full

# Quick summary (page count + first page)
go run ${CLAUDE_SKILL_DIR}/read_pdf.go ./report.pdf --summary
```

## OCR Note

This skill extracts **embedded text** from PDFs. For scanned or image-based PDFs:
- Text extraction will return empty or garbled results.
- Use an OCR tool such as `tesseract`, `ocrmypdf`, or an online service to convert to a text-based PDF first.
- Example: `ocrmypdf scanned.pdf searchable.pdf` then re-run this skill on `searchable.pdf`.
