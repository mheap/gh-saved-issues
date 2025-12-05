package savedsearches

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncerResetDeletesOnly(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfgYAML := `
searches:
  - id: SSC_keep
    name: Existing
    query: "state:open"
  - name: NoID
    query: "state:open"
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	client := &stubClient{}
	syncer := NewSyncer(client, false, true)
	if err := syncer.Sync(context.Background(), cfgPath); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if len(client.deleted) != 1 || client.deleted[0] != "SSC_keep" {
		t.Fatalf("expected delete SSC_keep, got %+v", client.deleted)
	}
	if len(client.created) != 0 || len(client.updated) != 0 {
		t.Fatalf("expected no creates/updates, got c:%+v u:%+v", client.created, client.updated)
	}

	updatedCfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("reload cfg: %v", err)
	}
	if updatedCfg.Searches[0].ID != "" {
		t.Fatalf("expected cleared id, got %s", updatedCfg.Searches[0].ID)
	}
}
