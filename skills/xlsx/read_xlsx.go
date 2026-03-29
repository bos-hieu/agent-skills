package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	sheetFlag := flag.String("sheet", "", "Sheet name to read (default: all sheets)")
	columnsFlag := flag.String("columns", "", "Comma-separated column names to include (default: all)")
	rowsFlag := flag.Int("rows", 5, "Number of preview rows to show")
	searchFlag := flag.String("search", "", "Keyword to search across all cells (case-insensitive)")
	csvFlag := flag.Bool("csv", false, "Export sheet to CSV on stdout")
	csvFileFlag := flag.String("csv-file", "", "Export sheet to a CSV file at the given path")
	listSheetsFlag := flag.Bool("list-sheets", false, "List sheet names and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: read_xlsx.go <file.xlsx|file.xlsm> [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	filePath := flag.Arg(0)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Fatalf("cannot open file: %v", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()

	// --list-sheets: just print and exit
	if *listSheetsFlag {
		fmt.Println("Sheets:")
		for _, s := range sheets {
			fmt.Printf("  - %s\n", s)
		}
		return
	}

	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("Sheets: %s\n", strings.Join(sheets, ", "))

	// Determine which sheets to process
	targetSheets := sheets
	if *sheetFlag != "" {
		found := false
		for _, s := range sheets {
			if strings.EqualFold(s, *sheetFlag) {
				targetSheets = []string{s}
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("sheet %q not found. Available sheets: %s", *sheetFlag, strings.Join(sheets, ", "))
		}
	}

	// Parse column filter
	var columnFilter []string
	if *columnsFlag != "" {
		for _, c := range strings.Split(*columnsFlag, ",") {
			columnFilter = append(columnFilter, strings.TrimSpace(strings.ToLower(c)))
		}
	}

	// --search mode
	if *searchFlag != "" {
		searchKeyword(f, targetSheets, *searchFlag)
		return
	}

	// CSV export mode
	if *csvFlag || *csvFileFlag != "" {
		if len(targetSheets) > 1 {
			log.Fatal("CSV export requires --sheet to select a single sheet")
		}
		exportCSV(f, targetSheets[0], columnFilter, *csvFileFlag)
		return
	}

	// Default: preview mode
	for _, sheet := range targetSheets {
		previewSheet(f, sheet, columnFilter, *rowsFlag)
	}
}

// previewSheet prints a preview of a sheet's contents.
func previewSheet(f *excelize.File, sheet string, columnFilter []string, maxRows int) {
	fmt.Printf("\n=== Sheet: %s ===\n", sheet)

	rows, err := f.GetRows(sheet)
	if err != nil {
		fmt.Printf("cannot read sheet: %v\n", err)
		return
	}
	if len(rows) == 0 {
		fmt.Println("(empty sheet)")
		return
	}

	// Expand merged cell values so all cells in a merged range show the value,
	// not just the top-left cell.
	rows = expandMergedCells(f, sheet, rows)

	// Collect non-empty rows with their 1-based Excel row numbers.
	type numberedRow struct {
		excelRow int
		data     []string
	}
	var dataRows []numberedRow
	for i, row := range rows {
		if !isEmptyRow(row) {
			dataRows = append(dataRows, numberedRow{excelRow: i + 1, data: row})
		}
	}
	if len(dataRows) == 0 {
		fmt.Println("(empty sheet)")
		return
	}

	headers := dataRows[0].data
	colIndexes := resolveColumnIndexes(headers, columnFilter)

	selectedHeaders := selectCols(headers, colIndexes)
	fmt.Printf("Columns (%d): %s\n", len(selectedHeaders), strings.Join(selectedHeaders, ", "))
	fmt.Printf("Total rows (including header): %d\n", len(rows))
	fmt.Printf("Non-empty rows: %d\n", len(dataRows))

	limit := maxRows + 1 // +1 for header
	if len(dataRows) < limit {
		limit = len(dataRows)
	}

	fmt.Println("\nPreview:")
	for i := 0; i < limit; i++ {
		nr := dataRows[i]
		label := fmt.Sprintf("row %d (Excel row %d)", i, nr.excelRow)
		if i == 0 {
			label = fmt.Sprintf("header (Excel row %d)", nr.excelRow)
		}
		fmt.Printf("  [%s] %v\n", label, selectCols(nr.data, colIndexes))
	}
	if len(dataRows) > limit {
		fmt.Printf("  ... (%d more non-empty rows)\n", len(dataRows)-limit)
	}
}

// expandMergedCells fills empty cells by querying excelize's GetCellValue,
// which resolves merged-cell values natively without manual range walking.
func expandMergedCells(f *excelize.File, sheet string, rows [][]string) [][]string {
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	for rowIdx, row := range rows {
		for len(row) < maxCols {
			row = append(row, "")
		}
		for colIdx := range row {
			if strings.TrimSpace(row[colIdx]) == "" {
				cellName, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					continue
				}
				val, _ := f.GetCellValue(sheet, cellName)
				row[colIdx] = val
			}
		}
		rows[rowIdx] = row
	}
	return rows
}

