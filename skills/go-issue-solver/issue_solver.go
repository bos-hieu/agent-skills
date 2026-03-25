package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

func main() {
	issueFlag := flag.String("issue", "", "Issue description, error message, or log snippet")
	fileFlag := flag.String("file", "", "Specific file or directory to analyze")
	searchFlag := flag.String("search", "", "Search the codebase for a term (shows file:line)")
	testFlag := flag.String("test", "", "Run go test for a package pattern and show failures")
	buildFlag := flag.Bool("build", false, "Run go build ./... and show errors")
	vetFlag := flag.Bool("vet", false, "Run go vet ./... and show warnings")
	scaffoldFlag := flag.String("scaffold", "", "Scaffold type: handler, service, repo, migration, task")
	nameFlag := flag.String("name", "", "Name for scaffolded item (used with --scaffold)")
	callersFlag := flag.String("callers", "", "Find all call sites of a function name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: issue_solver.go [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := findRoot()

	switch {
	case *issueFlag != "":
		analyzeIssue(root, *issueFlag, *fileFlag)
	case *searchFlag != "":
		searchCodebase(root, *searchFlag)
	case *callersFlag != "":
		findCallers(root, *callersFlag)
	case *testFlag != "":
		runTests(root, *testFlag)
	case *buildFlag:
		runBuild(root)
	case *vetFlag:
		runVet(root)
	case *scaffoldFlag != "":
		if *nameFlag == "" {
			log.Fatal("--scaffold requires --name")
		}
		if strings.ContainsAny(*nameFlag, "/\\") || strings.Contains(*nameFlag, "..") {
			log.Fatal("--name must not contain path separators or '..'")
		}
		scaffold(root, *scaffoldFlag, *nameFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}

// ── Issue analysis ────────────────────────────────────────────────────────────

func analyzeIssue(root, issue, specificFile string) {
	fmt.Printf("=== Issue Analysis ===\n")
	fmt.Printf("Input: %q\n\n", issue)

	// Extract search terms from the issue text
	terms := extractTerms(issue)
	fmt.Printf("Search terms: %v\n\n", terms)

	type match struct {
		file string
		line int
		text string
	}
	var matches []match
	seenFiles := map[string]bool{}

	for _, term := range terms {
		files := grepFiles(root, term)
		for _, f := range files {
			if specificFile != "" && !strings.Contains(f.file, specificFile) {
				continue
			}
			if !seenFiles[f.file] {
				seenFiles[f.file] = true
			}
			matches = append(matches, match{f.file, f.line, f.text})
		}
	}

	// Sort by file then line
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].file != matches[j].file {
			return matches[i].file < matches[j].file
		}
		return matches[i].line < matches[j].line
	})

	if len(matches) == 0 {
		fmt.Println("No matching code found for the extracted terms.")
	} else {
		fmt.Printf("Matched locations (%d):\n", len(matches))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		shown := map[string]bool{}
		for _, m := range matches {
			key := fmt.Sprintf("%s:%d", m.file, m.line)
			if shown[key] {
				continue
			}
			shown[key] = true
			rel, _ := filepath.Rel(root, m.file)
			fmt.Fprintf(w, "  %s:%d\t%s\n", rel, m.line, strings.TrimSpace(m.text))
		}
		w.Flush()
	}

	fmt.Printf("\nAffected files:\n")
	sortedFiles := make([]string, 0, len(seenFiles))
	for f := range seenFiles {
		sortedFiles = append(sortedFiles, f)
	}
	sort.Strings(sortedFiles)
	for _, f := range sortedFiles {
		rel, _ := filepath.Rel(root, f)
		fmt.Printf("  %s\n", rel)
	}

	// Suggest packages to test
	pkgs := affectedPackages(root, sortedFiles)
	if len(pkgs) > 0 {
		fmt.Printf("\nSuggested test command:\n")
		fmt.Printf("  go test -v -run . %s\n", strings.Join(pkgs, " "))
	}

	fmt.Println("\n--- Next steps ---")
	fmt.Println("1. Review the matched files above")
	fmt.Println("2. Use --search <term> to dig deeper into specific symbols")
	fmt.Println("3. Use --callers <func> to trace call chains")
	fmt.Println("4. Use --test <pkg> to run targeted tests")
}

