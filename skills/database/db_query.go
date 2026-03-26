package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gopkg.in/yaml.v3"
)

// dbCredential holds all connection info for one named database.
type dbCredential struct {
	Name   string
	Driver string // "postgres", "mysql", "mongodb"
	DSN    string
	Host   string
	Port   string
	User   string
	DBName string
	Source string // "env", "global", "project"
}

func (c dbCredential) masked() string {
	switch c.Driver {
	case "mysql":
		return fmt.Sprintf("mysql://%s:***@%s:%s/%s", c.User, c.Host, c.Port, c.DBName)
	case "mongodb":
		return fmt.Sprintf("mongodb://%s:***@%s:%s/%s", c.User, c.Host, c.Port, c.DBName)
	default:
		return fmt.Sprintf("postgres://%s:***@%s:%s/%s", c.User, c.Host, c.Port, c.DBName)
	}
}

// dbConfigEntry represents a single database entry in the YAML config file.
type dbConfigEntry struct {
	Driver   string `yaml:"driver,omitempty"`
	DSN      string `yaml:"dsn,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
	DBName   string `yaml:"dbname,omitempty"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
}

// dbConfigFile represents the YAML config file structure.
type dbConfigFile struct {
	Databases map[string]*dbConfigEntry `yaml:"databases"`
}

