package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// googleConfig holds Google credentials shared across google-docs, google-sheets, google-drive.
type googleConfig struct {
	AuthMethod         string `yaml:"auth_method" json:"auth_method"`
	AccessToken        string `yaml:"access_token,omitempty" json:"access_token,omitempty"`
	ServiceAccountFile string `yaml:"service_account_file,omitempty" json:"service_account_file,omitempty"`
}

func (c googleConfig) masked() string {
	method := c.AuthMethod
	if method == "" {
		method = "(not set)"
	}
	token := "***"
	if c.AccessToken == "" {
		token = "(not set)"
	}
	saFile := c.ServiceAccountFile
	if saFile == "" {
		saFile = "(not set)"
	}
	return fmt.Sprintf("Auth Method:          %s\nAccess Token:         %s\nService Account File: %s", method, token, saFile)
}

func (c googleConfig) valid() bool {
	switch c.AuthMethod {
	case "access_token":
		return c.AccessToken != ""
	case "service_account":
		return c.ServiceAccountFile != ""
	}
	return false
}

// loadConfig loads Google config from project, global, or env vars.
func loadConfig() googleConfig {
	// 1. Project config
	if cfg, ok := loadConfigFile(projectConfigPath()); ok {
		return cfg
	}
	// 2. Global config
	if cfg, ok := loadConfigFile(globalConfigPath()); ok {
		return cfg
	}
	// 3. Environment variables
	if tok := os.Getenv("GOOGLE_ACCESS_TOKEN"); tok != "" {
		return googleConfig{AuthMethod: "access_token", AccessToken: tok}
	}
	if sa := os.Getenv("GOOGLE_SERVICE_ACCOUNT_FILE"); sa != "" {
		return googleConfig{AuthMethod: "service_account", ServiceAccountFile: sa}
	}
	return googleConfig{}
}

func loadConfigFile(path string) (googleConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return googleConfig{}, false
	}
	var cfg googleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return googleConfig{}, false
	}
	if cfg.valid() {
		return cfg, true
	}
	return googleConfig{}, false
}

func projectConfigPath() string {
	return filepath.Join(".claude", "google-config.yaml")
}

func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "google-config.yaml")
}

func saveConfig(cfg googleConfig, global bool) error {
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

// resolveAccessToken returns a valid access token, handling service account JWT exchange if needed.
func resolveAccessToken(cfg googleConfig) string {
	switch cfg.AuthMethod {
	case "access_token":
		return cfg.AccessToken
	case "service_account":
		return getServiceAccountToken(cfg.ServiceAccountFile)
	default:
		log.Fatalf("unknown auth_method: %s", cfg.AuthMethod)
		return ""
	}
}

// serviceAccountKey holds the fields we need from a service account JSON key file.
type serviceAccountKey struct {
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	TokenURI     string `json:"token_uri"`
	PrivateKeyID string `json:"private_key_id"`
}

// getServiceAccountToken exchanges a service account key for an access token via JWT.
func getServiceAccountToken(saFile string) string {
	data, err := os.ReadFile(saFile)
	if err != nil {
		log.Fatalf("cannot read service account file: %v", err)
	}
	var sa serviceAccountKey
	if err := json.Unmarshal(data, &sa); err != nil {
		log.Fatalf("cannot parse service account file: %v", err)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}

	// Build JWT
	now := time.Now().Unix()
	header := base64URLEncode(mustJSON(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": sa.PrivateKeyID,
	}))
	claims := base64URLEncode(mustJSON(map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/documents https://www.googleapis.com/auth/drive",
		"aud":   sa.TokenURI,
		"iat":   now,
		"exp":   now + 3600,
	}))
	signingInput := header + "." + claims

	// Parse private key
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		log.Fatalf("cannot decode PEM from service account private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("cannot parse private key: %v", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		log.Fatalf("private key is not RSA")
	}

	// Sign
	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
	if err != nil {
		log.Fatalf("cannot sign JWT: %v", err)
	}
	jwt := signingInput + "." + base64URLEncode(sig)

	// Exchange JWT for access token
	resp, err := http.PostForm(sa.TokenURI, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	})
	if err != nil {
		log.Fatalf("cannot request token: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Fatalf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		log.Fatalf("cannot parse token response: %v", err)
	}
	if tokenResp.AccessToken == "" {
		log.Fatalf("empty access token in response: %s", string(body))
	}
	return tokenResp.AccessToken
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("cannot marshal JSON: %v", err)
	}
	return b
}

