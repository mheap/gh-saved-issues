package savedsearches

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type stubClient struct {
	created []SavedSearchInput
	updated []SavedSearchInput
	deleted []string
	nextID  string
	err     error
}

func (s *stubClient) CreateSavedSearch(ctx context.Context, input SavedSearchInput) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.created = append(s.created, input)
	if s.nextID != "" {
		return s.nextID, nil
	}
	return "SSC_stub", nil
}

func (s *stubClient) UpdateSavedSearch(ctx context.Context, id string, input SavedSearchInput) error {
	if s.err != nil {
		return s.err
	}
	s.updated = append(s.updated, input)
	return nil
}

func (s *stubClient) DeleteSavedSearch(ctx context.Context, id string) error {
	if s.err != nil {
		return s.err
	}
	s.deleted = append(s.deleted, id)
	return nil
}

func TestSyncerCreatesAndUpdatesConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfgYAML := `
searches:
  - section: Team
    id: SSC_team
  - name: Create
    query: "state:open"
  - name: Update
    id: SSC_existing
    query: "state:closed"
  - section: Other
  - name: Remove
    id: SSC_remove
    query: "state:open"
    remove: true
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	client := &stubClient{nextID: "SSC_new"}
	syncer := NewSyncer(client, false, false)
	if err := syncer.Sync(context.Background(), cfgPath); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if len(client.created) != 2 {
		t.Fatalf("expected two creates (search + header), got %+v", client.created)
	}
	if len(client.updated) != 2 {
		t.Fatalf("expected two updates (existing header + update), got %+v", client.updated)
	}
	if len(client.deleted) != 1 || client.deleted[0] != "SSC_remove" {
		t.Fatalf("expected delete SSC_remove, got %+v", client.deleted)
	}

	updatedCfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("reload cfg: %v", err)
	}

	expectedOrder := []struct {
		name    string
		section string
	}{
		{name: "", section: "Team"},
		{name: "Create"},
		{name: "Update"},
		{name: "", section: "Other"},
		{name: "Remove"},
	}
	if len(updatedCfg.Searches) != len(expectedOrder) {
		t.Fatalf("unexpected search count: %d", len(updatedCfg.Searches))
	}
	for i, exp := range expectedOrder {
		if updatedCfg.Searches[i].Name != exp.name || updatedCfg.Searches[i].Section != exp.section {
			t.Fatalf("unexpected order, index %d = %+v", i, updatedCfg.Searches[i])
		}
	}
	if updatedCfg.Searches[0].ID != "SSC_team" {
		t.Fatalf("expected team header retained, got %s", updatedCfg.Searches[0].ID)
	}
	if updatedCfg.Searches[1].ID != "SSC_new" {
		t.Fatalf("expected new id persisted on create, got %s", updatedCfg.Searches[1].ID)
	}
	if updatedCfg.Searches[3].ID != "SSC_new" {
		t.Fatalf("expected new id for other header, got %s", updatedCfg.Searches[3].ID)
	}
	if updatedCfg.Searches[4].ID != "" {
		t.Fatalf("expected removed id cleared, got %s", updatedCfg.Searches[4].ID)
	}
}
