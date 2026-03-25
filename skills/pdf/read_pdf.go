package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

func main() {
	pagesFlag := flag.String("pages", "", "Page range to read: '1-5', '3', or '2,4,7' (default: all)")
	searchFlag := flag.String("search", "", "Keyword to search (case-insensitive)")
	linesFlag := flag.Int("lines", 20, "Max lines to preview per page")
	fullFlag := flag.Bool("full", false, "Print full text of every page (no line limit)")
	summaryFlag := flag.Bool("summary", false, "Print page count + first page only")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: read_pdf.go <file.pdf> [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	filePath := flag.Arg(0)

	f, r, err := pdf.Open(filePath)
	if err != nil {
		log.Fatalf("cannot open PDF: %v", err)
	}
	defer f.Close()

	totalPages := r.NumPage()
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("Total pages: %d\n\n", totalPages)

	if *summaryFlag {
		printPage(r, 1, 0) // 0 = no line limit for summary page
		return
	}

	// Determine which pages to process
	pageSet, err := parsePageSpec(*pagesFlag, totalPages)
	if err != nil {
		log.Fatalf("invalid --pages value: %v", err)
	}

	lineLimit := *linesFlag
	if *fullFlag {
		lineLimit = 0 // 0 = unlimited
	}

	if *searchFlag != "" {
		searchPDF(r, pageSet, *searchFlag)
		return
	}

	for _, pageNum := range pageSet {
		printPage(r, pageNum, lineLimit)
	}
}

// printPage prints the text of a single page with an optional line limit (0 = all).
func printPage(r *pdf.Reader, pageNum, lineLimit int) {
	p := r.Page(pageNum)
	if p.V.IsNull() {
		fmt.Printf("=== Page %d ===\n(page not found)\n", pageNum)
		return
	}

	text, err := p.GetPlainText(nil)
	if err != nil {
		fmt.Printf("=== Page %d ===\ncannot extract text: %v\n", pageNum, err)
		return
	}

	content := strings.TrimSpace(text)
	if content == "" {
		fmt.Printf("=== Page %d ===\n(no text found — may be a scanned/image page; OCR required)\n\n", pageNum)
		return
	}

	lines := strings.Split(content, "\n")
	fmt.Printf("=== Page %d (%d lines) ===\n", pageNum, len(lines))

	limit := len(lines)
	truncated := false
	if lineLimit > 0 && len(lines) > lineLimit {
		limit = lineLimit
		truncated = true
	}

	for i := 0; i < limit; i++ {
		fmt.Println(lines[i])
	}
	if truncated {
		fmt.Printf("... (%d more lines, use --full to see all)\n", len(lines)-limit)
	}
	fmt.Println()
}

// searchPDF searches for a keyword across the specified pages.
func searchPDF(r *pdf.Reader, pages []int, keyword string) {
	kw := strings.ToLower(keyword)
	fmt.Printf("Searching for %q...\n\n", keyword)

	totalMatches := 0
	for _, pageNum := range pages {
		p := r.Page(pageNum)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}

		lines := strings.Split(text, "\n")
		for lineIdx, line := range lines {
			if strings.Contains(strings.ToLower(line), kw) {
				fmt.Printf("  Page %d, Line %d: %s\n", pageNum, lineIdx+1, strings.TrimSpace(line))
				totalMatches++
			}
		}
	}

	if totalMatches == 0 {
		fmt.Println("No matches found.")
	} else {
		fmt.Printf("\nTotal matches: %d\n", totalMatches)
	}
}

// parsePageSpec parses a page specification string into an ordered slice of page numbers.
// Formats: "" (all), "3" (single), "1-5" (range), "2,4,7" (list), "1-3,5,7-9" (mixed).
func parsePageSpec(spec string, total int) ([]int, error) {
	if spec == "" {
		pages := make([]int, total)
		for i := range pages {
			pages[i] = i + 1
		}
		return pages, nil
	}

	seen := map[int]bool{}
	var pages []int

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid page %q", part)
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid page %q", part)
			}
			if start < 1 || end > total || start > end {
				return nil, fmt.Errorf("page range %d-%d out of bounds (1-%d)", start, end, total)
			}
			for p := start; p <= end; p++ {
				if !seen[p] {
					pages = append(pages, p)
					seen[p] = true
				}
			}
		} else {
			p, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page %q", part)
			}
			if p < 1 || p > total {
				return nil, fmt.Errorf("page %d out of bounds (1-%d)", p, total)
			}
			if !seen[p] {
				pages = append(pages, p)
				seen[p] = true
			}
		}
	}
	return pages, nil
}
