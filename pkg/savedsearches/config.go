package savedsearches

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration file.
type Config struct {
	Searches  []SearchDefinition            `yaml:"searches"`
	Templates map[string]TemplateDefinition `yaml:"templates"`
}

// SearchDefinition is a single saved search definition.
type SearchDefinition struct {
	ID       string         `yaml:"id,omitempty"`
	Name     string         `yaml:"name,omitempty"`
	Query    string         `yaml:"query,omitempty"`
	Section  string         `yaml:"section,omitempty"`
	Template string         `yaml:"template,omitempty"`
	Vars     map[string]any `yaml:"vars,omitempty"`
	Remove   bool           `yaml:"remove,omitempty"`
}

// TemplateTemplate describes a reusable template for queries.
type TemplateDefinition struct {
	Query string `yaml:"query"`
}

// ResolveConfigPath chooses the config path based on flags and env.
func ResolveConfigPath(flagValue string) (string, error) {
	if flagValue != "" {
		return expandPath(flagValue)
	}

	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, ".github-searches.yaml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}

	return filepath.Join(home, ".config", ".github-searches.yaml"), nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve absolute path: %w", err)
		}
		return abs, nil
	}
	return path, nil
}

// LoadConfig reads YAML from disk.
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes YAML back to disk.
func SaveConfig(path string, cfg Config) error {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// RenderQuery resolves the query for a search, handling templates.
func RenderQuery(def SearchDefinition, templates map[string]TemplateDefinition) (string, error) {
	if def.Query != "" {
		return def.Query, nil
	}

	if isSectionHeader(def) {
		return "", nil
	}

	if def.Template == "" {
		return "", errors.New("search must have either query or template")
	}

	tpl, ok := templates[def.Template]
	if !ok {
		return "", fmt.Errorf("template %q not found", def.Template)
	}

	normalized := normalizeTemplateSyntax(tpl.Query)

	funcs := template.FuncMap{
		"default": func(value any, fallback string) string {
			switch v := value.(type) {
			case string:
				if v != "" {
					return v
				}
			case nil:
				// keep empty
			}
			return fallback
		},
		"join": func(value any, sep string) string {
			switch v := value.(type) {
			case []any:
				parts := make([]string, 0, len(v))
				for _, item := range v {
					parts = append(parts, fmt.Sprint(item))
				}
				return strings.Join(parts, " "+sep+" ")
			case []string:
				return strings.Join(v, " "+sep+" ")
			default:
				return fmt.Sprint(v)
			}
		},
	}

	for key, val := range def.Vars {
		v := val
		funcs[key] = func() any { return v }
	}

	for _, name := range referencedIdentifiers(normalized) {
		if _, ok := funcs[name]; ok {
			continue
		}
		name := name
		funcs[name] = func() any {
			_ = name
			return nil
		}
	}

	t, err := template.New("query").Option("missingkey=zero").Funcs(funcs).Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", def.Template, err)
	}

	var buf []byte
	b := &buffer{&buf}
	if err := t.Execute(b, def.Vars); err != nil {
		return "", fmt.Errorf("execute template %q: %w", def.Template, err)
	}

	return string(buf), nil
}

// buffer is a minimal bytes.Buffer replacement without pulling extra deps.
type buffer struct {
	b *[]byte
}

func (b *buffer) Write(p []byte) (int, error) {
	*b.b = append(*b.b, p...)
	return len(p), nil
}

var defaultCallPattern = regexp.MustCompile(`\{\{\s*default\(\s*([^,]+?)\s*,\s*"(.*?)"\s*\)\s*\}\}`)
var joinCallPattern = regexp.MustCompile(`\{\{\s*join\(\s*([^,]+?)\s*,\s*"(.*?)"\s*\)\s*\}\}`)
var actionPattern = regexp.MustCompile(`\{\{[^}]+\}\}`)
var identifierPattern = regexp.MustCompile(`([a-zA-Z_][\\w]*)`)

func normalizeTemplateSyntax(in string) string {
	out := defaultCallPattern.ReplaceAllStringFunc(in, func(m string) string {
		parts := defaultCallPattern.FindStringSubmatch(m)
		return fmt.Sprintf("{{ default %s \"%s\" }}", ensureDot(strings.TrimSpace(parts[1])), parts[2])
	})
	out = joinCallPattern.ReplaceAllStringFunc(out, func(m string) string {
		parts := joinCallPattern.FindStringSubmatch(m)
		return fmt.Sprintf("{{ join %s \"%s\" }}", ensureDot(strings.TrimSpace(parts[1])), parts[2])
	})
	return out
}

func ensureDot(expr string) string {
	if strings.HasPrefix(expr, ".") || strings.HasPrefix(expr, "$") {
		return expr
	}
	if strings.ContainsAny(expr, " ()") {
		return expr
	}
	return "." + expr
}

func referencedIdentifiers(in string) []string {
	seen := map[string]bool{}
	var ids []string
	for _, action := range actionPattern.FindAllString(in, -1) {
		for _, m := range identifierPattern.FindAllStringSubmatch(action, -1) {
			name := m[1]
			switch name {
			case "default", "join", "range", "end", "if", "else", "with":
				continue
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			ids = append(ids, name)
		}
	}
	// Also check default() patterns for first argument identifiers.
	for _, m := range defaultCallPattern.FindAllStringSubmatch(in, -1) {
		name := strings.TrimSpace(m[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		ids = append(ids, name)
	}
	return ids
}

func isSectionHeader(def SearchDefinition) bool {
	return def.Section != "" && def.Template == "" && def.Query == ""
}
