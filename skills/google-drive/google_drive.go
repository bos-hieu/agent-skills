package main

import (
	"bytes"
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
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// googleConfig holds Google credentials shared with google-docs.
type googleConfig struct {
	AuthMethod         string `yaml:"auth_method" json:"auth_method"`
	AccessToken        string `yaml:"access_token" json:"access_token"`
	ServiceAccountFile string `yaml:"service_account_file" json:"service_account_file"`
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

// resolveAccessToken returns a valid access token from config.
// For service accounts, it performs JWT-based token exchange.
func resolveAccessToken(cfg googleConfig) string {
	switch cfg.AuthMethod {
	case "access_token":
		return cfg.AccessToken
	case "service_account":
		return getServiceAccountToken(cfg.ServiceAccountFile)
	default:
		log.Fatalf("unknown auth method: %s", cfg.AuthMethod)
		return ""
	}
}

// serviceAccountKey holds the relevant fields from a service account JSON key file.
type serviceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// getServiceAccountToken exchanges a service account key for an access token via JWT.
func getServiceAccountToken(saFile string) string {
	data, err := os.ReadFile(saFile)
	if err != nil {
		log.Fatalf("cannot read service account file: %v", err)
	}
	var sa serviceAccountKey
	if err := json.Unmarshal(data, &sa); err != nil {
		log.Fatalf("cannot parse service account JSON: %v", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		log.Fatalf("service account file missing client_email or private_key")
	}
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	now := time.Now().Unix()
	header := base64URLEncode([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claimSet := map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/drive",
		"aud":   tokenURI,
		"iat":   now,
		"exp":   now + 3600,
	}
	claimJSON, _ := json.Marshal(claimSet)
	payload := base64URLEncode(claimJSON)

	signingInput := header + "." + payload
	signature := rsaSign(sa.PrivateKey, []byte(signingInput))
	jwt := signingInput + "." + base64URLEncode(signature)

	// Exchange JWT for access token.
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	}
	resp, err := http.PostForm(tokenURI, form)
	if err != nil {
		log.Fatalf("token exchange failed: %v", err)
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
		log.Fatalf("empty access token in response")
	}
	return tokenResp.AccessToken
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func rsaSign(privateKeyPEM string, data []byte) []byte {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		log.Fatalf("cannot decode PEM block from private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("cannot parse private key: %v", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		log.Fatalf("private key is not RSA")
	}
	hash := sha256.Sum256(data)
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
	if err != nil {
		log.Fatalf("cannot sign JWT: %v", err)
	}
	return sig
}

// driveRequest performs an authenticated Google Drive API request.
func driveRequest(token, method, rawURL string, body io.Reader, contentType string) ([]byte, int, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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

const driveAPIBase = "https://www.googleapis.com/drive/v3"
const uploadAPIBase = "https://www.googleapis.com/upload/drive/v3"

func addSharedDriveParams(params url.Values) {
	params.Set("supportsAllDrives", "true")
	params.Set("includeItemsFromAllDrives", "true")
}

// listFiles lists files in Google Drive.
func listFiles(token, query, folderID, format string, maxResults int) {
	params := url.Values{}
	addSharedDriveParams(params)
	params.Set("pageSize", fmt.Sprintf("%d", maxResults))
	params.Set("fields", "files(id,name,mimeType,size,modifiedTime,parents)")

	q := query
	if folderID != "" {
		folderQ := fmt.Sprintf("'%s' in parents", folderID)
		if q != "" {
			q = folderQ + " and " + q
		} else {
			q = folderQ
		}
	}
	if q != "" {
		params.Set("q", q)
	}

	u := driveAPIBase + "/files?" + params.Encode()
	data, status, err := driveRequest(token, "GET", u, nil, "")
	if err != nil {
		log.Fatalf("cannot list files: %v", err)
	}
	if status != 200 {
		log.Fatalf("list files failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Files []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			MimeType     string `json:"mimeType"`
			Size         string `json:"size"`
			ModifiedTime string `json:"modifiedTime"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse file list: %v", err)
	}

	fmt.Printf("Files (%d):\n\n", len(result.Files))
	for _, f := range result.Files {
		sizeStr := f.Size
		if sizeStr == "" {
			sizeStr = "-"
		}
		modified := f.ModifiedTime
		if len(modified) > 10 {
			modified = modified[:10]
		}
		fmt.Printf("  %-44s  %-12s  %s  %s\n", f.ID, sizeStr, modified, f.Name)
	}
}

// getFile retrieves file metadata.
func getFile(token, fileID, format string) {
	params := url.Values{}
	addSharedDriveParams(params)
	params.Set("fields", "id,name,mimeType,size,modifiedTime,createdTime,parents,webViewLink,owners,shared")
	u := driveAPIBase + "/files/" + url.PathEscape(fileID) + "?" + params.Encode()
	data, status, err := driveRequest(token, "GET", u, nil, "")
	if err != nil {
		log.Fatalf("cannot get file: %v", err)
	}
	if status != 200 {
		log.Fatalf("get file failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var f struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		MimeType     string `json:"mimeType"`
		Size         string `json:"size"`
		ModifiedTime string `json:"modifiedTime"`
		CreatedTime  string `json:"createdTime"`
		WebViewLink  string `json:"webViewLink"`
		Shared       bool   `json:"shared"`
		Parents      []string `json:"parents"`
		Owners       []struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"owners"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		log.Fatalf("cannot parse file metadata: %v", err)
	}

	fmt.Printf("ID:        %s\n", f.ID)
	fmt.Printf("Name:      %s\n", f.Name)
	fmt.Printf("MIME Type: %s\n", f.MimeType)
	fmt.Printf("Size:      %s\n", f.Size)
	fmt.Printf("Created:   %s\n", f.CreatedTime)
	fmt.Printf("Modified:  %s\n", f.ModifiedTime)
	fmt.Printf("Shared:    %v\n", f.Shared)
	if f.WebViewLink != "" {
		fmt.Printf("URL:       %s\n", f.WebViewLink)
	}
	if len(f.Parents) > 0 {
		fmt.Printf("Parents:   %s\n", strings.Join(f.Parents, ", "))
	}
	if len(f.Owners) > 0 {
		fmt.Printf("Owner:     %s (%s)\n", f.Owners[0].DisplayName, f.Owners[0].EmailAddress)
	}
}

// downloadFile downloads file content.
func downloadFile(token, fileID, output string) {
	// First get metadata to determine mime type.
	metaParams := url.Values{}
	addSharedDriveParams(metaParams)
	metaParams.Set("fields", "mimeType,name")
	metaURL := driveAPIBase + "/files/" + url.PathEscape(fileID) + "?" + metaParams.Encode()
	metaData, metaStatus, err := driveRequest(token, "GET", metaURL, nil, "")
	if err != nil {
		log.Fatalf("cannot get file metadata: %v", err)
	}
	if metaStatus != 200 {
		log.Fatalf("get file metadata failed (HTTP %d): %s", metaStatus, string(metaData))
	}

	var meta struct {
		MimeType string `json:"mimeType"`
		Name     string `json:"name"`
	}
	json.Unmarshal(metaData, &meta)

	var downloadURL string
	// Google Workspace files need export; others use direct download.
	switch meta.MimeType {
	case "application/vnd.google-apps.document":
		downloadURL = driveAPIBase + "/files/" + url.PathEscape(fileID) + "/export?mimeType=text/plain"
	case "application/vnd.google-apps.spreadsheet":
		downloadURL = driveAPIBase + "/files/" + url.PathEscape(fileID) + "/export?mimeType=text/csv"
	case "application/vnd.google-apps.presentation":
		downloadURL = driveAPIBase + "/files/" + url.PathEscape(fileID) + "/export?mimeType=application/pdf"
	case "application/vnd.google-apps.drawing":
		downloadURL = driveAPIBase + "/files/" + url.PathEscape(fileID) + "/export?mimeType=image/png"
	default:
		downloadURL = driveAPIBase + "/files/" + url.PathEscape(fileID) + "?alt=media&supportsAllDrives=true"
	}

	if strings.Contains(downloadURL, "/export?") {
		downloadURL += "&supportsAllDrives=true"
	}

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		log.Fatalf("cannot create download request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("download failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	if output == "" {
		// Write to stdout.
		if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
			log.Fatalf("cannot write to stdout: %v", err)
		}
		return
	}

	f, err := os.Create(output)
	if err != nil {
		log.Fatalf("cannot create output file: %v", err)
	}
	defer f.Close()
	n, err := io.Copy(f, resp.Body)
	if err != nil {
		log.Fatalf("cannot write output file: %v", err)
	}
	fmt.Printf("Downloaded %s to %s (%d bytes)\n", meta.Name, output, n)
}

// uploadFile uploads a local file to Google Drive.
func uploadFile(token, localPath, title, folderID, format string) {
	if localPath == "" {
		log.Fatalf("--file is required for --upload")
	}

	fileData, err := os.ReadFile(localPath)
	if err != nil {
		log.Fatalf("cannot read file %s: %v", localPath, err)
	}

	if title == "" {
		title = filepath.Base(localPath)
	}

	// Detect MIME type from file extension.
	mimeType := mime.TypeByExtension(filepath.Ext(localPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Build metadata.
	metadata := map[string]interface{}{
		"name": title,
	}
	if folderID != "" {
		metadata["parents"] = []string{folderID}
	}
	metaJSON, _ := json.Marshal(metadata)

	// Use multipart upload.
	boundary := "skill_upload_boundary"
	var body bytes.Buffer
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: application/json; charset=UTF-8\r\n\r\n")
	body.Write(metaJSON)
	body.WriteString("\r\n--" + boundary + "\r\n")
	body.WriteString(fmt.Sprintf("Content-Type: %s\r\n\r\n", mimeType))
	body.Write(fileData)
	body.WriteString("\r\n--" + boundary + "--\r\n")

	u := uploadAPIBase + "/files?uploadType=multipart&fields=id,name,mimeType,size,webViewLink&supportsAllDrives=true"
	contentType := "multipart/related; boundary=" + boundary
	data, status, err := driveRequest(token, "POST", u, &body, contentType)
	if err != nil {
		log.Fatalf("upload failed: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("upload failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		MimeType    string `json:"mimeType"`
		Size        string `json:"size"`
		WebViewLink string `json:"webViewLink"`
	}
	json.Unmarshal(data, &result)

	fmt.Println("File uploaded successfully.")
	fmt.Printf("ID:        %s\n", result.ID)
	fmt.Printf("Name:      %s\n", result.Name)
	fmt.Printf("MIME Type: %s\n", result.MimeType)
	fmt.Printf("Size:      %s\n", result.Size)
	if result.WebViewLink != "" {
		fmt.Printf("URL:       %s\n", result.WebViewLink)
	}
}

// mkdirDrive creates a folder in Google Drive.
func mkdirDrive(token, title, folderID, format string) {
	if title == "" {
		log.Fatalf("--title is required for --mkdir")
	}

	metadata := map[string]interface{}{
		"name":     title,
		"mimeType": "application/vnd.google-apps.folder",
	}
	if folderID != "" {
		metadata["parents"] = []string{folderID}
	}
	metaJSON, _ := json.Marshal(metadata)

	u := driveAPIBase + "/files?fields=id,name,mimeType,webViewLink&supportsAllDrives=true"
	data, status, err := driveRequest(token, "POST", u, bytes.NewReader(metaJSON), "application/json")
	if err != nil {
		log.Fatalf("cannot create folder: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("create folder failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		WebViewLink string `json:"webViewLink"`
	}
	json.Unmarshal(data, &result)

	fmt.Println("Folder created successfully.")
	fmt.Printf("ID:   %s\n", result.ID)
	fmt.Printf("Name: %s\n", result.Name)
	if result.WebViewLink != "" {
		fmt.Printf("URL:  %s\n", result.WebViewLink)
	}
}

// deleteFile moves a file to trash.
func deleteFile(token, fileID string) {
	metaJSON := []byte(`{"trashed":true}`)
	u := driveAPIBase + "/files/" + url.PathEscape(fileID) + "?supportsAllDrives=true"
	data, status, err := driveRequest(token, "PATCH", u, bytes.NewReader(metaJSON), "application/json")
	if err != nil {
		log.Fatalf("cannot trash file: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("trash file failed (HTTP %d): %s", status, string(data))
	}
	fmt.Printf("File %s moved to trash.\n", fileID)
}

// searchFiles searches Google Drive files.
func searchFiles(token, query, format string, maxResults int) {
	q := fmt.Sprintf("fullText contains '%s' or name contains '%s'",
		strings.ReplaceAll(query, "'", "\\'"),
		strings.ReplaceAll(query, "'", "\\'"))

	params := url.Values{}
	addSharedDriveParams(params)
	params.Set("q", q)
	params.Set("pageSize", fmt.Sprintf("%d", maxResults))
	params.Set("fields", "files(id,name,mimeType,size,modifiedTime)")

	u := driveAPIBase + "/files?" + params.Encode()
	data, status, err := driveRequest(token, "GET", u, nil, "")
	if err != nil {
		log.Fatalf("cannot search files: %v", err)
	}
	if status != 200 {
		log.Fatalf("search failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Files []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			MimeType     string `json:"mimeType"`
			Size         string `json:"size"`
			ModifiedTime string `json:"modifiedTime"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse search results: %v", err)
	}

	fmt.Printf("Found %d file(s):\n\n", len(result.Files))
	for i, f := range result.Files {
		sizeStr := f.Size
		if sizeStr == "" {
			sizeStr = "-"
		}
		modified := f.ModifiedTime
		if len(modified) > 10 {
			modified = modified[:10]
		}
		fmt.Printf("%d. [%s] %s\n", i+1, f.ID, f.Name)
		fmt.Printf("   Type: %s  Size: %s  Modified: %s\n\n", f.MimeType, sizeStr, modified)
	}
}

// shareFile shares a file with a user.
func shareFile(token, fileID, email, role string) {
	if email == "" {
		log.Fatalf("--email is required for --share")
	}
	if role == "" {
		role = "reader"
	}
	validRoles := map[string]bool{"reader": true, "writer": true, "commenter": true}
	if !validRoles[role] {
		log.Fatalf("invalid role %q: must be reader, writer, or commenter", role)
	}

	payload := map[string]interface{}{
		"type":         "user",
		"role":         role,
		"emailAddress": email,
	}
	payloadJSON, _ := json.Marshal(payload)

	u := driveAPIBase + "/files/" + url.PathEscape(fileID) + "/permissions?supportsAllDrives=true"
	data, status, err := driveRequest(token, "POST", u, bytes.NewReader(payloadJSON), "application/json")
	if err != nil {
		log.Fatalf("cannot share file: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("share failed (HTTP %d): %s", status, string(data))
	}
	fmt.Printf("File %s shared with %s as %s.\n", fileID, email, role)
}

func main() {
	// Config flags
	setupFlag := flag.Bool("setup", false, "Save Google credentials to config")
	showConfigFlag := flag.Bool("show-config", false, "Show current config (masks token)")
	globalFlag := flag.Bool("global", false, "Target global config instead of project")
	accessTokenFlag := flag.String("access-token", "", "Google access token")
	serviceAccountFileFlag := flag.String("service-account-file", "", "Path to service account JSON key file")

	// Operation flags
	listFlag := flag.Bool("list", false, "List files")
	getFlag := flag.String("get", "", "Get file metadata by ID")
	downloadFlag := flag.String("download", "", "Download file content by ID")
	uploadFlag := flag.Bool("upload", false, "Upload a file")
	mkdirFlag := flag.Bool("mkdir", false, "Create a folder")
	deleteFlag := flag.String("delete", "", "Move file to trash by ID")
	searchFlag := flag.String("search", "", "Search files by name/content")
	shareFlag := flag.String("share", "", "Share a file by ID")

	// Content flags
	queryFlag := flag.String("query", "", "Drive search query (for --list)")
	folderFlag := flag.String("folder", "", "Folder ID")
	fileFlag := flag.String("file", "", "Local file path (for --upload)")
	titleFlag := flag.String("title", "", "File/folder name")
	outputFlag := flag.String("output", "", "Output path (for --download)")
	emailFlag := flag.String("email", "", "Email for sharing")
	roleFlag := flag.String("role", "", "Share role: reader, writer, commenter")
	formatFlag := flag.String("format", "text", "Output format: text, json")
	rowsFlag := flag.Int("rows", 25, "Max results")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: google_drive.go [flags]\n\nManage Google Drive files from the command line.\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Handle setup
	if *setupFlag {
		cfg := googleConfig{}
		if *accessTokenFlag != "" {
			cfg.AuthMethod = "access_token"
			cfg.AccessToken = *accessTokenFlag
		} else if *serviceAccountFileFlag != "" {
			cfg.AuthMethod = "service_account"
			cfg.ServiceAccountFile = *serviceAccountFileFlag
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

	// All remaining operations need valid config
	cfg := loadConfig()
	if !cfg.valid() {
		log.Fatalf("Google not configured. Run with --setup or set GOOGLE_ACCESS_TOKEN or GOOGLE_SERVICE_ACCOUNT_FILE env vars.")
	}
	token := resolveAccessToken(cfg)

	switch {
	case *listFlag:
		listFiles(token, *queryFlag, *folderFlag, *formatFlag, *rowsFlag)
	case *getFlag != "":
		getFile(token, *getFlag, *formatFlag)
	case *downloadFlag != "":
		downloadFile(token, *downloadFlag, *outputFlag)
	case *uploadFlag:
		uploadFile(token, *fileFlag, *titleFlag, *folderFlag, *formatFlag)
	case *mkdirFlag:
		mkdirDrive(token, *titleFlag, *folderFlag, *formatFlag)
	case *deleteFlag != "":
		deleteFile(token, *deleteFlag)
	case *searchFlag != "":
		searchFiles(token, *searchFlag, *formatFlag, *rowsFlag)
	case *shareFlag != "":
		shareFile(token, *shareFlag, *emailFlag, *roleFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
