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

// jiraConfig holds Jira connection credentials.
type jiraConfig struct {
	BaseURL  string `yaml:"base_url" json:"base_url"`
	Email    string `yaml:"email" json:"email"`
	APIToken string `yaml:"api_token" json:"api_token"`
}

func (c jiraConfig) masked() string {
	token := "***"
	if c.APIToken == "" {
		token = "(not set)"
	}
	return fmt.Sprintf("Base URL: %s\nEmail:    %s\nToken:    %s", c.BaseURL, c.Email, token)
}

func (c jiraConfig) valid() bool {
	return c.BaseURL != "" && c.Email != "" && c.APIToken != ""
}

// loadConfig loads Jira config from project, global, or env vars.
func loadConfig() jiraConfig {
	// 1. Project config
	if cfg, ok := loadConfigFile(projectConfigPath()); ok {
		return cfg
	}
	// 2. Global config
	if cfg, ok := loadConfigFile(globalConfigPath()); ok {
		return cfg
	}
	// 3. Environment variables
	cfg := jiraConfig{
		BaseURL:  os.Getenv("JIRA_BASE_URL"),
		Email:    os.Getenv("JIRA_EMAIL"),
		APIToken: os.Getenv("JIRA_API_TOKEN"),
	}
	if cfg.valid() {
		return cfg
	}
	return jiraConfig{}
}

func loadConfigFile(path string) (jiraConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return jiraConfig{}, false
	}
	var cfg jiraConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return jiraConfig{}, false
	}
	if cfg.valid() {
		return cfg, true
	}
	return jiraConfig{}, false
}

func projectConfigPath() string {
	return filepath.Join(".claude", "jira-config.yaml")
}

func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "jira-config.yaml")
}

func saveConfig(cfg jiraConfig, global bool) error {
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

// apiRequest performs an authenticated Jira REST API request.
func apiRequest(cfg jiraConfig, method, endpoint string, body io.Reader) ([]byte, int, error) {
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

// createIssue creates a new Jira issue.
func createIssue(cfg jiraConfig, project, issueType, summary, description, priority, assignee, labels, parent string) {
	if project == "" || issueType == "" || summary == "" {
		log.Fatalf("--project, --type, and --summary are required for --create")
	}

	fields := map[string]interface{}{
		"project":   map[string]string{"key": project},
		"issuetype": map[string]string{"name": issueType},
		"summary":   summary,
	}
	if description != "" {
		fields["description"] = map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": description,
						},
					},
				},
			},
		}
	}
	if priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if assignee != "" {
		fields["assignee"] = map[string]string{"id": resolveAccountID(cfg, assignee)}
	}
	if labels != "" {
		fields["labels"] = strings.Split(labels, ",")
	}
	if parent != "" {
		fields["parent"] = map[string]string{"key": parent}
	}

	payload := map[string]interface{}{"fields": fields}
	jsonData, _ := json.Marshal(payload)
	data, status, err := apiRequest(cfg, "POST", "/rest/api/3/issue", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot create issue: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("create issue failed (HTTP %d): %s", status, string(data))
	}

	var result struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}
	json.Unmarshal(data, &result)
	fmt.Printf("Issue created successfully.\n")
	fmt.Printf("Key:  %s\n", result.Key)
	fmt.Printf("ID:   %s\n", result.ID)
	fmt.Printf("URL:  %s/browse/%s\n", strings.TrimRight(cfg.BaseURL, "/"), result.Key)
}

// resolveAccountID resolves an email to a Jira account ID, or returns the input if it looks like an account ID already.
func resolveAccountID(cfg jiraConfig, emailOrID string) string {
	if !strings.Contains(emailOrID, "@") {
		return emailOrID
	}
	endpoint := fmt.Sprintf("/rest/api/3/user/search?query=%s", url.QueryEscape(emailOrID))
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot search for user: %v", err)
	}
	if status != 200 {
		log.Fatalf("user search failed (HTTP %d): %s", status, string(data))
	}
	var users []struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(data, &users); err != nil || len(users) == 0 {
		log.Fatalf("no user found for %q", emailOrID)
	}
	return users[0].AccountID
}

