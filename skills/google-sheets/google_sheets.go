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

// googleConfig holds Google auth credentials (shared with google-docs).
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
	return c.AccessToken != "" || c.ServiceAccountFile != ""
}

// projectConfigPath returns the project-level config path.
func projectConfigPath() string {
	return filepath.Join(".claude", "google-config.yaml")
}

// globalConfigPath returns the global config path.
func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "google-config.yaml")
}

// loadConfigFile reads a config from a YAML file.
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
	if token := os.Getenv("GOOGLE_ACCESS_TOKEN"); token != "" {
		return googleConfig{AuthMethod: "access_token", AccessToken: token}
	}
	if saFile := os.Getenv("GOOGLE_SERVICE_ACCOUNT_FILE"); saFile != "" {
		return googleConfig{AuthMethod: "service_account", ServiceAccountFile: saFile}
	}
	return googleConfig{}
}

// saveConfig writes config to a YAML file with 0600 permissions.
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
		if cfg.AccessToken != "" {
			return cfg.AccessToken
		}
		if cfg.ServiceAccountFile != "" {
			return getServiceAccountToken(cfg.ServiceAccountFile)
		}
		log.Fatalf("no valid auth method configured")
		return ""
	}
}

// serviceAccountKey represents the JSON key file for a service account.
type serviceAccountKey struct {
	Type         string `json:"type"`
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyID string `json:"private_key_id"`
	TokenURI     string `json:"token_uri"`
}

// getServiceAccountToken exchanges a service account key for an access token via JWT.
func getServiceAccountToken(saFilePath string) string {
	data, err := os.ReadFile(saFilePath)
	if err != nil {
		log.Fatalf("cannot read service account file: %v", err)
	}
	var sa serviceAccountKey
	if err := json.Unmarshal(data, &sa); err != nil {
		log.Fatalf("cannot parse service account file: %v", err)
	}
	if sa.PrivateKey == "" || sa.ClientEmail == "" {
		log.Fatalf("service account file missing required fields (private_key, client_email)")
	}

	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	now := time.Now()
	// JWT header
	header := base64URLEncode([]byte(`{"alg":"RS256","typ":"JWT"}`))
	// JWT claims
	claims := map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/spreadsheets https://www.googleapis.com/auth/drive",
		"aud":   tokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64URLEncode(claimsJSON)

	// Sign
	signingInput := header + "." + claimsB64
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		log.Fatalf("cannot decode PEM block from service account private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("cannot parse private key: %v", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		log.Fatalf("private key is not RSA")
	}
	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
	if err != nil {
		log.Fatalf("cannot sign JWT: %v", err)
	}
	jwt := signingInput + "." + base64URLEncode(sig)

	// Exchange JWT for access token
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	}
	resp, err := http.PostForm(tokenURI, form)
	if err != nil {
		log.Fatalf("cannot request token: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("cannot read token response: %v", err)
	}
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
		log.Fatalf("token response missing access_token")
	}
	return tokenResp.AccessToken
}

// base64URLEncode encodes bytes to base64url without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// sheetsAPI performs an authenticated request to the Google Sheets API.
func sheetsAPI(token, method, urlStr string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, urlStr, body)
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

const sheetsBaseURL = "https://sheets.googleapis.com/v4/spreadsheets"

