package savedsearches

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(tmp, ".github-searches.yaml")
	if path != want {
		t.Fatalf("expected %s, got %s", want, path)
	}
}

func TestResolveConfigPathFlag(t *testing.T) {
	path, err := ResolveConfigPath("./foo.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	abs, _ := filepath.Abs("./foo.yaml")
	if path != abs {
		t.Fatalf("expected absolute path %s, got %s", abs, path)
	}
}

func TestRenderQueryWithTemplate(t *testing.T) {
	cfg := map[string]TemplateDefinition{
		"recent": {Query: "assignee:{{ user }} updated:>@today-{{ default(time, \"7d\") }}"},
	}

	query, err := RenderQuery(SearchDefinition{
		Name:     "test",
		Template: "recent",
		Vars: map[string]any{
			"user": "alice",
			"time": "",
		},
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "assignee:alice updated:>@today-7d"
	if query != want {
		t.Fatalf("expected %s, got %s", want, query)
	}
}

func TestRenderQueryWithJoin(t *testing.T) {
	cfg := map[string]TemplateDefinition{
		"repos": {Query: "is:pr ({{ join(repos, \"OR\") }})"},
	}

	query, err := RenderQuery(SearchDefinition{
		Name:     "test",
		Template: "repos",
		Vars: map[string]any{
			"repos": []any{"repo:a", "repo:b", "repo:c"},
		},
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "is:pr (repo:a OR repo:b OR repo:c)"
	if query != want {
		t.Fatalf("expected %s, got %s", want, query)
	}
}

func TestRenderQueryMissingTemplate(t *testing.T) {
	_, err := RenderQuery(SearchDefinition{Template: "missing"}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRenderQuerySectionHeader(t *testing.T) {
	q, err := RenderQuery(SearchDefinition{
		Name:    "== Demo ==",
		Section: "Demo",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q != "" {
		t.Fatalf("expected empty query")
	}
}

func TestRenderQueryMissingVarDefaultsToNil(t *testing.T) {
	cfg := map[string]TemplateDefinition{
		"withMissing": {Query: "repo:{{ default(additional, \"fallback\") }}"},
	}

	query, err := RenderQuery(SearchDefinition{
		Template: "withMissing",
		Vars:     map[string]any{},
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query != "repo:fallback" {
		t.Fatalf("expected fallback resolution, got %s", query)
	}
}
