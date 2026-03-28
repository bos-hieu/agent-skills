package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// ── Data structures ───────────────────────────────────────────────────────────

// HostEntry represents a single SSH host entry in the YAML config file.
type HostEntry struct {
	Host    string            `yaml:"host"`
	Port    int               `yaml:"port,omitempty"`
	User    string            `yaml:"user,omitempty"`
	Bastion string            `yaml:"bastion,omitempty"`
	Tags    []string          `yaml:"tags,omitempty,flow"`
	Tunnels map[string]string `yaml:"tunnels,omitempty"`
}

// ConfigFile represents the YAML config file structure.
type ConfigFile struct {
	Hosts map[string]*HostEntry `yaml:"hosts"`
}

// ── Config file I/O ───────────────────────────────────────────────────────────

func loadConfig(path string) (*ConfigFile, error) {
	cfg := &ConfigFile{Hosts: map[string]*HostEntry{}}

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
	if cfg.Hosts == nil {
		cfg.Hosts = map[string]*HostEntry{}
	}
	return cfg, nil
}

func saveConfig(path string, cfg *ConfigFile) error {
	// Ensure parent directory exists with 0700 permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
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

// findProjectConfig walks up from cwd looking for .claude/ directory,
// returns .claude/ssh-config.yaml path.
func findProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join(".claude", "ssh-config.yaml")
	}
	for {
		claudeDir := filepath.Join(dir, ".claude")
		if info, err := os.Stat(claudeDir); err == nil && info.IsDir() {
			return filepath.Join(dir, ".claude", "ssh-config.yaml")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: use cwd/.claude/
	return filepath.Join(".claude", "ssh-config.yaml")
}

// globalConfigPath returns ~/.claude/ssh-config.yaml.
func globalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".claude", "ssh-config.yaml")
}

// configPathForScope returns the config path for the given scope.
func configPathForScope(scope string) string {
	if scope == "global" {
		return globalConfigPath()
	}
	return findProjectConfig()
}

// mergedHosts returns all hosts merged (project overrides global) along with
// a source tracking map (host name -> "project" or "global").
func mergedHosts() (map[string]*HostEntry, map[string]string) {
	hosts := map[string]*HostEntry{}
	sources := map[string]string{}

	// Layer 1: global config (lower priority)
	globalPath := globalConfigPath()
	if globalCfg, err := loadConfig(globalPath); err == nil {
		for name, entry := range globalCfg.Hosts {
			hosts[name] = entry
			sources[name] = "global"
		}
	}

	// Layer 2: project config (higher priority, overrides global)
	projectPath := findProjectConfig()
	if projectPath != globalPath {
		if projectCfg, err := loadConfig(projectPath); err == nil {
			for name, entry := range projectCfg.Hosts {
				hosts[name] = entry
				sources[name] = "project"
			}
		}
	}

	return hosts, sources
}

// ── CLI ───────────────────────────────────────────────────────────────────────