// createSpreadsheet creates a new spreadsheet.
func createSpreadsheet(token, title, format string) {
	if title == "" {
		log.Fatalf("--title is required for --create")
	}
	payload := map[string]interface{}{
		"properties": map[string]interface{}{
			"title": title,
		},
	}
	jsonData, _ := json.Marshal(payload)
	data, status, err := sheetsAPI(token, "POST", sheetsBaseURL, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot create spreadsheet: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("create spreadsheet failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		SpreadsheetID  string `json:"spreadsheetId"`
		SpreadsheetURL string `json:"spreadsheetUrl"`
		Properties     struct {
			Title string `json:"title"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse response: %v", err)
	}
	fmt.Printf("Spreadsheet created successfully.\n")
	fmt.Printf("ID:    %s\n", result.SpreadsheetID)
	fmt.Printf("Title: %s\n", result.Properties.Title)
	fmt.Printf("URL:   %s\n", result.SpreadsheetURL)
}

// getSpreadsheet retrieves spreadsheet metadata including sheet list.
func getSpreadsheet(token, spreadsheetID, format string) {
	u := fmt.Sprintf("%s/%s?fields=spreadsheetId,properties,sheets.properties", sheetsBaseURL, url.PathEscape(spreadsheetID))
	data, status, err := sheetsAPI(token, "GET", u, nil)
	if err != nil {
		log.Fatalf("cannot get spreadsheet: %v", err)
	}
	if status != 200 {
		log.Fatalf("get spreadsheet failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		SpreadsheetID string `json:"spreadsheetId"`
		Properties    struct {
			Title  string `json:"title"`
			Locale string `json:"locale"`
		} `json:"properties"`
		Sheets []struct {
			Properties struct {
				SheetID    int    `json:"sheetId"`
				Title      string `json:"title"`
				Index      int    `json:"index"`
				SheetType  string `json:"sheetType"`
				GridProps  struct {
					RowCount    int `json:"rowCount"`
					ColumnCount int `json:"columnCount"`
				} `json:"gridProperties"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse spreadsheet: %v", err)
	}

	fmt.Printf("Spreadsheet: %s\n", result.Properties.Title)
	fmt.Printf("ID:          %s\n", result.SpreadsheetID)
	if result.Properties.Locale != "" {
		fmt.Printf("Locale:      %s\n", result.Properties.Locale)
	}
	fmt.Printf("\nSheets (%d):\n", len(result.Sheets))
	for _, s := range result.Sheets {
		p := s.Properties
		fmt.Printf("  [%d] %-30s %s (%d rows x %d cols)\n",
			p.SheetID, p.Title, p.SheetType, p.GridProps.RowCount, p.GridProps.ColumnCount)
	}
}

// readValues reads cell values from a spreadsheet range.
func readValues(token, spreadsheetID, rangeStr, sheet, format string, maxRows int) {
	if rangeStr == "" && sheet == "" {
		log.Fatalf("--range or --sheet is required for --read")
	}
	if rangeStr == "" && sheet != "" {
		rangeStr = sheet
	}
	u := fmt.Sprintf("%s/%s/values/%s?valueRenderOption=FORMATTED_VALUE",
		sheetsBaseURL, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr))
	data, status, err := sheetsAPI(token, "GET", u, nil)
	if err != nil {
		log.Fatalf("cannot read values: %v", err)
	}
	if status != 200 {
		log.Fatalf("read values failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Range  string            `json:"range"`
		Values [][]interface{}   `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse values: %v", err)
	}

	if len(result.Values) == 0 {
		fmt.Printf("Range: %s\n(no data)\n", result.Range)
		return
	}

	rows := result.Values
	if maxRows > 0 && len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	if format == "csv" {
		printCSV(rows)
		return
	}

	// Default text format: aligned columns
	fmt.Printf("Range: %s (%d rows)\n\n", result.Range, len(result.Values))
	printTextTable(rows)
	if maxRows > 0 && len(result.Values) > maxRows {
		fmt.Printf("\n... showing %d of %d rows\n", maxRows, len(result.Values))
	}
}

// printCSV outputs rows as CSV.
func printCSV(rows [][]interface{}) {
	for _, row := range rows {
		parts := make([]string, len(row))
		for i, cell := range row {
			s := fmt.Sprintf("%v", cell)
			if strings.ContainsAny(s, ",\"\n") {
				s = "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
			}
			parts[i] = s
		}
		fmt.Println(strings.Join(parts, ","))
	}
}

// printTextTable prints rows as an aligned text table.
func printTextTable(rows [][]interface{}) {
	// Determine max column count
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	if maxCols == 0 {
		return
	}

	// Compute column widths
	widths := make([]int, maxCols)
	for _, row := range rows {
		for i, cell := range row {
			s := fmt.Sprintf("%v", cell)
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	// Cap column widths at 40
	for i := range widths {
		if widths[i] > 40 {
			widths[i] = 40
		}
		if widths[i] < 2 {
			widths[i] = 2
		}
	}

	// Print rows
	for _, row := range rows {
		parts := make([]string, maxCols)
		for i := 0; i < maxCols; i++ {
			val := ""
			if i < len(row) {
				val = fmt.Sprintf("%v", row[i])
			}
			if len(val) > 40 {
				val = val[:37] + "..."
			}
			parts[i] = fmt.Sprintf("%-*s", widths[i], val)
		}
		fmt.Println(strings.Join(parts, "  "))
	}
}

// parseValues parses a JSON array of arrays from a string or file.
func parseValues(valuesStr, valuesFile string) [][]interface{} {
	raw := valuesStr
	if valuesFile != "" {
		data, err := os.ReadFile(valuesFile)
		if err != nil {
			log.Fatalf("cannot read values file: %v", err)
		}
		raw = string(data)
	}
	if raw == "" {
		log.Fatalf("--values or --values-file is required")
	}
	var values [][]interface{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		log.Fatalf("cannot parse values JSON (expected array of arrays): %v", err)
	}
	return values
}

// writeValues writes values to a spreadsheet range.
func writeValues(token, spreadsheetID, rangeStr, valuesStr, valuesFile, format string) {
	if rangeStr == "" {
		log.Fatalf("--range is required for --write")
	}
	values := parseValues(valuesStr, valuesFile)

	payload := map[string]interface{}{
		"range":          rangeStr,
		"majorDimension": "ROWS",
		"values":         values,
	}
	jsonData, _ := json.Marshal(payload)

	u := fmt.Sprintf("%s/%s/values/%s?valueInputOption=USER_ENTERED",
		sheetsBaseURL, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr))
	data, status, err := sheetsAPI(token, "PUT", u, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot write values: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("write values failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		UpdatedRange string `json:"updatedRange"`
		UpdatedRows  int    `json:"updatedRows"`
		UpdatedCols  int    `json:"updatedColumns"`
		UpdatedCells int    `json:"updatedCells"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse write response: %v", err)
	}
	fmt.Printf("Values written successfully.\n")
	fmt.Printf("Range:   %s\n", result.UpdatedRange)
	fmt.Printf("Updated: %d rows, %d columns, %d cells\n", result.UpdatedRows, result.UpdatedCols, result.UpdatedCells)
}

// appendValues appends rows to a spreadsheet range.
func appendValues(token, spreadsheetID, rangeStr, valuesStr, valuesFile, format string) {
	if rangeStr == "" {
		log.Fatalf("--range is required for --append")
	}
	values := parseValues(valuesStr, valuesFile)

	payload := map[string]interface{}{
		"range":          rangeStr,
		"majorDimension": "ROWS",
		"values":         values,
	}
	jsonData, _ := json.Marshal(payload)

	u := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=USER_ENTERED&insertDataOption=INSERT_ROWS",
		sheetsBaseURL, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr))
	data, status, err := sheetsAPI(token, "POST", u, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot append values: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("append values failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Updates struct {
			UpdatedRange string `json:"updatedRange"`
			UpdatedRows  int    `json:"updatedRows"`
			UpdatedCols  int    `json:"updatedColumns"`
			UpdatedCells int    `json:"updatedCells"`
		} `json:"updates"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse append response: %v", err)
	}
	fmt.Printf("Rows appended successfully.\n")
	fmt.Printf("Range:   %s\n", result.Updates.UpdatedRange)
	fmt.Printf("Updated: %d rows, %d columns, %d cells\n", result.Updates.UpdatedRows, result.Updates.UpdatedCols, result.Updates.UpdatedCells)
}

// addSheet adds a new sheet tab to a spreadsheet.
func addSheet(token, spreadsheetID, sheetName, format string) {
	if sheetName == "" {
		log.Fatalf("--sheet is required for --add-sheet")
	}
	payload := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"addSheet": map[string]interface{}{
					"properties": map[string]interface{}{
						"title": sheetName,
					},
				},
			},
		},
	}
	jsonData, _ := json.Marshal(payload)

	u := fmt.Sprintf("%s/%s:batchUpdate", sheetsBaseURL, url.PathEscape(spreadsheetID))
	data, status, err := sheetsAPI(token, "POST", u, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot add sheet: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("add sheet failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		Replies []struct {
			AddSheet struct {
				Properties struct {
					SheetID int    `json:"sheetId"`
					Title   string `json:"title"`
				} `json:"properties"`
			} `json:"addSheet"`
		} `json:"replies"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse add-sheet response: %v", err)
	}
	if len(result.Replies) > 0 {
		p := result.Replies[0].AddSheet.Properties
		fmt.Printf("Sheet added successfully.\n")
		fmt.Printf("Sheet ID: %d\n", p.SheetID)
		fmt.Printf("Title:    %s\n", p.Title)
	} else {
		fmt.Println("Sheet added successfully.")
	}
}

