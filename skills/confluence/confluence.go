package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// confluenceConfig holds Confluence connection credentials.
type confluenceConfig struct {
	BaseURL  string `yaml:"base_url" json:"base_url"`
	Email    string `yaml:"email" json:"email"`
	APIToken string `yaml:"api_token" json:"api_token"`
}

func (c confluenceConfig) masked() string {
	token := "***"
	if c.APIToken == "" {
		token = "(not set)"
	}
	return fmt.Sprintf("Base URL: %s\nEmail:    %s\nToken:    %s", c.BaseURL, c.Email, token)
}

func (c confluenceConfig) valid() bool {
	return c.BaseURL != "" && c.Email != "" && c.APIToken != ""
}

// loadConfig loads Confluence config from project, global, or env vars.
func loadConfig() confluenceConfig {
	// 1. Project config
	if cfg, ok := loadConfigFile(projectConfigPath()); ok {
		return cfg
	}
	// 2. Global config
	if cfg, ok := loadConfigFile(globalConfigPath()); ok {
		return cfg
	}
	// 3. Environment variables
	cfg := confluenceConfig{
		BaseURL:  os.Getenv("CONFLUENCE_BASE_URL"),
		Email:    os.Getenv("CONFLUENCE_EMAIL"),
		APIToken: os.Getenv("CONFLUENCE_API_TOKEN"),
	}
	if cfg.valid() {
		return cfg
	}
	return confluenceConfig{}
}

func loadConfigFile(path string) (confluenceConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return confluenceConfig{}, false
	}
	var cfg confluenceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return confluenceConfig{}, false
	}
	if cfg.valid() {
		return cfg, true
	}
	return confluenceConfig{}, false
}

func projectConfigPath() string {
	return filepath.Join(".claude", "confluence-config.yaml")
}

func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "confluence-config.yaml")
}