func main() {
	// Command flags
	addHostFlag := flag.String("add-host", "", "Add a new SSH host entry")
	editHostFlag := flag.String("edit-host", "", "Edit an existing SSH host entry")
	removeHostFlag := flag.String("remove-host", "", "Remove an SSH host entry")
	listFlag := flag.Bool("list", false, "List all SSH hosts in a table")
	getFlag := flag.String("get", "", "Get a host as KEY=VALUE lines")
	getByTagFlag := flag.String("get-by-tag", "", "Get hosts matching a tag")
	showConfigFlag := flag.Bool("show-config", false, "Print both project and global configs")
	addTunnelFlag := flag.String("add-tunnel", "", "Add a tunnel to a host")
	removeTunnelFlag := flag.String("remove-tunnel", "", "Remove a tunnel from a host")

	// Host parameter flags (used with --add-host / --edit-host)
	hostFlag := flag.String("host", "", "SSH hostname or IP address")
	portFlag := flag.Int("port", 0, "SSH port (default: 22)")
	userFlag := flag.String("user", "", "SSH username")
	bastionFlag := flag.String("bastion", "", "Bastion/jump host")
	tagsFlag := flag.String("tags", "", "Comma-separated tags")

	// Tunnel parameter flags
	tunnelNameFlag := flag.String("tunnel-name", "", "Tunnel name (used with --add-tunnel / --remove-tunnel)")
	tunnelMappingFlag := flag.String("tunnel-mapping", "", "Tunnel mapping (e.g. 8080:localhost:80)")

	// Scope flag
	scopeFlag := flag.String("scope", "project", "Config scope: global or project")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: config.go [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	switch {
	case *addHostFlag != "":
		handleAddHost(*addHostFlag, *scopeFlag, *hostFlag, *portFlag, *userFlag, *bastionFlag, *tagsFlag)
	case *editHostFlag != "":
		handleEditHost(*editHostFlag, *scopeFlag, *hostFlag, *portFlag, *userFlag, *bastionFlag, *tagsFlag)
	case *removeHostFlag != "":
		handleRemoveHost(*removeHostFlag, *scopeFlag)
	case *listFlag:
		handleList()
	case *getFlag != "":
		handleGet(*getFlag)
	case *getByTagFlag != "":
		handleGetByTag(*getByTagFlag)
	case *showConfigFlag:
		handleShowConfig()
	case *addTunnelFlag != "":
		handleAddTunnel(*addTunnelFlag, *scopeFlag, *tunnelNameFlag, *tunnelMappingFlag)
	case *removeTunnelFlag != "":
		handleRemoveTunnel(*removeTunnelFlag, *scopeFlag, *tunnelNameFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}

// ── Host CRUD handlers ────────────────────────────────────────────────────────

func handleAddHost(name, scope, host string, port int, user, bastion, tags string) {
	if host == "" {
		fmt.Fprintf(os.Stderr, "error: --host is required when adding a host\n")
		os.Exit(1)
	}

	path := configPathForScope(scope)
	cfg, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
		os.Exit(1)
	}

	if _, exists := cfg.Hosts[name]; exists {
		fmt.Fprintf(os.Stderr, "error: host %q already exists in %s. Use --edit-host to modify it.\n", name, path)
		os.Exit(1)
	}

	entry := &HostEntry{
		Host: host,
		Port: port,
		User: user,
		Bastion: bastion,
	}
	if tags != "" {
		entry.Tags = parseTags(tags)
	}

	cfg.Hosts[name] = entry
	if err := saveConfig(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added host %q to %s config (%s)\n", name, scope, path)
}

func handleEditHost(name, scope, host string, port int, user, bastion, tags string) {
	path := configPathForScope(scope)
	cfg, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
		os.Exit(1)
	}

	existing, exists := cfg.Hosts[name]
	if !exists {
		fmt.Fprintf(os.Stderr, "error: host %q not found in %s. Use --add-host to create it.\n", name, path)
		os.Exit(1)
	}

	// Merge: only override fields that were explicitly provided
	if host != "" {
		existing.Host = host
	}
	if port != 0 {
		existing.Port = port
	}
	if user != "" {
		existing.User = user
	}
	if bastion != "" {
		existing.Bastion = bastion
	}
	if tags != "" {
		existing.Tags = parseTags(tags)
	}

	if err := saveConfig(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated host %q in %s config (%s)\n", name, scope, path)
}

func handleRemoveHost(name, scope string) {
	path := configPathForScope(scope)
	cfg, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
		os.Exit(1)
	}

	if _, exists := cfg.Hosts[name]; !exists {
		fmt.Fprintf(os.Stderr, "error: host %q not found in %s\n", name, path)
		os.Exit(1)
	}

	delete(cfg.Hosts, name)
	if err := saveConfig(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed host %q from %s config (%s)\n", name, scope, path)
}

// ── List / Get handlers ───────────────────────────────────────────────────────

func handleList() {
	hosts, sources := mergedHosts()

	if len(hosts) == 0 {
		fmt.Println("No SSH hosts configured.")
		return
	}

	// Sort host names alphabetically
	names := make([]string, 0, len(hosts))
	for name := range hosts {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tHOST\tPORT\tUSER\tBASTION\tTAGS\tSOURCE")
	for _, name := range names {
		entry := hosts[name]
		port := effectivePort(entry.Port)
		tagsStr := strings.Join(entry.Tags, ",")
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			name, entry.Host, port, entry.User, entry.Bastion, tagsStr, sources[name])
	}
	w.Flush()
}

func handleGet(name string) {
	hosts, _ := mergedHosts()

	entry, exists := hosts[name]
	if !exists {
		fmt.Fprintf(os.Stderr, "error: host %q not found\n", name)
		os.Exit(1)
	}

	port := effectivePort(entry.Port)
	fmt.Printf("NAME=%s\n", name)
	fmt.Printf("HOST=%s\n", entry.Host)
	fmt.Printf("PORT=%d\n", port)
	fmt.Printf("USER=%s\n", entry.User)
	fmt.Printf("BASTION=%s\n", entry.Bastion)
	fmt.Printf("TAGS=%s\n", strings.Join(entry.Tags, ","))

	// Output tunnel mappings
	if len(entry.Tunnels) > 0 {
		tunnelNames := make([]string, 0, len(entry.Tunnels))
		for tn := range entry.Tunnels {
			tunnelNames = append(tunnelNames, tn)
		}
		sort.Strings(tunnelNames)
		for _, tn := range tunnelNames {
			fmt.Printf("TUNNEL_%s=%s\n", strings.ToUpper(tn), entry.Tunnels[tn])
		}
	}
}

func handleGetByTag(tag string) {
	hosts, sources := mergedHosts()

	// Collect matching hosts
	type match struct {
		name   string
		entry  *HostEntry
		source string
	}
	var matches []match
	for name, entry := range hosts {
		for _, t := range entry.Tags {
			if t == tag {
				matches = append(matches, match{name: name, entry: entry, source: sources[name]})
				break
			}
		}
	}

	if len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "error: no hosts found with tag %q\n", tag)
		os.Exit(1)
	}

	// Sort alphabetically by name
	sort.Slice(matches, func(i, j int) bool { return matches[i].name < matches[j].name })

	for _, m := range matches {
		port := effectivePort(m.entry.Port)
		fmt.Printf("NAME=%s HOST=%s PORT=%d USER=%s BASTION=%s\n",
			m.name, m.entry.Host, port, m.entry.User, m.entry.Bastion)
	}
}

// ── Show config handler ───────────────────────────────────────────────────────

func handleShowConfig() {
	projectPath := findProjectConfig()
	globalPath := globalConfigPath()

	fmt.Printf("=== Project config: %s ===\n", projectPath)
	printConfigFile(projectPath)

	fmt.Println()

	fmt.Printf("=== Global config: %s ===\n", globalPath)
	printConfigFile(globalPath)
}

func printConfigFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(not found)")
			return
		}
		fmt.Fprintf(os.Stderr, "error: reading %s: %v\n", path, err)
		return
	}
	if len(data) == 0 {
		fmt.Println("(empty)")
		return
	}
	fmt.Print(string(data))
}