// deleteSheet deletes a sheet tab from a spreadsheet by name.
func deleteSheet(token, spreadsheetID, sheetName, format string) {
	if sheetName == "" {
		log.Fatalf("--sheet is required for --delete-sheet")
	}

	// First, get the sheet ID by name
	sheetID := resolveSheetID(token, spreadsheetID, sheetName)

	payload := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"deleteSheet": map[string]interface{}{
					"sheetId": sheetID,
				},
			},
		},
	}
	jsonData, _ := json.Marshal(payload)

	u := fmt.Sprintf("%s/%s:batchUpdate", sheetsBaseURL, url.PathEscape(spreadsheetID))
	data, status, err := sheetsAPI(token, "POST", u, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Fatalf("cannot delete sheet: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("delete sheet failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Sheet %q deleted successfully.\n", sheetName)
}

// resolveSheetID finds the sheet ID for a given sheet name.
func resolveSheetID(token, spreadsheetID, sheetName string) int {
	u := fmt.Sprintf("%s/%s?fields=sheets.properties", sheetsBaseURL, url.PathEscape(spreadsheetID))
	data, status, err := sheetsAPI(token, "GET", u, nil)
	if err != nil {
		log.Fatalf("cannot get spreadsheet metadata: %v", err)
	}
	if status != 200 {
		log.Fatalf("get spreadsheet failed (HTTP %d): %s", status, string(data))
	}

	var result struct {
		Sheets []struct {
			Properties struct {
				SheetID int    `json:"sheetId"`
				Title   string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse spreadsheet: %v", err)
	}

	for _, s := range result.Sheets {
		if s.Properties.Title == sheetName {
			return s.Properties.SheetID
		}
	}
	log.Fatalf("sheet %q not found in spreadsheet %s", sheetName, spreadsheetID)
	return 0
}

// clearRange clears a range of cells.
func clearRange(token, spreadsheetID, rangeStr, format string) {
	if rangeStr == "" {
		log.Fatalf("--range is required for --clear")
	}

	u := fmt.Sprintf("%s/%s/values/%s:clear",
		sheetsBaseURL, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr))
	data, status, err := sheetsAPI(token, "POST", u, strings.NewReader("{}"))
	if err != nil {
		log.Fatalf("cannot clear range: %v", err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("clear range failed (HTTP %d): %s", status, string(data))
	}

	if format == "json" {
		fmt.Println(string(data))
		return
	}

	var result struct {
		ClearedRange string `json:"clearedRange"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("cannot parse clear response: %v", err)
	}
	fmt.Printf("Range cleared successfully.\n")
	fmt.Printf("Cleared: %s\n", result.ClearedRange)
}

func main() {
	// Config flags
	setupFlag := flag.Bool("setup", false, "Save Google credentials to config")
	showConfigFlag := flag.Bool("show-config", false, "Show current config (masks token)")
	globalFlag := flag.Bool("global", false, "Target global config instead of project")
	accessTokenFlag := flag.String("access-token", "", "Google OAuth2 access token")
	serviceAccountFileFlag := flag.String("service-account-file", "", "Path to service account JSON key file")

	// Operation flags
	createFlag := flag.Bool("create", false, "Create a new spreadsheet")
	getFlag := flag.String("get", "", "Get spreadsheet metadata (spreadsheet ID)")
	readFlag := flag.String("read", "", "Read cell values (spreadsheet ID)")
	writeFlag := flag.String("write", "", "Write values to cells (spreadsheet ID)")
	appendFlag := flag.String("append", "", "Append rows (spreadsheet ID)")
	addSheetFlag := flag.String("add-sheet", "", "Add a new sheet tab (spreadsheet ID)")
	deleteSheetFlag := flag.String("delete-sheet", "", "Delete a sheet tab (spreadsheet ID)")
	clearFlag := flag.String("clear", "", "Clear a range of cells (spreadsheet ID)")

	// Content flags
	titleFlag := flag.String("title", "", "Spreadsheet title (for --create)")
	rangeFlag := flag.String("range", "", "Cell range in A1 notation")
	sheetFlag := flag.String("sheet", "", "Sheet tab name")
	valuesFlag := flag.String("values", "", "JSON array of row arrays")
	valuesFileFlag := flag.String("values-file", "", "Read values JSON from a file")
	formatFlag := flag.String("format", "text", "Output format: text, json, csv")
	rowsFlag := flag.Int("rows", 50, "Max rows to display")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: google_sheets.go [flags]\n\nManage Google Sheets from the command line.\n\nFlags:\n")
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
	case *createFlag:
		createSpreadsheet(token, *titleFlag, *formatFlag)
	case *getFlag != "":
		getSpreadsheet(token, *getFlag, *formatFlag)
	case *readFlag != "":
		readValues(token, *readFlag, *rangeFlag, *sheetFlag, *formatFlag, *rowsFlag)
	case *writeFlag != "":
		writeValues(token, *writeFlag, *rangeFlag, *valuesFlag, *valuesFileFlag, *formatFlag)
	case *appendFlag != "":
		appendValues(token, *appendFlag, *rangeFlag, *valuesFlag, *valuesFileFlag, *formatFlag)
	case *addSheetFlag != "":
		addSheet(token, *addSheetFlag, *sheetFlag, *formatFlag)
	case *deleteSheetFlag != "":
		deleteSheet(token, *deleteSheetFlag, *sheetFlag, *formatFlag)
	case *clearFlag != "":
		clearRange(token, *clearFlag, *rangeFlag, *formatFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