func saveConfig(cfg confluenceConfig, global bool) error {
	path := projectConfigPath()
	if global {
		path = globalConfigPath()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %v", err)
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("cannot marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("cannot write config: %v", err)
	}
	fmt.Printf("Config saved to %s\n", path)
	return nil
}

// apiRequest performs an authenticated Confluence REST API request.
func apiRequest(cfg confluenceConfig, method, endpoint string, body io.Reader) ([]byte, int, error) {
	u := strings.TrimRight(cfg.BaseURL, "/") + endpoint
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(cfg.Email, cfg.APIToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

// createPage creates a new Confluence page using the v2 API.
func createPage(cfg confluenceConfig, spaceKey, title, bodyContent, parentID string) {
	if spaceKey == "" || title == "" {
		log.Fatalf("--space and --title are required for --create")
	}

	// First, resolve space key to space ID using v1 API.
	spaceData, status, err := apiRequest(cfg, "GET", "/wiki/rest/api/space/"+url.PathEscape(spaceKey), nil)
	if err != nil {
		log.Fatalf("cannot fetch space: %v", err)
	}
	if status != 200 {
		log.Fatalf("cannot fetch space %q (HTTP %d): %s", spaceKey, status, string(spaceData))
	}
	var spaceResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(spaceData, &spaceResp); err != nil {
		log.Fatalf("cannot parse space response: %v", err)
	}

	payload := map[string]interface{}{
		"spaceId": fmt.Sprintf("%d", spaceResp.ID),
		"status":  "current",
		"title":   title,
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          bodyContent,
		},
	}
	if parentID != "" {
		payload["parentId"] = parentID
	}

	jsonData, _ := json.Marshal(payload)
	data, status, err := apiRequest(cfg, "POST", "/wiki/api/v2/pages", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot create page: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("create page failed (HTTP %d): %s", status, string(data))
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	fmt.Printf("Page created successfully.\n")
	fmt.Printf("ID:    %v\n", result["id"])
	fmt.Printf("Title: %v\n", result["title"])
	if link, ok := result["_links"].(map[string]interface{}); ok {
		if webui, ok := link["webui"].(string); ok {
			fmt.Printf("URL:   %s%s\n", strings.TrimRight(cfg.BaseURL, "/"), webui)
		}
	}
}

// getPage retrieves a page by ID using the v2 API.
func getPage(cfg confluenceConfig, pageID, format string) {
	endpoint := fmt.Sprintf("/wiki/api/v2/pages/%s?body-format=storage", url.PathEscape(pageID))
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot get page: %v", err)
	}
	if status != 200 {
		log.Fatalf("get page failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
		Body   struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
		Links map[string]interface{} `json:"_links"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse page: %v", err)
	}

	fmt.Printf("ID:      %s\n", result.ID)
	fmt.Printf("Title:   %s\n", result.Title)
	fmt.Printf("Status:  %s\n", result.Status)
	fmt.Printf("Version: %d\n", result.Version.Number)
	if webui, ok := result.Links["webui"].(string); ok {
		fmt.Printf("URL:     %s%s\n", strings.TrimRight(cfg.BaseURL, "/"), webui)
	}
	fmt.Printf("\n--- Content (storage format) ---\n%s\n", result.Body.Storage.Value)
}

// updatePage updates a page by ID using the v2 API.
func updatePage(cfg confluenceConfig, pageID, title, bodyContent string) {
	// First, get the current page to obtain version number and current values.
	endpoint := fmt.Sprintf("/wiki/api/v2/pages/%s?body-format=storage", url.PathEscape(pageID))
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot get current page: %v", err)
	}
	if status != 200 {
		log.Fatalf("get page failed (HTTP %d): %s", status, string(data))
	}

	var current struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
		Body   struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	if err := json.Unmarshal(data, &current); err != nil {
		log.Fatalf("cannot parse current page: %v", err)
	}

	// Use current values as defaults.
	if title == "" {
		title = current.Title
	}
	if bodyContent == "" {
		bodyContent = current.Body.Storage.Value
	}

	payload := map[string]interface{}{
		"id":     pageID,
		"status": "current",
		"title":  title,
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          bodyContent,
		},
		"version": map[string]interface{}{
			"number": current.Version.Number + 1,
		},
	}

	jsonData, _ := json.Marshal(payload)
	data, status, err = apiRequest(cfg, "PUT", fmt.Sprintf("/wiki/api/v2/pages/%s", url.PathEscape(pageID)), strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot update page: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("update page failed (HTTP %d): %s", status, string(data))
	}

	fmt.Printf("Page %s updated successfully (version %d).\n", pageID, current.Version.Number+1)
}

// deletePage deletes a page by ID.
func deletePage(cfg confluenceConfig, pageID string) {
	data, status, err := apiRequest(cfg, "DELETE", fmt.Sprintf("/wiki/api/v2/pages/%s", url.PathEscape(pageID)), nil)
	if err != nil {
		log.Fatalf("cannot delete page: %v", err)
	}
	if status != 204 && status != 200 {
		log.Fatalf("delete page failed (HTTP %d): %s", status, string(data))
	}
	fmt.Printf("Page %s deleted successfully.\n", pageID)
}

// searchPages searches Confluence using CQL.
func searchPages(cfg confluenceConfig, cql string, maxResults int, format string) {
	endpoint := fmt.Sprintf("/wiki/rest/api/content/search?cql=%s&limit=%d", url.QueryEscape(cql), maxResults)
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot search: %v", err)
	}
	if status != 200 {
		log.Fatalf("search failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Results []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Type  string `json:"type"`
			Links struct {
				WebUI string `json:"webui"`
			} `json:"_links"`
			Space struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"space"`
		} `json:"results"`
		Size int `json:"size"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse search results: %v", err)
	}

	fmt.Printf("Found %d result(s):\n\n", result.Size)
	for i, r := range result.Results {
		fmt.Printf("%d. [%s] %s\n", i+1, r.ID, r.Title)
		if r.Space.Key != "" {
			fmt.Printf("   Space: %s (%s)\n", r.Space.Name, r.Space.Key)
		}
		if r.Links.WebUI != "" {
			fmt.Printf("   URL:   %s%s\n", strings.TrimRight(cfg.BaseURL, "/"), r.Links.WebUI)
		}
		fmt.Println()
	}
}

// listSpaces lists all Confluence spaces.
func listSpaces(cfg confluenceConfig, maxResults int, format string) {
	endpoint := fmt.Sprintf("/wiki/api/v2/spaces?limit=%d", maxResults)
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot list spaces: %v", err)
	}
	if status != 200 {
		log.Fatalf("list spaces failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Results []struct {
			ID   string `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse spaces: %v", err)
	}

	fmt.Printf("Spaces (%d):\n\n", len(result.Results))
	for _, s := range result.Results {
		fmt.Printf("  %-10s  %-12s  %s\n", s.Key, s.Type, s.Name)
	}
}

// listChildren lists child pages of a given page.
func listChildren(cfg confluenceConfig, pageID string, maxResults int, format string) {
	endpoint := fmt.Sprintf("/wiki/api/v2/pages/%s/children/page?limit=%d", url.PathEscape(pageID), maxResults)
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot list children: %v", err)
	}
	if status != 200 {
		log.Fatalf("list children failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Results []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse children: %v", err)
	}

	fmt.Printf("Child pages of %s (%d):\n\n", pageID, len(result.Results))
	for _, c := range result.Results {
		fmt.Printf("  [%s] %s (%s)\n", c.ID, c.Title, c.Status)
	}
}

// addComment adds a comment to a page.
func addComment(cfg confluenceConfig, pageID, bodyContent string) {
	if bodyContent == "" {
		log.Fatalf("--body is required for --comment")
	}

	payload := map[string]interface{}{
		"pageId": pageID,
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          bodyContent,
		},
	}
	jsonData, _ := json.Marshal(payload)
	data, status, err := apiRequest(cfg, "POST", "/wiki/api/v2/footer-comments", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot add comment: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("add comment failed (HTTP %d): %s", status, string(data))
	}

	fmt.Printf("Comment added to page %s.\n", pageID)
}