func main() {
	// Query flags
	listFlag := flag.Bool("list", false, "List all detected databases and exit")
	dbFlag := flag.String("db", "", "Database name to connect to")
	queryFlag := flag.String("query", "", "SQL query to execute")
	tablesFlag := flag.Bool("tables", false, "List all tables")
	describeFlag := flag.String("describe", "", "Describe a table (columns, types, nullability)")
	rowsFlag := flag.Int("rows", 20, "Max rows to display")
	formatFlag := flag.String("format", "table", "Output format: table, csv, json")
	noHeaderFlag := flag.Bool("no-header", false, "Suppress column headers")
	collectionFlag := flag.String("collection", "", "MongoDB collection name (required for --query with MongoDB)")
	sortFlag := flag.String("sort", "", "MongoDB sort specification as JSON (e.g. '{\"createdAt\": -1}')")
	projectFlag := flag.String("project", "", "MongoDB projection as JSON (e.g. '{\"name\": 1, \"email\": 1}')")
	skipFlag := flag.Int("skip", 0, "Number of documents to skip (MongoDB)")
	aggregateFlag := flag.String("aggregate", "", "MongoDB aggregation pipeline as JSON array")
	countFlag := flag.Bool("count", false, "Count documents matching the filter (MongoDB)")
	distinctFlag := flag.String("distinct", "", "Get distinct values for a field (MongoDB)")
	readOnlyFlag := flag.Bool("read-only", true, "Enforce read-only queries via database transaction (enabled by default)")
	readWriteFlag := flag.Bool("read-write", false, "Allow data-modifying queries (disables --read-only)")

	// Config management flags
	addDBFlag := flag.String("add-db", "", "Add a database to config file")
	editDBFlag := flag.String("edit-db", "", "Edit a database in config file")
	removeDBFlag := flag.String("remove-db", "", "Remove a database from config file")
	globalFlag := flag.Bool("global", false, "Target global config (~/.claude/db-config.yaml) instead of project")

	// Connection parameter flags (used with --add-db / --edit-db)
	dsnFlag := flag.String("dsn", "", "Database connection string (DSN)")
	hostFlag := flag.String("host", "", "Database host")
	portFlag := flag.String("port", "", "Database port")
	userFlag := flag.String("user", "", "Database user")
	passwordFlag := flag.String("password", "", "Database password")
	dbnameFlag := flag.String("dbname", "", "Database name")
	driverFlag := flag.String("driver", "", "Database driver (postgres, mysql)")
	sslModeFlag := flag.String("ssl-mode", "", "SSL mode (default: disable)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: db_query.go [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// --read-write disables --read-only
	readOnly := *readOnlyFlag && !*readWriteFlag

	// Handle config management commands first
	switch {
	case *addDBFlag != "":
		handleAddDB(*addDBFlag, *globalFlag, *dsnFlag, *hostFlag, *portFlag, *userFlag, *passwordFlag, *dbnameFlag, *driverFlag, *sslModeFlag)
		return
	case *editDBFlag != "":
		handleEditDB(*editDBFlag, *globalFlag, *dsnFlag, *hostFlag, *portFlag, *userFlag, *passwordFlag, *dbnameFlag, *driverFlag, *sslModeFlag)
		return
	case *removeDBFlag != "":
		handleRemoveDB(*removeDBFlag, *globalFlag)
		return
	}

	creds := discoverAllCredentials()

	if *listFlag {
		printCredentialList(creds)
		return
	}

	if *dbFlag == "" {
		if len(creds) == 0 {
			log.Fatal("no database credentials found. Use --list to debug or --add-db to add one.")
		}
		fmt.Println("Available databases (use --db <name> to select):")
		for _, c := range creds {
			fmt.Printf("  %s (%s)\n", c.Name, c.Source)
		}
		os.Exit(0)
	}

	cred, ok := findCred(creds, *dbFlag)
	if !ok {
		fmt.Fprintf(os.Stderr, "database %q not found. Available:\n", *dbFlag)
		for _, c := range creds {
			fmt.Fprintf(os.Stderr, "  %s (%s)\n", c.Name, c.Source)
		}
		os.Exit(1)
	}

	if cred.Driver == "mongodb" {
		client, err := connectMongo(cred.DSN)
		if err != nil {
			log.Fatalf("cannot connect to MongoDB %q (%s): %v", cred.Name, cred.masked(), err)
		}
		defer client.Disconnect(context.Background())

		dbName := mongoDBName(cred)
		fmt.Printf("Connected to [%s] %s (source: %s, db: %s)\n\n", cred.Name, cred.masked(), cred.Source, dbName)

		switch {
		case *tablesFlag:
			listCollectionsMongo(client, dbName)
		case *describeFlag != "":
			describeCollectionMongo(client, dbName, *describeFlag, *rowsFlag)
		case *aggregateFlag != "":
			if *collectionFlag == "" {
				log.Fatal("--collection is required for MongoDB aggregation")
			}
			runAggregateMongo(client, dbName, *collectionFlag, *aggregateFlag, *rowsFlag, *formatFlag, *noHeaderFlag, readOnly)
		case *countFlag:
			if *collectionFlag == "" {
				log.Fatal("--collection is required for MongoDB count")
			}
			runCountMongo(client, dbName, *collectionFlag, *queryFlag, readOnly)
		case *distinctFlag != "":
			if *collectionFlag == "" {
				log.Fatal("--collection is required for MongoDB distinct")
			}
			runDistinctMongo(client, dbName, *collectionFlag, *distinctFlag, *queryFlag, *rowsFlag, readOnly)
		case *queryFlag != "":
			if *collectionFlag == "" {
				log.Fatal("--collection is required for MongoDB queries")
			}
			runQueryMongo(client, dbName, *collectionFlag, *queryFlag, *rowsFlag, *formatFlag, *noHeaderFlag, readOnly, *sortFlag, *projectFlag, *skipFlag)
		default:
			fmt.Println("Specify one of: --tables, --describe <collection>, --query <json> --collection <name>, --aggregate <pipeline>, --count, --distinct <field>")
			flag.Usage()
			os.Exit(1)
		}
	} else {
		db, err := sql.Open(cred.Driver, cred.DSN)
		if err != nil {
			log.Fatalf("cannot open DB %q: %v", cred.Name, err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			log.Fatalf("cannot connect to DB %q (%s): %v", cred.Name, cred.masked(), err)
		}
		fmt.Printf("Connected to [%s] %s (source: %s)\n\n", cred.Name, cred.masked(), cred.Source)

		switch {
		case *tablesFlag:
			listTables(db, cred.Driver)
		case *describeFlag != "":
			describeTable(db, cred.Driver, *describeFlag)
		case *queryFlag != "":
			runQuery(db, *queryFlag, *rowsFlag, *formatFlag, *noHeaderFlag, readOnly)
		default:
			fmt.Println("Specify one of: --tables, --describe <table>, --query <sql>")
			flag.Usage()
			os.Exit(1)
		}
	}
}

// ── Config file paths ─────────────────────────────────────────────────────────

func projectConfigPath() string {
	// Walk up from cwd to find .claude/ directory (usually at git root)
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join(".claude", "db-config.yaml")
	}
	for {
		candidate := filepath.Join(dir, ".claude", "db-config.yaml")
		claudeDir := filepath.Join(dir, ".claude")
		if info, err := os.Stat(claudeDir); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: use cwd/.claude/
	return filepath.Join(".claude", "db-config.yaml")
}

func globalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot determine home directory: %v", err)
	}
	return filepath.Join(home, ".claude", "db-config.yaml")
}

func configPathForScope(global bool) string {
	if global {
		return globalConfigPath()
	}
	return projectConfigPath()
}

// ── Config file I/O ───────────────────────────────────────────────────────────

func loadConfigFile(path string) (*dbConfigFile, error) {
	cfg := &dbConfigFile{Databases: map[string]*dbConfigEntry{}}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // empty config
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.Databases == nil {
		cfg.Databases = map[string]*dbConfigEntry{}
	}
	return cfg, nil
}

func saveConfigFile(path string, cfg *dbConfigFile) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ── Config CRUD handlers ──────────────────────────────────────────────────────

