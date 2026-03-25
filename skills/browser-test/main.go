package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const frontendDir = "tests/frontend"

func main() {
	projectFlag := flag.String("project", "", "Test project: api, e2e, smoke (default: all)")
	fileFlag := flag.String("file", "", "Filter test files by pattern")
	testFlag := flag.String("test", "", "Filter by test name (--grep)")
	headedFlag := flag.Bool("headed", false, "Run browser in headed mode")
	debugFlag := flag.Bool("debug", false, "Run with Playwright inspector (PWDEBUG=1)")
	retriesFlag := flag.Int("retries", -1, "Override retry count")
	baseURLFlag := flag.String("base-url", "", "Override BASE_URL (default: http://localhost:3030)")
	installFlag := flag.Bool("install", false, "Install npm deps + Playwright browsers")
	listFlag := flag.Bool("list-tests", false, "List tests without running")
	reportFlag := flag.Bool("report", false, "Open HTML report after run")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: browser_test.go [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Resolve project root (walk up from cwd to find tests/frontend)
	root := findProjectRoot()
	if root == "" {
		log.Fatal("cannot find project root (tests/frontend directory not found). Run from the project directory.")
	}
	feDir := filepath.Join(root, frontendDir)

	if *installFlag {
		runInstall(feDir)
		return
	}

	// Build playwright args
	args := []string{"playwright", "test"}

	if *projectFlag != "" {
		args = append(args, "--project", *projectFlag)
	}

	if *fileFlag != "" {
		args = append(args, *fileFlag)
	}

	if *testFlag != "" {
		args = append(args, "--grep", *testFlag)
	}

	if *headedFlag {
		args = append(args, "--headed")
	}

	if *retriesFlag >= 0 {
		args = append(args, "--retries", fmt.Sprintf("%d", *retriesFlag))
	}

	if *listFlag {
		args = append(args, "--list")
	}

	if *reportFlag {
		// Run tests then open report
		runPlaywright(feDir, args, *debugFlag, *baseURLFlag)
		openReport(feDir)
		return
	}

	runPlaywright(feDir, args, *debugFlag, *baseURLFlag)
}

func runInstall(feDir string) {
	fmt.Println("Installing npm dependencies...")
	npmCmd := exec.Command("npm", "install")
	npmCmd.Dir = feDir
	npmCmd.Stdout = os.Stdout
	npmCmd.Stderr = os.Stderr
	if err := npmCmd.Run(); err != nil {
		log.Fatalf("npm install failed: %v", err)
	}

	fmt.Println("\nInstalling Playwright browsers (chromium)...")
	pwCmd := exec.Command("npx", "playwright", "install", "chromium")
	pwCmd.Dir = feDir
	pwCmd.Stdout = os.Stdout
	pwCmd.Stderr = os.Stderr
	if err := pwCmd.Run(); err != nil {
		log.Fatalf("playwright install failed: %v", err)
	}
	fmt.Println("\nInstallation complete.")
}

func runPlaywright(feDir string, args []string, debug bool, baseURL string) {
	cmd := exec.Command("npx", args...)
	cmd.Dir = feDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Inherit env, then overlay overrides
	env := os.Environ()
	if debug {
		env = setEnv(env, "PWDEBUG", "1")
	}
	if baseURL != "" {
		env = setEnv(env, "BASE_URL", baseURL)
	}
	cmd.Env = env

	fmt.Printf("Running: npx %s\n", strings.Join(args, " "))
	fmt.Printf("Dir: %s\n\n", feDir)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Playwright exits non-zero on test failures — that's expected
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("playwright error: %v", err)
	}
}

func openReport(feDir string) {
	fmt.Println("\nOpening HTML report...")
	cmd := exec.Command("npx", "playwright", "show-report")
	cmd.Dir = feDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// findProjectRoot walks up from cwd until it finds a directory containing tests/frontend.
func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, frontendDir)
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// setEnv replaces or appends key=value in an env slice.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