// getIssue retrieves an issue by key.
func getIssue(cfg jiraConfig, issueKey, format string) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(issueKey))
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot get issue: %v", err)
	}
	if status != 200 {
		log.Fatalf("get issue failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Status      struct{ Name string } `json:"status"`
			Priority    struct{ Name string } `json:"priority"`
			IssueType   struct{ Name string } `json:"issuetype"`
			Assignee    struct{ DisplayName string; EmailAddress string } `json:"assignee"`
			Reporter    struct{ DisplayName string } `json:"reporter"`
			Labels      []string `json:"labels"`
			Created     string `json:"created"`
			Updated     string `json:"updated"`
			Description json.RawMessage `json:"description"`
			Parent      struct{ Key string } `json:"parent"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse issue: %v", err)
	}

	f := result.Fields
	fmt.Printf("Key:         %s\n", result.Key)
	fmt.Printf("Summary:     %s\n", f.Summary)
	fmt.Printf("Type:        %s\n", f.IssueType.Name)
	fmt.Printf("Status:      %s\n", f.Status.Name)
	fmt.Printf("Priority:    %s\n", f.Priority.Name)
	if f.Assignee.DisplayName != "" {
		fmt.Printf("Assignee:    %s (%s)\n", f.Assignee.DisplayName, f.Assignee.EmailAddress)
	} else {
		fmt.Printf("Assignee:    Unassigned\n")
	}
	fmt.Printf("Reporter:    %s\n", f.Reporter.DisplayName)
	if len(f.Labels) > 0 {
		fmt.Printf("Labels:      %s\n", strings.Join(f.Labels, ", "))
	}
	if f.Parent.Key != "" {
		fmt.Printf("Parent:      %s\n", f.Parent.Key)
	}
	fmt.Printf("Created:     %s\n", f.Created)
	fmt.Printf("Updated:     %s\n", f.Updated)
	fmt.Printf("URL:         %s/browse/%s\n", strings.TrimRight(cfg.BaseURL, "/"), result.Key)

	if f.Description != nil && string(f.Description) != "null" {
		desc := extractADFText(f.Description)
		if desc != "" {
			fmt.Printf("\n--- Description ---\n%s\n", desc)
		}
	}
}

// extractADFText extracts plain text from Atlassian Document Format JSON.
func extractADFText(raw json.RawMessage) string {
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	var sb strings.Builder
	extractTextNodes(&sb, doc)
	return strings.TrimSpace(sb.String())
}

func extractTextNodes(sb *strings.Builder, node map[string]interface{}) {
	if t, ok := node["type"].(string); ok && t == "text" {
		if text, ok := node["text"].(string); ok {
			sb.WriteString(text)
		}
	}
	if t, ok := node["type"].(string); ok && (t == "paragraph" || t == "heading") {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
	}
	if content, ok := node["content"].([]interface{}); ok {
		for _, c := range content {
			if m, ok := c.(map[string]interface{}); ok {
				extractTextNodes(sb, m)
			}
		}
	}
}

// updateIssue updates an issue by key.
func updateIssue(cfg jiraConfig, issueKey, summary, description, priority, assignee, labels, status string) {
	fields := map[string]interface{}{}
	if summary != "" {
		fields["summary"] = summary
	}
	if description != "" {
		fields["description"] = map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": description,
						},
					},
				},
			},
		}
	}
	if priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if assignee != "" {
		fields["assignee"] = map[string]string{"id": resolveAccountID(cfg, assignee)}
	}
	if labels != "" {
		fields["labels"] = strings.Split(labels, ",")
	}

	if len(fields) == 0 && status == "" {
		log.Fatalf("no fields to update; provide at least one of --summary, --description, --priority, --assignee, --labels, or --status")
	}

	// If status is provided along with other fields, update fields first, then transition.
	if len(fields) > 0 {
		payload := map[string]interface{}{"fields": fields}
		jsonData, _ := json.Marshal(payload)
		data, code, err := apiRequest(cfg, "PUT", fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(issueKey)), strings.NewReader(string(jsonData)))
		if err != nil {
			log.Fatalf("cannot update issue: %v", err)
		}
		if code < 200 || code >= 300 {
			log.Fatalf("update issue failed (HTTP %d): %s", code, string(data))
		}
		fmt.Printf("Issue %s updated successfully.\n", issueKey)
	}

	if status != "" {
		transitionIssue(cfg, issueKey, status)
	}
}

// transitionIssue transitions an issue to a new status.
func transitionIssue(cfg jiraConfig, issueKey, targetStatus string) {
	if targetStatus == "" {
		log.Fatalf("--status is required for --transition")
	}

	// Get available transitions
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(issueKey))
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot get transitions: %v", err)
	}
	if status != 200 {
		log.Fatalf("get transitions failed (HTTP %d): %s", status, string(data))
	}

	var result struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse transitions: %v", err)
	}

	// Find the matching transition (match on transition name or target status name, case-insensitive)
	var transitionID string
	lower := strings.ToLower(targetStatus)
	for _, t := range result.Transitions {
		if strings.ToLower(t.Name) == lower || strings.ToLower(t.To.Name) == lower {
			transitionID = t.ID
			break
		}
	}
	if transitionID == "" {
		var available []string
		for _, t := range result.Transitions {
			available = append(available, fmt.Sprintf("%s (-> %s)", t.Name, t.To.Name))
		}
		log.Fatalf("no transition found for status %q; available transitions: %s", targetStatus, strings.Join(available, ", "))
	}

	payload := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}
	jsonData, _ := json.Marshal(payload)
	data, status2, err := apiRequest(cfg, "POST", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot transition issue: %v", err)
	}
	if status2 < 200 || status2 >= 300 {
		log.Fatalf("transition failed (HTTP %d): %s", status2, string(data))
	}
	fmt.Printf("Issue %s transitioned to %q.\n", issueKey, targetStatus)
}

// addComment adds a comment to an issue.
func addComment(cfg jiraConfig, issueKey, body string) {
	if body == "" {
		log.Fatalf("--body is required for --comment")
	}

	payload := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{
				map[string]interface{}{
					"type": "paragraph",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": body,
						},
					},
				},
			},
		},
	}
	jsonData, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(issueKey))
	data, status, err := apiRequest(cfg, "POST", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot add comment: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("add comment failed (HTTP %d): %s", status, string(data))
	}
	fmt.Printf("Comment added to %s.\n", issueKey)
}

// searchIssues searches Jira using JQL.
func searchIssues(cfg jiraConfig, jql string, maxResults int, format string) {
	endpoint := fmt.Sprintf("/rest/api/3/search?jql=%s&maxResults=%d", url.QueryEscape(jql), maxResults)
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
		Total  int `json:"total"`
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary   string `json:"summary"`
				Status    struct{ Name string } `json:"status"`
				Priority  struct{ Name string } `json:"priority"`
				IssueType struct{ Name string } `json:"issuetype"`
				Assignee  struct{ DisplayName string } `json:"assignee"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse search results: %v", err)
	}

	fmt.Printf("Found %d issue(s):\n\n", result.Total)
	for i, issue := range result.Issues {
		f := issue.Fields
		assignee := "Unassigned"
		if f.Assignee.DisplayName != "" {
			assignee = f.Assignee.DisplayName
		}
		fmt.Printf("%d. [%s] %s\n", i+1, issue.Key, f.Summary)
		fmt.Printf("   Type: %s  Status: %s  Priority: %s  Assignee: %s\n", f.IssueType.Name, f.Status.Name, f.Priority.Name, assignee)
		fmt.Printf("   URL:  %s/browse/%s\n", strings.TrimRight(cfg.BaseURL, "/"), issue.Key)
		fmt.Println()
	}
}

