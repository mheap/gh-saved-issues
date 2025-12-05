package savedsearches

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncerForceRecreates(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfgYAML := `
searches:
  - id: SSC_keep
    name: Existing
    query: "state:open"
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	client := &stubClient{nextID: "SSC_new"}
	syncer := NewSyncer(client, true, false)
	if err := syncer.Sync(context.Background(), cfgPath); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if len(client.deleted) != 1 || client.deleted[0] != "SSC_keep" {
		t.Fatalf("expected delete existing id, got %+v", client.deleted)
	}
	if len(client.created) != 1 || client.created[0].Name != "Existing" {
		t.Fatalf("expected recreate search, got %+v", client.created)
	}
	if len(client.updated) != 0 {
		t.Fatalf("expected no updates when forcing, got %+v", client.updated)
	}
	updatedCfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("reload cfg: %v", err)
	}
	if updatedCfg.Searches[0].ID != "SSC_new" {
		t.Fatalf("expected new id persisted, got %s", updatedCfg.Searches[0].ID)
	}
}