func handleAddDB(name string, global bool, dsn, host, port, user, password, dbname, driver, sslMode string) {
	path := configPathForScope(global)
	cfg, err := loadConfigFile(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if _, exists := cfg.Databases[name]; exists {
		log.Fatalf("database %q already exists in %s. Use --edit-db to modify it.", name, path)
	}

	entry := buildConfigEntry(dsn, host, port, user, password, dbname, driver, sslMode)
	if entry.DSN == "" && entry.Host == "" {
		log.Fatal("provide either --dsn or at least --host to add a database")
	}

	cfg.Databases[name] = entry
	if err := saveConfigFile(path, cfg); err != nil {
		log.Fatalf("save config: %v", err)
	}

	scope := "project"
	if global {
		scope = "global"
	}
	fmt.Printf("Added database %q to %s config (%s)\n", name, scope, path)
	printMaskedEntry(name, entry)
}

func handleEditDB(name string, global bool, dsn, host, port, user, password, dbname, driver, sslMode string) {
	path := configPathForScope(global)
	cfg, err := loadConfigFile(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	existing, exists := cfg.Databases[name]
	if !exists {
		log.Fatalf("database %q not found in %s. Use --add-db to create it.", name, path)
	}

	// Merge: only override fields that were explicitly provided
	if dsn != "" {
		existing.DSN = dsn
		// When DSN is set, clear individual fields
		existing.Host = ""
		existing.Port = ""
		existing.User = ""
		existing.Password = ""
		existing.DBName = ""
		existing.SSLMode = ""
	}
	if host != "" {
		existing.Host = host
	}
	if port != "" {
		existing.Port = port
	}
	if user != "" {
		existing.User = user
	}
	if password != "" {
		existing.Password = password
	}
	if dbname != "" {
		existing.DBName = dbname
	}
	if driver != "" {
		existing.Driver = driver
	}
	if sslMode != "" {
		existing.SSLMode = sslMode
	}

	if err := saveConfigFile(path, cfg); err != nil {
		log.Fatalf("save config: %v", err)
	}

	scope := "project"
	if global {
		scope = "global"
	}
	fmt.Printf("Updated database %q in %s config (%s)\n", name, scope, path)
	printMaskedEntry(name, existing)
}

func handleRemoveDB(name string, global bool) {
	path := configPathForScope(global)
	cfg, err := loadConfigFile(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if _, exists := cfg.Databases[name]; !exists {
		log.Fatalf("database %q not found in %s", name, path)
	}

	delete(cfg.Databases, name)
	if err := saveConfigFile(path, cfg); err != nil {
		log.Fatalf("save config: %v", err)
	}

	scope := "project"
	if global {
		scope = "global"
	}
	fmt.Printf("Removed database %q from %s config (%s)\n", name, scope, path)
}

func buildConfigEntry(dsn, host, port, user, password, dbname, driver, sslMode string) *dbConfigEntry {
	return &dbConfigEntry{
		DSN:      dsn,
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		Driver:   driver,
		SSLMode:  sslMode,
	}
}

func printMaskedEntry(name string, e *dbConfigEntry) {
	if e.DSN != "" {
		fmt.Printf("  %s: dsn=%s\n", name, maskDSN(e.DSN))
	} else {
		pw := "***"
		if e.Password == "" {
			pw = "(empty)"
		}
		drv := e.Driver
		if drv == "" {
			drv = guessDriverFromPort(e.Port)
		}
		fmt.Printf("  %s: %s://%s:%s@%s:%s/%s\n", name, drv, e.User, pw, e.Host, e.Port, e.DBName)
	}
}

// ── Credential discovery (merged) ─────────────────────────────────────────────

// discoverAllCredentials merges credentials from env vars, global config, and
// project config. Priority: project config > global config > env vars.
func discoverAllCredentials() []dbCredential {
	merged := map[string]dbCredential{}

	// Layer 1: env vars (lowest priority)
	for _, c := range discoverEnvCredentials() {
		c.Source = "env"
		merged[c.Name] = c
	}

	// Layer 2: global config
	globalPath := globalConfigPath()
	if globalCfg, err := loadConfigFile(globalPath); err == nil {
		for name, entry := range globalCfg.Databases {
			c := configEntryToCredential(name, entry, "global")
			merged[name] = c
		}
	}

	// Layer 3: project config (highest priority)
	projectPath := projectConfigPath()
	if projectPath != globalPath { // avoid double-loading if same path
		if projectCfg, err := loadConfigFile(projectPath); err == nil {
			for name, entry := range projectCfg.Databases {
				c := configEntryToCredential(name, entry, "project")
				merged[name] = c
			}
		}
	}

	// Convert to sorted slice
	var creds []dbCredential
	for _, c := range merged {
		creds = append(creds, c)
	}
	sort.Slice(creds, func(i, j int) bool { return creds[i].Name < creds[j].Name })
	return creds
}

func configEntryToCredential(name string, e *dbConfigEntry, source string) dbCredential {
	if e.DSN != "" {
		driver := e.Driver
		if driver == "" {
			driver = guessDriverFromDSN(e.DSN)
		}
		normDSN := normalizeDSN(driver, e.DSN)
		// Try to extract host/user/dbname from DSN for display
		host, port, user, dbname := parseDSNParts(e.DSN)
		return dbCredential{
			Name: name, Driver: driver, DSN: normDSN,
			Host: host, Port: port, User: user, DBName: dbname,
			Source: source,
		}
	}

	driver := e.Driver
	if driver == "" {
		driver = guessDriverFromPort(e.Port)
	}
	ssl := e.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	dsn := buildDSN(driver, e.User, e.Password, e.Host, e.Port, e.DBName, ssl)
	return dbCredential{
		Name: name, Driver: driver, DSN: dsn,
		Host: e.Host, Port: e.Port, User: e.User, DBName: e.DBName,
		Source: source,
	}
}

// parseDSNParts extracts host, port, user, dbname from a URL-style DSN for display.
func parseDSNParts(dsn string) (host, port, user, dbname string) {
	// Handle postgres://user:pass@host:port/dbname?...
	idx := strings.Index(dsn, "://")
	if idx == -1 {
		return
	}
	rest := dsn[idx+3:]

	// Split user:pass@host:port/dbname
	atIdx := strings.LastIndex(rest, "@")
	if atIdx == -1 {
		return
	}
	userInfo := rest[:atIdx]
	hostPath := rest[atIdx+1:]

	// Extract user (before :)
	if colonIdx := strings.Index(userInfo, ":"); colonIdx != -1 {
		user = userInfo[:colonIdx]
	} else {
		user = userInfo
	}

	// Extract host:port/dbname
	slashIdx := strings.Index(hostPath, "/")
	hostPort := hostPath
	if slashIdx != -1 {
		hostPort = hostPath[:slashIdx]
		dbPart := hostPath[slashIdx+1:]
		if qIdx := strings.Index(dbPart, "?"); qIdx != -1 {
			dbname = dbPart[:qIdx]
		} else {
			dbname = dbPart
		}
	}

	if colonIdx := strings.LastIndex(hostPort, ":"); colonIdx != -1 {
		host = hostPort[:colonIdx]
		port = hostPort[colonIdx+1:]
	} else {
		host = hostPort
	}
	return
}

// discoverEnvCredentials discovers credentials from environment variables only.
func discoverEnvCredentials() []dbCredential {
	var creds []dbCredential
	seen := map[string]bool{}

	// Pattern A: component-based (<PREFIX>_DB_HOST / DB_HOST for main)
	for prefix, name := range componentBasedPrefixes() {
		host := envKey(prefix, "DB_HOST")
		if host == "" {
			continue
		}
		port := envKey(prefix, "DB_PORT")
		user := envKey(prefix, "DB_USER")
		pass := envKey(prefix, "DB_PASSWORD")
		dbname := envKey(prefix, "DB_NAME")
		ssl := envKey(prefix, "DB_SSL_MODE")
		if ssl == "" {
			ssl = "disable"
		}
		driver := envKey(prefix, "DB_DRIVER")
		if driver == "" {
			driver = guessDriverFromPort(port)
		}
		dsn := buildDSN(driver, user, pass, host, port, dbname, ssl)
		if !seen[name] {
			creds = append(creds, dbCredential{
				Name: name, Driver: driver, DSN: dsn,
				Host: host, Port: port, User: user, DBName: dbname,
			})
			seen[name] = true
		}
	}

	// Pattern B: DSN-based (<NAME>_DSN, DATABASE_URL, DATABASE_URL_<NAME>, MONGODB_URI)
	for _, env := range os.Environ() {
		k, v, _ := strings.Cut(env, "=")
		if v == "" {
			continue
		}
		var name string
		switch {
		case k == "DATABASE_URL":
			name = "default"
		case strings.HasPrefix(k, "DATABASE_URL_"):
			name = strings.ToLower(k[len("DATABASE_URL_"):])
		case k == "MONGODB_URI" || k == "MONGO_URI" || k == "MONGO_URL":
			name = "mongodb"
		case strings.HasSuffix(k, "_DSN"):
			name = strings.ToLower(k[:len(k)-4])
		default:
			continue
		}
		if seen[name] {
			continue
		}
		driver := guessDriverFromDSN(v)
		normDSN := normalizeDSN(driver, v)
		creds = append(creds, dbCredential{Name: name, Driver: driver, DSN: normDSN})
		seen[name] = true
	}

	sort.Slice(creds, func(i, j int) bool { return creds[i].Name < creds[j].Name })
	return creds
}

// ── Env helpers ───────────────────────────────────────────────────────────────

func componentBasedPrefixes() map[string]string {
	result := map[string]string{}
	if os.Getenv("DB_HOST") != "" {
		result[""] = "main"
	}
	for _, env := range os.Environ() {
		k, _, _ := strings.Cut(env, "=")
		if !strings.HasSuffix(k, "_DB_HOST") {
			continue
		}
		prefix := k[:len(k)-len("_DB_HOST")]
		result[prefix] = strings.ToLower(prefix)
	}
	return result
}

func envKey(prefix, key string) string {
	if prefix == "" {
		return os.Getenv(key)
	}
	return os.Getenv(prefix + "_" + key)
}

func guessDriverFromPort(port string) string {
	switch port {
	case "3306":
		return "mysql"
	case "27017":
		return "mongodb"
	default:
		return "postgres"
	}
}

func guessDriverFromDSN(dsn string) string {
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "mysql://"):
		return "mysql"
	case strings.HasPrefix(lower, "mongodb://"), strings.HasPrefix(lower, "mongodb+srv://"):
		return "mongodb"
	default:
		return "postgres"
	}
}

func buildDSN(driver, user, pass, host, port, dbname, ssl string) string {
	switch driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, dbname)
	case "mongodb":
		dsn := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pass, host, port, dbname)
		if ssl != "" && ssl != "disable" {
			dsn += "?tls=true"
		}
		return dsn
	default:
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, pass, dbname, ssl)
	}
}