func main() {
	// Config flags
	setupFlag := flag.Bool("setup", false, "Save Confluence credentials to config")
	showConfigFlag := flag.Bool("show-config", false, "Show current config (masks token)")
	globalFlag := flag.Bool("global", false, "Target global config instead of project")
	baseURLFlag := flag.String("base-url", "", "Confluence base URL")
	emailFlag := flag.String("email", "", "Confluence user email")
	apiTokenFlag := flag.String("api-token", "", "Confluence API token")

	// Operation flags
	createFlag := flag.Bool("create", false, "Create a new page")
	getFlag := flag.String("get", "", "Get a page by ID")
	updateFlag := flag.String("update", "", "Update a page by ID")
	deleteFlag := flag.String("delete", "", "Delete a page by ID")
	searchFlag := flag.String("search", "", "Search pages using CQL")
	commentFlag := flag.String("comment", "", "Add a comment to a page (page ID)")
	childrenFlag := flag.String("children", "", "List child pages of a page (page ID)")
	spacesFlag := flag.Bool("spaces", false, "List all spaces")

	// Content flags
	spaceFlag := flag.String("space", "", "Space key (for --create)")
	titleFlag := flag.String("title", "", "Page title")
	bodyFlag := flag.String("body", "", "Page body (Confluence storage format / HTML)")
	bodyFileFlag := flag.String("body-file", "", "Read page body from a file")
	parentFlag := flag.String("parent", "", "Parent page ID (for --create)")
	formatFlag := flag.String("format", "text", "Output format: text, json")
	rowsFlag := flag.Int("rows", 25, "Max results for search/list")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: confluence.go [flags]\n\nManage Confluence pages from the command line.\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Handle setup
	if *setupFlag {
		cfg := confluenceConfig{
			BaseURL:  *baseURLFlag,
			Email:    *emailFlag,
			APIToken: *apiTokenFlag,
		}
		if !cfg.valid() {
			log.Fatalf("--base-url, --email, and --api-token are all required for --setup")
		}
		if err := saveConfig(cfg, *globalFlag); err != nil {
			log.Fatalf("setup failed: %v", err)
		}
		return
	}

	// Handle show-config
	if *showConfigFlag {
		cfg := loadConfig()
		if !cfg.valid() {
			fmt.Println("No Confluence configuration found.")
			fmt.Println("Run with --setup to configure, or set CONFLUENCE_BASE_URL, CONFLUENCE_EMAIL, CONFLUENCE_API_TOKEN env vars.")
			return
		}
		fmt.Println(cfg.masked())
		return
	}

	// Resolve body content from --body or --body-file
	bodyContent := *bodyFlag
	if *bodyFileFlag != "" {
		data, err := os.ReadFile(*bodyFileFlag)
		if err != nil {
			log.Fatalf("cannot read body file: %v", err)
		}
		bodyContent = string(data)
	}

	// All remaining operations need valid config
	cfg := loadConfig()
	if !cfg.valid() {
		log.Fatalf("Confluence not configured. Run with --setup or set CONFLUENCE_BASE_URL, CONFLUENCE_EMAIL, CONFLUENCE_API_TOKEN env vars.")
	}

	switch {
	case *createFlag:
		createPage(cfg, *spaceFlag, *titleFlag, bodyContent, *parentFlag)
	case *getFlag != "":
		getPage(cfg, *getFlag, *formatFlag)
	case *updateFlag != "":
		updatePage(cfg, *updateFlag, *titleFlag, bodyContent)
	case *deleteFlag != "":
		deletePage(cfg, *deleteFlag)
	case *searchFlag != "":
		searchPages(cfg, *searchFlag, *rowsFlag, *formatFlag)
	case *commentFlag != "":
		addComment(cfg, *commentFlag, bodyContent)
	case *childrenFlag != "":
		listChildren(cfg, *childrenFlag, *rowsFlag, *formatFlag)
	case *spacesFlag:
		listSpaces(cfg, *rowsFlag, *formatFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