// assignIssue assigns an issue to a user.
func assignIssue(cfg jiraConfig, issueKey, assignee string) {
	if assignee == "" {
		log.Fatalf("--assignee is required for --assign")
	}
	accountID := resolveAccountID(cfg, assignee)
	payload := map[string]string{"accountId": accountID}
	jsonData, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/assignee", url.PathEscape(issueKey))
	data, status, err := apiRequest(cfg, "PUT", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot assign issue: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("assign issue failed (HTTP %d): %s", status, string(data))
	}
	fmt.Printf("Issue %s assigned to %s.\n", issueKey, assignee)
}

// listProjects lists all Jira projects.
func listProjects(cfg jiraConfig, maxResults int, format string) {
	endpoint := fmt.Sprintf("/rest/api/3/project/search?maxResults=%d", maxResults)
	data, status, err := apiRequest(cfg, "GET", endpoint, nil)
	if err != nil {
		log.Fatalf("cannot list projects: %v", err)
	}
	if status != 200 {
		log.Fatalf("list projects failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Values []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
			Style string `json:"style"`
			Lead struct {
				DisplayName string `json:"displayName"`
			} `json:"lead"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse projects: %v", err)
	}

	fmt.Printf("Projects (%d):\n\n", len(result.Values))
	for _, p := range result.Values {
		lead := ""
		if p.Lead.DisplayName != "" {
			lead = fmt.Sprintf("  Lead: %s", p.Lead.DisplayName)
		}
		fmt.Printf("  %-10s  %s%s\n", p.Key, p.Name, lead)
	}
}

func main() {
	// Config flags
	setupFlag := flag.Bool("setup", false, "Save Jira credentials to config")
	showConfigFlag := flag.Bool("show-config", false, "Show current config (masks token)")
	globalFlag := flag.Bool("global", false, "Target global config instead of project")
	baseURLFlag := flag.String("base-url", "", "Jira base URL")
	emailFlag := flag.String("email", "", "Jira user email")
	apiTokenFlag := flag.String("api-token", "", "Jira API token")

	// Operation flags
	createFlag := flag.Bool("create", false, "Create a new issue")
	getFlag := flag.String("get", "", "Get an issue by key")
	updateFlag := flag.String("update", "", "Update an issue by key")
	transitionFlag := flag.String("transition", "", "Transition an issue status (issue key)")
	commentFlag := flag.String("comment", "", "Add a comment to an issue (issue key)")
	searchFlag := flag.String("search", "", "Search issues using JQL")
	assignFlag := flag.String("assign", "", "Assign an issue (issue key)")
	projectsFlag := flag.Bool("projects", false, "List all projects")

	// Content flags
	projectFlag := flag.String("project", "", "Project key (for --create)")
	typeFlag := flag.String("type", "", "Issue type (for --create)")
	summaryFlag := flag.String("summary", "", "Issue summary")
	descriptionFlag := flag.String("description", "", "Issue description")
	priorityFlag := flag.String("priority", "", "Priority name")
	assigneeFlag := flag.String("assignee", "", "Assignee email or account ID")
	labelsFlag := flag.String("labels", "", "Comma-separated labels")
	parentFlag := flag.String("parent", "", "Parent issue key (for subtasks/child issues)")
	statusFlag := flag.String("status", "", "Target status (for --transition or --update)")
	bodyFlag := flag.String("body", "", "Comment body (for --comment)")
	formatFlag := flag.String("format", "text", "Output format: text, json")
	rowsFlag := flag.Int("rows", 25, "Max results for search/list")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jira.go [flags]\n\nManage Jira issues from the command line.\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Handle setup
	if *setupFlag {
		cfg := jiraConfig{
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
			fmt.Println("No Jira configuration found.")
			fmt.Println("Run with --setup to configure, or set JIRA_BASE_URL, JIRA_EMAIL, JIRA_API_TOKEN env vars.")
			return
		}
		fmt.Println(cfg.masked())
		return
	}

	// All remaining operations need valid config
	cfg := loadConfig()
	if !cfg.valid() {
		log.Fatalf("Jira not configured. Run with --setup or set JIRA_BASE_URL, JIRA_EMAIL, JIRA_API_TOKEN env vars.")
	}

	switch {
	case *createFlag:
		createIssue(cfg, *projectFlag, *typeFlag, *summaryFlag, *descriptionFlag, *priorityFlag, *assigneeFlag, *labelsFlag, *parentFlag)
	case *getFlag != "":
		getIssue(cfg, *getFlag, *formatFlag)
	case *updateFlag != "":
		updateIssue(cfg, *updateFlag, *summaryFlag, *descriptionFlag, *priorityFlag, *assigneeFlag, *labelsFlag, *statusFlag)
	case *transitionFlag != "":
		transitionIssue(cfg, *transitionFlag, *statusFlag)
	case *commentFlag != "":
		addComment(cfg, *commentFlag, *bodyFlag)
	case *searchFlag != "":
		searchIssues(cfg, *searchFlag, *rowsFlag, *formatFlag)
	case *assignFlag != "":
		assignIssue(cfg, *assignFlag, *assigneeFlag)
	case *projectsFlag:
		listProjects(cfg, *rowsFlag, *formatFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