// normalizeDSN converts URL-style DSNs to driver-native format.
func normalizeDSN(driver, dsn string) string {
	if driver == "mongodb" {
		return dsn // MongoDB DSNs are already native format
	}
	if driver == "mysql" {
		s := strings.TrimPrefix(dsn, "mysql://")
		userInfo, hostPath, ok := strings.Cut(s, "@")
		if !ok {
			return dsn
		}
		host, path, _ := strings.Cut(hostPath, "/")
		return fmt.Sprintf("%s@tcp(%s)/%s", userInfo, host, path)
	}
	return dsn
}

func maskDSN(dsn string) string {
	if idx := strings.Index(dsn, "://"); idx != -1 {
		rest := dsn[idx+3:]
		if at := strings.LastIndex(rest, "@"); at != -1 {
			userInfo := rest[:at]
			if colon := strings.Index(userInfo, ":"); colon != -1 {
				userInfo = userInfo[:colon] + ":***"
			}
			return dsn[:idx+3] + userInfo + rest[at:]
		}
	}
	parts := strings.Fields(dsn)
	for i, p := range parts {
		if strings.HasPrefix(strings.ToLower(p), "password=") {
			parts[i] = "password=***"
		}
	}
	return strings.Join(parts, " ")
}

func findCred(creds []dbCredential, name string) (dbCredential, bool) {
	for _, c := range creds {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return dbCredential{}, false
}

func printCredentialList(creds []dbCredential) {
	if len(creds) == 0 {
		fmt.Println("No databases detected.")
		fmt.Println("\nAdd a database with:")
		fmt.Println("  --add-db <name> --dsn 'postgres://user:pass@host:5432/dbname'")
		fmt.Println("  --add-db <name> --host localhost --port 5432 --user postgres --password secret --dbname mydb")
		fmt.Println("  --add-db <name> --global  (to save in global config)")
		fmt.Println("\nOr set environment variables:")
		fmt.Println("  DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME       → name: main")
		fmt.Println("  <PREFIX>_DB_HOST / ...                                    → name: <prefix>")
		fmt.Println("  DATABASE_URL=postgres://...                               → name: default")
		fmt.Println("  <NAME>_DSN=postgres://...                                 → name: <name>")
		return
	}

	fmt.Printf("Detected %d database(s):\n\n", len(creds))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDRIVER\tSOURCE\tCONNECTION")
	fmt.Fprintln(w, "----\t------\t------\t----------")
	for _, c := range creds {
		conn := c.masked()
		if c.Host == "" {
			conn = maskDSN(c.DSN)
		}
		source := c.Source
		if source == "" {
			source = "env"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Name, c.Driver, source, conn)
	}
	w.Flush()

	fmt.Printf("\nConfig files:\n")
	fmt.Printf("  Project: %s\n", projectConfigPath())
	fmt.Printf("  Global:  %s\n", globalConfigPath())
}

// ── Database operations ───────────────────────────────────────────────────────

func listTables(db *sql.DB, driver string) {
	var query string
	switch driver {
	case "mysql":
		query = "SHOW TABLES"
	default: // postgres
		query = `SELECT table_schema || '.' || table_name
		         FROM information_schema.tables
		         WHERE table_schema NOT IN ('pg_catalog','information_schema')
		         ORDER BY table_schema, table_name`
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("cannot list tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			log.Fatalf("scan error: %v", err)
		}
		tables = append(tables, t)
	}

	fmt.Printf("Tables (%d):\n", len(tables))
	for _, t := range tables {
		fmt.Printf("  %s\n", t)
	}
}

func describeTable(db *sql.DB, driver, table string) {
	var (
		query string
		args  []any
	)

	schema, tbl, hasSchema := strings.Cut(table, ".")

	switch driver {
	case "mysql":
		if hasSchema {
			query = `SELECT column_name, data_type, is_nullable, column_default
			         FROM information_schema.columns
			         WHERE table_schema=? AND table_name=?
			         ORDER BY ordinal_position`
			args = []any{schema, tbl}
		} else {
			query = `SELECT column_name, data_type, is_nullable, column_default
			         FROM information_schema.columns
			         WHERE table_name=?
			         ORDER BY ordinal_position`
			args = []any{table}
		}
	default: // postgres
		if hasSchema {
			query = `SELECT column_name, data_type, is_nullable, column_default
			         FROM information_schema.columns
			         WHERE table_schema=$1 AND table_name=$2
			         ORDER BY ordinal_position`
			args = []any{schema, tbl}
		} else {
			query = `SELECT column_name, data_type, is_nullable, column_default
			         FROM information_schema.columns
			         WHERE table_name=$1
			         ORDER BY ordinal_position`
			args = []any{table}
		}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Fatalf("cannot describe table %q: %v", table, err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	fmt.Printf("Table: %s\n\n", table)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(cols, "\t"))
	fmt.Fprintln(w, strings.Repeat("-\t", len(cols)))
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			log.Fatalf("scan error: %v", err)
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = anyStr(v)
		}
		fmt.Fprintln(w, strings.Join(parts, "\t"))
	}
	w.Flush()
}