// extractTerms picks meaningful tokens from the issue text.
func extractTerms(issue string) []string {
	// Split on common delimiters
	r := strings.NewReplacer(":", " ", "/", " ", ".", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", "\"", " ", "'", " ", "\n", " ", "\t", " ")
	tokens := strings.Fields(r.Replace(issue))

	seen := map[string]bool{}
	var terms []string
	skip := map[string]bool{
		"the": true, "a": true, "an": true, "in": true, "at": true, "of": true,
		"to": true, "is": true, "on": true, "for": true, "with": true, "and": true,
		"error": true, "ERROR": true, "panic": true, "nil": true, "go": true,
	}

	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if len(t) < 3 || skip[strings.ToLower(t)] || seen[t] {
			continue
		}
		// Prefer CamelCase, snake_case, or file-like tokens
		if strings.ContainsAny(t, "_") || (t[0] >= 'A' && t[0] <= 'Z') || strings.Contains(t, "go") {
			terms = append(terms, t)
			seen[t] = true
		}
	}

	// Fallback: take the longest tokens
	if len(terms) == 0 {
		sort.Slice(tokens, func(i, j int) bool { return len(tokens[i]) > len(tokens[j]) })
		for _, t := range tokens {
			if len(t) >= 4 && !skip[strings.ToLower(t)] && !seen[t] {
				terms = append(terms, t)
				seen[t] = true
				if len(terms) >= 5 {
					break
				}
			}
		}
	}

	if len(terms) > 8 {
		terms = terms[:8]
	}
	return terms
}

type grepResult struct {
	file string
	line int
	text string
}

func grepFiles(root, term string) []grepResult {
	cmd := exec.Command("grep", "-rn", "--include=*.go", "-F", term, root)
	out, _ := cmd.Output()
	var results []grepResult
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		// format: /path/to/file.go:42:  code here
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		var lineNum int
		fmt.Sscanf(parts[1], "%d", &lineNum)
		results = append(results, grepResult{parts[0], lineNum, parts[2]})
	}
	return results
}

func affectedPackages(root string, files []string) []string {
	pkgSet := map[string]bool{}
	for _, f := range files {
		if !strings.HasSuffix(f, ".go") || strings.HasSuffix(f, "_test.go") {
			continue
		}
		dir := filepath.Dir(f)
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			continue
		}
		pkgSet["./"+rel+"/..."] = true
	}
	var pkgs []string
	for p := range pkgSet {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	return pkgs
}

// ── Search ────────────────────────────────────────────────────────────────────

func searchCodebase(root, term string) {
	fmt.Printf("Searching for %q...\n\n", term)
	results := grepFiles(root, term)
	if len(results) == 0 {
		fmt.Println("No matches found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, r := range results {
		rel, _ := filepath.Rel(root, r.file)
		fmt.Fprintf(w, "  %s:%d\t%s\n", rel, r.line, strings.TrimSpace(r.text))
	}
	w.Flush()
	fmt.Printf("\n%d match(es)\n", len(results))
}

func findCallers(root, funcName string) {
	fmt.Printf("Finding callers of %q...\n\n", funcName)
	// Search for funcName( to find call sites
	results := grepFiles(root, funcName+"(")
	if len(results) == 0 {
		fmt.Println("No call sites found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, r := range results {
		rel, _ := filepath.Rel(root, r.file)
		fmt.Fprintf(w, "  %s:%d\t%s\n", rel, r.line, strings.TrimSpace(r.text))
	}
	w.Flush()
	fmt.Printf("\n%d call site(s)\n", len(results))
}

// ── Build / test / vet ────────────────────────────────────────────────────────

func runBuild(root string) {
	fmt.Println("Running: go build ./...")
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nbuild FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Build OK")
}

func runVet(root string) {
	fmt.Println("Running: go vet ./...")
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nvet reported issues\n")
		os.Exit(1)
	}
	fmt.Println("Vet OK")
}

func runTests(root, pkg string) {
	fmt.Printf("Running: go test -v %s\n\n", pkg)
	cmd := exec.Command("go", "test", "-v", pkg)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nTests FAILED\n")
		os.Exit(1)
	}
	fmt.Println("\nAll tests passed.")
}

// ── Scaffolding ───────────────────────────────────────────────────────────────

func scaffold(root, kind, name string) {
	switch kind {
	case "handler":
		scaffoldHandler(root, name)
	case "service":
		scaffoldService(root, name)
	case "repo":
		scaffoldRepo(root, name)
	case "migration":
		scaffoldMigration(root, name)
	case "task":
		scaffoldTask(root, name)
	default:
		log.Fatalf("unknown scaffold type %q. Options: handler, service, repo, migration, task", kind)
	}
}

func writeScaffold(path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		log.Fatalf("file already exists: %s", path)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		log.Fatalf("write: %v", err)
	}
	fmt.Printf("Created: %s\n", path)
}

func toCamel(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			b.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	return b.String()
}

