package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const configVersion = 1

var rootNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)

type Root struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Capabilities struct {
	Read    bool `json:"read"`
	Search  bool `json:"search"`
	Write   bool `json:"write"`
	Command bool `json:"command"`
}

type Config struct {
	Version      int          `json:"version"`
	Roots        []Root       `json:"roots"`
	Capabilities Capabilities `json:"capabilities"`
}

func defaultConfig() Config {
	return Config{Version: configVersion, Roots: []Root{}, Capabilities: Capabilities{}}
}

func configPath() (string, error) {
	base := os.Getenv("APPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "chaOS", "config.json"), nil
}

func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config JSON: %w", err)
	}
	if cfg.Version != configVersion {
		return Config{}, fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	seen := map[string]bool{}
	for i := range cfg.Roots {
		root := &cfg.Roots[i]
		root.Name = strings.ToLower(strings.TrimSpace(root.Name))
		if !rootNamePattern.MatchString(root.Name) {
			return Config{}, fmt.Errorf("invalid root name %q", root.Name)
		}
		if seen[root.Name] {
			return Config{}, fmt.Errorf("duplicate root name %q", root.Name)
		}
		seen[root.Name] = true
		absolute, err := filepath.Abs(root.Path)
		if err != nil {
			return Config{}, fmt.Errorf("invalid path for root %q: %w", root.Name, err)
		}
		stat, err := os.Stat(absolute)
		if err != nil {
			return Config{}, fmt.Errorf("root %q is unavailable: %w", root.Name, err)
		}
		if !stat.IsDir() {
			return Config{}, fmt.Errorf("root %q is not a directory", root.Name)
		}
		root.Path = filepath.Clean(absolute)
	}
	sort.Slice(cfg.Roots, func(i, j int) bool { return cfg.Roots[i].Name < cfg.Roots[j].Name })
	return cfg, nil
}

func saveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	cfg.Version = configVersion
	sort.Slice(cfg.Roots, func(i, j int) bool { return cfg.Roots[i].Name < cfg.Roots[j].Name })
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func addRoot(name, hostPath string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if !rootNamePattern.MatchString(name) {
		return fmt.Errorf("root name must match %s", rootNamePattern.String())
	}
	absolute, err := filepath.Abs(hostPath)
	if err != nil {
		return err
	}
	stat, err := os.Stat(absolute)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", absolute)
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	for _, root := range cfg.Roots {
		if root.Name == name {
			return fmt.Errorf("root %q already exists; remove it first to change its path", name)
		}
	}
	cfg.Roots = append(cfg.Roots, Root{Name: name, Path: filepath.Clean(absolute)})
	return saveConfig(cfg)
}

func removeRoot(name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	kept := cfg.Roots[:0]
	found := false
	for _, root := range cfg.Roots {
		if root.Name == name {
			found = true
			continue
		}
		kept = append(kept, root)
	}
	if !found {
		return fmt.Errorf("root %q does not exist", name)
	}
	cfg.Roots = kept
	return saveConfig(cfg)
}

func setCapability(name string, enabled bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	switch strings.ToLower(name) {
	case "read":
		cfg.Capabilities.Read = enabled
	case "search":
		cfg.Capabilities.Search = enabled
	case "write":
		cfg.Capabilities.Write = enabled
	case "command":
		cfg.Capabilities.Command = enabled
	default:
		return fmt.Errorf("unknown capability %q; expected read, search, write or command", name)
	}
	return saveConfig(cfg)
}

func rootByName(cfg Config, name string) (Root, bool) {
	for _, root := range cfg.Roots {
		if root.Name == name {
			return root, true
		}
	}
	return Root{}, false
}

// resolveVirtual maps /root/relative/path to an approved host root. Existing paths are
// evaluated through symlinks. For a missing write target, its closest existing parent is
// evaluated so a symlinked parent cannot escape the approved root.
func resolveVirtual(cfg Config, virtual string, allowMissing bool) (string, string, error) {
	virtual = strings.ReplaceAll(strings.TrimSpace(virtual), `\`, "/")
	if !strings.HasPrefix(virtual, "/") {
		return "", "", fmt.Errorf("path must use an approved virtual root such as /project/file.txt")
	}
	clean := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(filepath.FromSlash(virtual))), "/")
	parts := strings.Split(clean, "/")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "." {
		return "", "", fmt.Errorf("path must name an approved root")
	}
	root, ok := rootByName(cfg, strings.ToLower(parts[0]))
	if !ok {
		return "", "", fmt.Errorf("unknown approved root /%s", parts[0])
	}
	relParts := parts[1:]
	for _, part := range relParts {
		if part == ".." {
			return "", "", fmt.Errorf("path traversal is not allowed")
		}
	}
	target := root.Path
	if len(relParts) > 0 {
		target = filepath.Join(append([]string{root.Path}, relParts...)...)
	}
	if err := ensureInside(root.Path, target); err != nil {
		return "", "", err
	}
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		if err := ensureInside(root.Path, resolved); err != nil {
			return "", "", fmt.Errorf("resolved path escapes approved root: %w", err)
		}
		target = resolved
	} else if !allowMissing || !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	} else {
		parent := filepath.Dir(target)
		for {
			resolvedParent, parentErr := filepath.EvalSymlinks(parent)
			if parentErr == nil {
				if err := ensureInside(root.Path, resolvedParent); err != nil {
					return "", "", fmt.Errorf("write parent escapes approved root: %w", err)
				}
				break
			}
			if !errors.Is(parentErr, os.ErrNotExist) {
				return "", "", parentErr
			}
			next := filepath.Dir(parent)
			if next == parent {
				return "", "", parentErr
			}
			parent = next
		}
	}
	canonical := "/" + root.Name
	if len(relParts) > 0 {
		canonical += "/" + strings.Join(relParts, "/")
	}
	return target, canonical, nil
}

func ensureInside(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path is outside approved root")
	}
	return nil
}