func runQuery(db *sql.DB, query string, maxRows int, format string, noHeader bool, readOnly bool) {
	var rows *sql.Rows
	var err error
	if readOnly {
		tx, txErr := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		if txErr != nil {
			log.Fatalf("cannot start read-only transaction: %v", txErr)
		}
		defer tx.Rollback()
		rows, err = tx.Query(query)
	} else {
		rows, err = db.Query(query)
	}
	if err != nil {
		log.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		log.Fatalf("cannot get columns: %v", err)
	}

	var records [][]string
	count := 0
	truncated := false

	for rows.Next() {
		if count >= maxRows {
			truncated = true
			break
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			log.Fatalf("scan error: %v", err)
		}
		row := make([]string, len(vals))
		for i, v := range vals {
			row[i] = anyStr(v)
		}
		records = append(records, row)
		count++
	}

	switch format {
	case "csv":
		printCSV(cols, records, noHeader)
	case "json":
		printJSON(cols, records)
	default:
		printTable(cols, records, noHeader)
	}

	fmt.Printf("\n%d row(s) returned", count)
	if truncated {
		fmt.Printf(" (truncated — use --rows to show more)")
	}
	fmt.Println()
}

func printTable(cols []string, records [][]string, noHeader bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if !noHeader {
		fmt.Fprintln(w, strings.Join(cols, "\t"))
		fmt.Fprintln(w, strings.Repeat("-\t", len(cols)))
	}
	for _, row := range records {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func printCSV(cols []string, records [][]string, noHeader bool) {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if !noHeader {
		_ = w.Write(cols)
	}
	for _, row := range records {
		_ = w.Write(row)
	}
}

func printJSON(cols []string, records [][]string) {
	var out []map[string]string
	for _, row := range records {
		m := make(map[string]string, len(cols))
		for i, col := range cols {
			if i < len(row) {
				m[col] = row[i]
			}
		}
		out = append(out, m)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func anyStr(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ── MongoDB operations ────────────────────────────────────────────────────────

func connectMongo(dsn string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(dsn).SetTimeout(10 * time.Second))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client, nil
}

func mongoDBName(cred dbCredential) string {
	if cred.DBName != "" {
		return cred.DBName
	}
	_, _, _, dbname := parseDSNParts(cred.DSN)
	if dbname != "" {
		return dbname
	}
	return "test" // MongoDB default
}

func listCollectionsMongo(client *mongo.Client, dbName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	names, err := client.Database(dbName).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		log.Fatalf("cannot list collections: %v", err)
	}
	sort.Strings(names)
	fmt.Printf("Collections (%d):\n", len(names))
	for _, name := range names {
		fmt.Printf("  %s\n", name)
	}
}

func describeCollectionMongo(client *mongo.Client, dbName, collection string, maxSample int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	coll := client.Database(dbName).Collection(collection)

	opts := options.Find().SetLimit(int64(maxSample))
	cursor, err := coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		log.Fatalf("cannot sample collection %q: %v", collection, err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		log.Fatalf("cannot read documents: %v", err)
	}

	if len(docs) == 0 {
		fmt.Printf("Collection %q is empty\n", collection)
		return
	}

	// Collect all field names and their types
	type fieldInfo struct {
		Types map[string]int
	}
	fieldsMap := map[string]*fieldInfo{}
	var fieldOrder []string

	for _, doc := range docs {
		for k, v := range doc {
			if _, exists := fieldsMap[k]; !exists {
				fieldsMap[k] = &fieldInfo{Types: map[string]int{}}
				fieldOrder = append(fieldOrder, k)
			}
			typeName := bsonTypeName(v)
			fieldsMap[k].Types[typeName]++
		}
	}
	sort.Strings(fieldOrder)

	fmt.Printf("Collection: %s (sampled %d documents)\n\n", collection, len(docs))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FIELD\tTYPE(S)\tPRESENT")
	fmt.Fprintln(w, "-----\t-------\t-------")
	for _, name := range fieldOrder {
		fi := fieldsMap[name]
		var types []string
		for t, c := range fi.Types {
			if c == len(docs) {
				types = append(types, t)
			} else {
				types = append(types, fmt.Sprintf("%s(%d)", t, c))
			}
		}
		sort.Strings(types)
		total := 0
		for _, c := range fi.Types {
			total += c
		}
		pct := fmt.Sprintf("%d/%d", total, len(docs))
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, strings.Join(types, ", "), pct)
	}
	w.Flush()
}

func bsonTypeName(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case bson.ObjectID:
		return "ObjectId"
	case string:
		return "string"
	case int32:
		return "int32"
	case int64:
		return "int64"
	case float64:
		return "double"
	case bool:
		return "bool"
	case bson.DateTime:
		return "date"
	case bson.M:
		return "object"
	case bson.A:
		return "array"
	case bson.Binary:
		return "binary"
	case bson.Decimal128:
		return "decimal128"
	case bson.Timestamp:
		return "timestamp"
	case bson.Regex:
		return "regex"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// unsafeMongoOps lists MongoDB operators that execute server-side JavaScript.
var unsafeMongoOps = map[string]bool{
	"$where":       true,
	"$function":    true,
	"$accumulator": true,
}

// rejectUnsafeMongoOps blocks MongoDB operators that execute server-side
// JavaScript when read-only mode is active. Recursively walks nested documents.
func rejectUnsafeMongoOps(filter bson.D) {
	rejectUnsafeKeys(filter)
}

// rejectUnsafeKeys recursively walks BSON structures to detect unsafe operators
// nested inside $and, $or, $nor, $not, or any other compound expression.
func rejectUnsafeKeys(v interface{}) {
	switch val := v.(type) {
	case bson.D:
		for _, elem := range val {
			if unsafeMongoOps[elem.Key] {
				log.Fatalf("--read-only: %s operator is not allowed (executes server-side JavaScript)", elem.Key)
			}
			rejectUnsafeKeys(elem.Value)
		}
	case bson.A:
		for _, item := range val {
			rejectUnsafeKeys(item)
		}
	case bson.M:
		for k, item := range val {
			if unsafeMongoOps[k] {
				log.Fatalf("--read-only: %s operator is not allowed (executes server-side JavaScript)", k)
			}
			rejectUnsafeKeys(item)
		}
	}
}

// parseMongoFilter parses a JSON string into a bson.D filter.
func parseMongoFilter(query string) bson.D {
	if query == "" || query == "{}" {
		return bson.D{}
	}
	var filter bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &filter); err != nil {
		log.Fatalf("invalid JSON filter: %v", err)
	}
	return filter
}

func runQueryMongo(client *mongo.Client, dbName, collection, query string, maxRows int, format string, noHeader bool, readOnly bool, sortJSON string, projectJSON string, skip int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	coll := client.Database(dbName).Collection(collection)

	filter := parseMongoFilter(query)

	if readOnly {
		rejectUnsafeMongoOps(filter)
	}

	opts := options.Find().SetLimit(int64(maxRows))

	if skip > 0 {
		opts.SetSkip(int64(skip))
	}

	if sortJSON != "" {
		var sortDoc bson.D
		if err := bson.UnmarshalExtJSON([]byte(sortJSON), false, &sortDoc); err != nil {
			log.Fatalf("invalid --sort JSON: %v", err)
		}
		if readOnly {
			rejectUnsafeKeys(sortDoc)
		}
		opts.SetSort(sortDoc)
	}

	if projectJSON != "" {
		var projDoc bson.D
		if err := bson.UnmarshalExtJSON([]byte(projectJSON), false, &projDoc); err != nil {
			log.Fatalf("invalid --project JSON: %v", err)
		}
		if readOnly {
			rejectUnsafeKeys(projDoc)
		}
		opts.SetProjection(projDoc)
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		log.Fatalf("query error: %v", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		log.Fatalf("cannot read results: %v", err)
	}

	printMongoDocs(docs, maxRows, format, noHeader)
}

// runAggregateMongo executes a MongoDB aggregation pipeline.
func runAggregateMongo(client *mongo.Client, dbName, collection, pipeline string, maxRows int, format string, noHeader bool, readOnly bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	coll := client.Database(dbName).Collection(collection)

	var stages bson.A
	if err := bson.UnmarshalExtJSON([]byte(pipeline), false, &stages); err != nil {
		log.Fatalf("invalid aggregation pipeline JSON: %v", err)
	}

	// In read-only mode, reject stages that modify data
	if readOnly {
		rejectUnsafeAggregateStages(stages)
	}

	// Always append a $limit stage to enforce maxRows cap, even if user
	// provided their own $limit (MongoDB uses the smaller of the two).
	stages = append(stages, bson.D{{Key: "$limit", Value: int64(maxRows)}})

	cursor, err := coll.Aggregate(ctx, stages)
	if err != nil {
		log.Fatalf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		log.Fatalf("cannot read aggregation results: %v", err)
	}

	printMongoDocs(docs, maxRows, format, noHeader)
}

// rejectUnsafeAggregateStages blocks aggregation stages that modify data or
// execute server-side JavaScript when read-only mode is active.
func rejectUnsafeAggregateStages(stages bson.A) {
	unsafeStages := map[string]bool{
		"$out":   true,
		"$merge": true,
	}
	for _, s := range stages {
		if stage, ok := s.(bson.D); ok {
			for _, elem := range stage {
				if unsafeStages[elem.Key] {
					log.Fatalf("--read-only: %s stage is not allowed (modifies data)", elem.Key)
				}
			}
		}
		// Recursively check for $function/$accumulator/$where inside stage values
		rejectUnsafeKeys(s)
	}
}

// runCountMongo counts documents matching a filter.
func runCountMongo(client *mongo.Client, dbName, collection, query string, readOnly bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	coll := client.Database(dbName).Collection(collection)

	filter := parseMongoFilter(query)

	if readOnly {
		rejectUnsafeMongoOps(filter)
	}

	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		log.Fatalf("count error: %v", err)
	}

	fmt.Printf("%d\n", count)
}

// runDistinctMongo gets distinct values for a field.
func runDistinctMongo(client *mongo.Client, dbName, collection, field, query string, maxRows int, readOnly bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	coll := client.Database(dbName).Collection(collection)

	filter := parseMongoFilter(query)

	if readOnly {
		rejectUnsafeMongoOps(filter)
	}

	result := coll.Distinct(ctx, field, filter)
	if result.Err() != nil {
		log.Fatalf("distinct error: %v", result.Err())
	}

	var values []interface{}
	if err := result.Decode(&values); err != nil {
		log.Fatalf("distinct decode error: %v", err)
	}

	displayed := len(values)
	if maxRows > 0 && displayed > maxRows {
		displayed = maxRows
	}
	for i := 0; i < displayed; i++ {
		fmt.Println(mongoValueStr(values[i]))
	}
	if displayed < len(values) {
		fmt.Printf("\n%d distinct value(s) shown out of %d (use --rows to show more)\n", displayed, len(values))
	} else {
		fmt.Printf("\n%d distinct value(s)\n", len(values))
	}
}

// printMongoDocs formats and prints MongoDB documents.
func printMongoDocs(docs []bson.M, maxRows int, format string, noHeader bool) {
	if len(docs) == 0 {
		fmt.Println("No documents found.")
		return
	}

	// Collect all unique keys to form columns
	keySet := map[string]bool{}
	var keyOrder []string
	for _, doc := range docs {
		for k := range doc {
			if !keySet[k] {
				keySet[k] = true
				keyOrder = append(keyOrder, k)
			}
		}
	}
	sort.Strings(keyOrder)

	// Convert to records
	var records [][]string
	for _, doc := range docs {
		row := make([]string, len(keyOrder))
		for i, k := range keyOrder {
			if v, ok := doc[k]; ok {
				row[i] = mongoValueStr(v)
			} else {
				row[i] = "NULL"
			}
		}
		records = append(records, row)
	}

	switch format {
	case "csv":
		printCSV(keyOrder, records, noHeader)
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(docs)
	default:
		printTable(keyOrder, records, noHeader)
	}

	fmt.Printf("\n%d document(s) returned", len(docs))
	if len(docs) >= maxRows {
		fmt.Printf(" (limit reached — use --rows to show more)")
	}
	fmt.Println()
}

func mongoValueStr(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case bson.ObjectID:
		return val.Hex()
	case bson.DateTime:
		return val.Time().Format(time.RFC3339)
	case bson.M:
		b, _ := json.Marshal(val)
		return string(b)
	case bson.A:
		b, _ := json.Marshal(val)
		return string(b)
	case bson.Binary:
		return fmt.Sprintf("Binary(%d bytes)", len(val.Data))
	case bson.Decimal128:
		return val.String()
	case bson.Timestamp:
		return fmt.Sprintf("Timestamp(%d, %d)", val.T, val.I)
	default:
		return fmt.Sprintf("%v", val)
	}
}