func scaffoldHandler(root, name string) {
	camel := toCamel(name)
	path := filepath.Join(root, "internal/handlers/web", name+".go")
	writeScaffold(path, fmt.Sprintf(`package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// %sHandler handles HTTP requests for %s.
type %sHandler struct{}

// New%sHandler creates a new %sHandler.
func New%sHandler() *%sHandler {
	return &%sHandler{}
}

// Register registers routes on the given router group.
func (h *%sHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/%s")
	{
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.POST("", h.Create)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}

func (h *%sHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}})
}

func (h *%sHandler) Get(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *%sHandler) Create(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{})
}

func (h *%sHandler) Update(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *%sHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}
`, camel, name, camel, camel, camel, camel, camel, camel, camel, name,
		camel, camel, camel, camel, camel))
}

func scaffoldService(root, name string) {
	camel := toCamel(name)
	path := filepath.Join(root, "internal/services/web", name+"_service.go")
	writeScaffold(path, fmt.Sprintf(`package web

import "context"

// %sService defines the business logic interface for %s.
type %sService interface {
	List(ctx context.Context) ([]any, error)
	GetByID(ctx context.Context, id string) (any, error)
	Create(ctx context.Context, input any) (any, error)
	Update(ctx context.Context, id string, input any) (any, error)
	Delete(ctx context.Context, id string) error
}

type %sServiceImpl struct{}

// New%sService creates a new %sServiceImpl.
func New%sService() %sService {
	return &%sServiceImpl{}
}

func (s *%sServiceImpl) List(_ context.Context) ([]any, error)                   { return nil, nil }
func (s *%sServiceImpl) GetByID(_ context.Context, _ string) (any, error)        { return nil, nil }
func (s *%sServiceImpl) Create(_ context.Context, _ any) (any, error)            { return nil, nil }
func (s *%sServiceImpl) Update(_ context.Context, _ string, _ any) (any, error)  { return nil, nil }
func (s *%sServiceImpl) Delete(_ context.Context, _ string) error                { return nil }
`, camel, name, camel, camel, camel, camel, camel, camel, camel, camel, camel, camel, camel, camel))
}

func scaffoldRepo(root, name string) {
	camel := toCamel(name)
	path := filepath.Join(root, "internal/repositories", name+"_repository.go")
	writeScaffold(path, fmt.Sprintf(`package repositories

import "gorm.io/gorm"

// %sRepository defines database access for %s.
type %sRepository interface {
	FindAll() ([]any, error)
	FindByID(id string) (any, error)
	Create(entity any) error
	Update(entity any) error
	Delete(id string) error
}

type %sRepositoryImpl struct {
	db *gorm.DB
}

// New%sRepository creates a new %sRepositoryImpl.
func New%sRepository(db *gorm.DB) %sRepository {
	return &%sRepositoryImpl{db: db}
}

func (r *%sRepositoryImpl) FindAll() ([]any, error)          { return nil, nil }
func (r *%sRepositoryImpl) FindByID(_ string) (any, error)   { return nil, nil }
func (r *%sRepositoryImpl) Create(_ any) error               { return nil }
func (r *%sRepositoryImpl) Update(_ any) error               { return nil }
func (r *%sRepositoryImpl) Delete(_ string) error            { return nil }
`, camel, name, camel, camel, camel, camel, camel, camel, camel, camel, camel, camel, camel))
}

func scaffoldMigration(root, name string) {
	ts := time.Now().Format("20060102150405")
	path := filepath.Join(root, "migrations", fmt.Sprintf("%s_%s.up.sql", ts, name))
	writeScaffold(path, fmt.Sprintf(`-- Migration: %s
-- Created: %s

-- TODO: add your SQL here
-- Example:
-- ALTER TABLE foo ADD COLUMN bar TEXT;
-- CREATE INDEX idx_foo_bar ON foo(bar);
`, name, time.Now().Format("2006-01-02 15:04:05")))
}

func scaffoldTask(root, name string) {
	camel := toCamel(name)
	path := filepath.Join(root, "internal/tasks", name+"_task.go")
	writeScaffold(path, fmt.Sprintf(`package tasks

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

const Type%s = "task:%s"

type %sPayload struct {
	// TODO: add payload fields
}

// New%sTask creates an Asynq task for %s.
func New%sTask(payload %sPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(Type%s, data), nil
}

// Handle%s processes the %s task.
func Handle%s(_ context.Context, t *asynq.Task) error {
	var p %sPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	// TODO: implement task logic
	return nil
}
`, camel, name, camel, camel, name, camel, camel, camel, camel, name, camel, camel))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func findRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	log.Fatal("cannot find go.mod — run from within the project")
	return ""
}

// Ensure bufio is used (for future interactive prompts)
var _ = bufio.NewScanner