// ── Tunnel handlers ───────────────────────────────────────────────────────────

func handleAddTunnel(hostName, scope, tunnelName, tunnelMapping string) {
	if tunnelName == "" {
		fmt.Fprintf(os.Stderr, "error: --tunnel-name is required\n")
		os.Exit(1)
	}
	if tunnelMapping == "" {
		fmt.Fprintf(os.Stderr, "error: --tunnel-mapping is required\n")
		os.Exit(1)
	}

	path := configPathForScope(scope)
	cfg, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
		os.Exit(1)
	}

	entry, exists := cfg.Hosts[hostName]
	if !exists {
		fmt.Fprintf(os.Stderr, "error: host %q not found in %s\n", hostName, path)
		os.Exit(1)
	}

	if entry.Tunnels == nil {
		entry.Tunnels = map[string]string{}
	}
	entry.Tunnels[tunnelName] = tunnelMapping

	if err := saveConfig(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added tunnel %q (%s) to host %q in %s config (%s)\n",
		tunnelName, tunnelMapping, hostName, scope, path)
}

func handleRemoveTunnel(hostName, scope, tunnelName string) {
	if tunnelName == "" {
		fmt.Fprintf(os.Stderr, "error: --tunnel-name is required\n")
		os.Exit(1)
	}

	path := configPathForScope(scope)
	cfg, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
		os.Exit(1)
	}

	entry, exists := cfg.Hosts[hostName]
	if !exists {
		fmt.Fprintf(os.Stderr, "error: host %q not found in %s\n", hostName, path)
		os.Exit(1)
	}

	if entry.Tunnels == nil {
		fmt.Fprintf(os.Stderr, "error: host %q has no tunnels\n", hostName)
		os.Exit(1)
	}

	if _, exists := entry.Tunnels[tunnelName]; !exists {
		fmt.Fprintf(os.Stderr, "error: tunnel %q not found on host %q\n", tunnelName, hostName)
		os.Exit(1)
	}

	delete(entry.Tunnels, tunnelName)

	if err := saveConfig(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed tunnel %q from host %q in %s config (%s)\n",
		tunnelName, hostName, scope, path)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// effectivePort returns the port to display, defaulting to 22 when not specified.
func effectivePort(port int) int {
	if port == 0 {
		return 22
	}
	return port
}

// parseTags splits a comma-separated tag string into a sorted, deduplicated slice.
func parseTags(s string) []string {
	seen := map[string]bool{}
	var tags []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}
	sort.Strings(tags)
	return tags
}