// isEmptyRow returns true when every cell in the row is blank.
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// exportCSV exports a sheet to CSV (stdout or file).
func exportCSV(f *excelize.File, sheet string, columnFilter []string, outPath string) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Fatalf("cannot read sheet %q: %v", sheet, err)
	}
	if len(rows) == 0 {
		log.Fatalf("sheet %q is empty", sheet)
	}

	var w *csv.Writer
	if outPath != "" {
		file, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("cannot create CSV file: %v", err)
		}
		defer file.Close()
		w = csv.NewWriter(file)
		fmt.Fprintf(os.Stderr, "Exporting sheet %q to %s...\n", sheet, outPath)
	} else {
		w = csv.NewWriter(os.Stdout)
	}
	defer w.Flush()

	colIndexes := resolveColumnIndexes(rows[0], columnFilter)

	for _, row := range rows {
		if err := w.Write(selectCols(row, colIndexes)); err != nil {
			log.Fatalf("csv write error: %v", err)
		}
	}

	if outPath != "" {
		fmt.Fprintf(os.Stderr, "Done. %d rows written.\n", len(rows))
	}
}

// searchKeyword searches for a keyword across the given sheets.
func searchKeyword(f *excelize.File, sheets []string, keyword string) {
	kw := strings.ToLower(keyword)
	fmt.Printf("Searching for %q...\n\n", keyword)

	totalMatches := 0
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			fmt.Printf("[%s] cannot read: %v\n", sheet, err)
			continue
		}

		var headers []string
		if len(rows) > 0 {
			headers = rows[0]
		}

		for rowIdx, row := range rows {
			for colIdx, cell := range row {
				if strings.Contains(strings.ToLower(cell), kw) {
					colLabel := colName(headers, colIdx)
					fmt.Printf("  Sheet=%q  Row=%d  Col=%s  Value=%q\n",
						sheet, rowIdx+1, colLabel, cell)
					totalMatches++
				}
			}
		}
	}

	if totalMatches == 0 {
		fmt.Println("No matches found.")
	} else {
		fmt.Printf("\nTotal matches: %d\n", totalMatches)
	}
}

// resolveColumnIndexes returns the indices for the requested columns (nil = all).
func resolveColumnIndexes(headers []string, filter []string) []int {
	if len(filter) == 0 {
		return nil // nil means "all columns"
	}
	var indexes []int
	for i, h := range headers {
		for _, f := range filter {
			if strings.EqualFold(strings.TrimSpace(h), f) {
				indexes = append(indexes, i)
				break
			}
		}
	}
	return indexes
}

// selectCols returns only the requested columns from a row. nil indexes = all.
func selectCols(row []string, indexes []int) []string {
	if indexes == nil {
		return row
	}
	result := make([]string, 0, len(indexes))
	for _, i := range indexes {
		if i < len(row) {
			result = append(result, row[i])
		} else {
			result = append(result, "")
		}
	}
	return result
}

// colName returns "HeaderName(idx)" or just the index when headers are missing.
func colName(headers []string, idx int) string {
	if idx < len(headers) && headers[idx] != "" {
		return fmt.Sprintf("%s(%d)", headers[idx], idx+1)
	}
	return fmt.Sprintf("col%d", idx+1)
}