// apiRequest performs an authenticated Google API request.
func apiRequest(token, method, rawURL string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
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

// createDocument creates a new Google Doc with the given title.
func createDocument(token, title, format string) {
	if title == "" {
		log.Fatalf("--title is required for --create")
	}
	payload := map[string]string{"title": title}
	jsonData, _ := json.Marshal(payload)

	data, status, err := apiRequest(token, "POST", "https://docs.googleapis.com/v1/documents", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot create document: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("create document failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		DocumentID string `json:"documentId"`
		Title      string `json:"title"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse response: %v", err)
	}

	fmt.Printf("Document created successfully.\n")
	fmt.Printf("ID:    %s\n", result.DocumentID)
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("URL:   https://docs.google.com/document/d/%s/edit\n", result.DocumentID)
}

// getDocument retrieves a Google Doc and prints its content.
func getDocument(token, docID, format string) {
	u := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s", url.PathEscape(docID))
	data, status, err := apiRequest(token, "GET", u, nil)
	if err != nil {
		log.Fatalf("cannot get document: %v", err)
	}
	if status != 200 {
		log.Fatalf("get document failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var doc struct {
		DocumentID string `json:"documentId"`
		Title      string `json:"title"`
		Body       struct {
			Content []struct {
				Paragraph *struct {
					Elements []struct {
						TextRun *struct {
							Content string `json:"content"`
						} `json:"textRun"`
					} `json:"elements"`
				} `json:"paragraph"`
			} `json:"content"`
		} `json:"body"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		log.Fatalf("cannot parse document: %v", err)
	}

	fmt.Printf("ID:    %s\n", doc.DocumentID)
	fmt.Printf("Title: %s\n", doc.Title)
	fmt.Printf("URL:   https://docs.google.com/document/d/%s/edit\n", doc.DocumentID)
	fmt.Printf("\n--- Content ---\n")

	for _, el := range doc.Body.Content {
		if el.Paragraph != nil {
			for _, pe := range el.Paragraph.Elements {
				if pe.TextRun != nil {
					fmt.Print(pe.TextRun.Content)
				}
			}
		}
	}
}

// getDocumentEndIndex returns the end index of the document body (before the trailing newline).
func getDocumentEndIndex(token, docID string) int {
	u := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s", url.PathEscape(docID))
	data, status, err := apiRequest(token, "GET", u, nil)
	if err != nil {
		log.Fatalf("cannot get document for end index: %v", err)
	}
	if status != 200 {
		log.Fatalf("get document failed (HTTP %d): %s", status, string(data))
	}

	var doc struct {
		Body struct {
			Content []struct {
				EndIndex int `json:"endIndex"`
			} `json:"content"`
		} `json:"body"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		log.Fatalf("cannot parse document structure: %v", err)
	}

	if len(doc.Body.Content) == 0 {
		return 1
	}
	endIdx := doc.Body.Content[len(doc.Body.Content)-1].EndIndex
	// The very last character is always a newline; insert before it.
	if endIdx > 1 {
		return endIdx - 1
	}
	return 1
}

// appendText appends text to the end of a document.
func appendText(token, docID, text, format string) {
	endIdx := getDocumentEndIndex(token, docID)

	requests := []map[string]interface{}{
		{
			"insertText": map[string]interface{}{
				"location": map[string]interface{}{
					"index": endIdx,
				},
				"text": text,
			},
		},
	}
	batchUpdate(token, docID, requests, format, "Text appended successfully.")
}

// prependText prepends text to the beginning of a document.
func prependText(token, docID, text, format string) {
	requests := []map[string]interface{}{
		{
			"insertText": map[string]interface{}{
				"location": map[string]interface{}{
					"index": 1,
				},
				"text": text,
			},
		},
	}
	batchUpdate(token, docID, requests, format, "Text prepended successfully.")
}

// replaceText performs find-and-replace in a document.
func replaceText(token, docID, find, replaceWith, format string) {
	if find == "" {
		log.Fatalf("--find is required for --replace")
	}
	requests := []map[string]interface{}{
		{
			"replaceAllText": map[string]interface{}{
				"containsText": map[string]interface{}{
					"text":      find,
					"matchCase": true,
				},
				"replaceText": replaceWith,
			},
		},
	}
	batchUpdate(token, docID, requests, format, fmt.Sprintf("Replaced all occurrences of %q with %q.", find, replaceWith))
}

// batchUpdate sends a batchUpdate request to the Docs API.
func batchUpdate(token, docID string, requests []map[string]interface{}, format, successMsg string) {
	payload := map[string]interface{}{
		"requests": requests,
	}
	jsonData, _ := json.Marshal(payload)

	u := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s:batchUpdate", url.PathEscape(docID))
	data, status, err := apiRequest(token, "POST", u, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot update document: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("update document failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}
	fmt.Println(successMsg)
}

// listDocuments lists recent Google Docs using the Drive API.
func listDocuments(token string, maxResults int, format string) {
	q := "mimeType='application/vnd.google-apps.document'"
	u := fmt.Sprintf(
		"https://www.googleapis.com/drive/v3/files?q=%s&pageSize=%d&orderBy=modifiedTime+desc&fields=files(id,name,modifiedTime,createdTime)",
		url.QueryEscape(q),
		maxResults,
	)

	data, status, err := apiRequest(token, "GET", u, nil)
	if err != nil {
		log.Fatalf("cannot list documents: %v", err)
	}
	if status != 200 {
		log.Fatalf("list documents failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Files []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			ModifiedTime string `json:"modifiedTime"`
			CreatedTime  string `json:"createdTime"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse file list: %v", err)
	}

	fmt.Printf("Documents (%d):\n\n", len(result.Files))
	for i, f := range result.Files {
		modified := f.ModifiedTime
		if t, err := time.Parse(time.RFC3339, f.ModifiedTime); err == nil {
			modified = t.Format("2006-01-02 15:04")
		}
		fmt.Printf("%d. %s\n   ID:       %s\n   Modified: %s\n   URL:      https://docs.google.com/document/d/%s/edit\n\n",
			i+1, f.Name, f.ID, modified, f.ID)
	}
}

func main() {
	// Config flags
	setupFlag := flag.Bool("setup", false, "Save Google credentials to config")
	showConfigFlag := flag.Bool("show-config", false, "Show current config (masks token)")
	globalFlag := flag.Bool("global", false, "Target global config instead of project")
	accessTokenFlag := flag.String("access-token", "", "OAuth2 access token")
	serviceAccountFileFlag := flag.String("service-account-file", "", "Path to service account JSON key file")

	// Operation flags
	createFlag := flag.Bool("create", false, "Create a new document")
	getFlag := flag.String("get", "", "Get document content by ID")
	appendFlag := flag.String("append", "", "Append text to document (doc ID)")
	prependFlag := flag.String("prepend", "", "Prepend text to document (doc ID)")
	replaceFlag := flag.String("replace", "", "Find and replace text in document (doc ID)")
	listFlag := flag.Bool("list", false, "List recent documents")

	// Content flags
	titleFlag := flag.String("title", "", "Document title (for --create)")
	bodyFlag := flag.String("body", "", "Text content to append/prepend")
	bodyFileFlag := flag.String("body-file", "", "Read text content from a file")
	findFlag := flag.String("find", "", "Text to find (for --replace)")
	replaceWithFlag := flag.String("replace-with", "", "Replacement text (for --replace)")
	formatFlag := flag.String("format", "text", "Output format: text, json")
	rowsFlag := flag.Int("rows", 25, "Max results for list")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: google_docs.go [flags]\n\nManage Google Docs from the command line.\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Handle setup
	if *setupFlag {
		var cfg googleConfig
		if *accessTokenFlag != "" {
			cfg = googleConfig{
				AuthMethod:  "access_token",
				AccessToken: *accessTokenFlag,
			}
		} else if *serviceAccountFileFlag != "" {
			// Validate that the file exists
			if _, err := os.Stat(*serviceAccountFileFlag); err != nil {
				log.Fatalf("service account file not found: %v", err)
			}
			cfg = googleConfig{
				AuthMethod:         "service_account",
				ServiceAccountFile: *serviceAccountFileFlag,
			}
		} else {
			log.Fatalf("--access-token or --service-account-file is required for --setup")
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
			fmt.Println("No Google configuration found.")
			fmt.Println("Run with --setup to configure, or set GOOGLE_ACCESS_TOKEN or GOOGLE_SERVICE_ACCOUNT_FILE env vars.")
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
		log.Fatalf("Google not configured. Run with --setup or set GOOGLE_ACCESS_TOKEN or GOOGLE_SERVICE_ACCOUNT_FILE env vars.")
	}

	token := resolveAccessToken(cfg)

	switch {
	case *createFlag:
		createDocument(token, *titleFlag, *formatFlag)
	case *getFlag != "":
		getDocument(token, *getFlag, *formatFlag)
	case *appendFlag != "":
		if bodyContent == "" {
			log.Fatalf("--body or --body-file is required for --append")
		}
		appendText(token, *appendFlag, bodyContent, *formatFlag)
	case *prependFlag != "":
		if bodyContent == "" {
			log.Fatalf("--body or --body-file is required for --prepend")
		}
		prependText(token, *prependFlag, bodyContent, *formatFlag)
	case *replaceFlag != "":
		replaceText(token, *replaceFlag, *findFlag, *replaceWithFlag, *formatFlag)
	case *listFlag:
		listDocuments(token, *rowsFlag, *formatFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
