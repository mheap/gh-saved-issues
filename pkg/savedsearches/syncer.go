package savedsearches

import (
	"context"
	"fmt"
	"time"
)

// Syncer applies configuration to GitHub.
type Syncer struct {
	client   Client
	recreate bool
	reset    bool
}

// NewSyncer constructs a Syncer.
func NewSyncer(client Client, recreate, reset bool) *Syncer {
	return &Syncer{client: client, recreate: recreate, reset: reset}
}

// Sync reads config, reconciles with GitHub, and writes any updates.
func (s *Syncer) Sync(ctx context.Context, configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	updated := false
	for i := range cfg.Searches {
		search := &cfg.Searches[i]

		if search.Name != "" {
			fmt.Println("Processing: " + search.Name)
		}

		query, err := RenderQuery(*search, cfg.Templates)
		if err != nil {
			return fmt.Errorf("%s: %w", search.Name, err)
		}

		name := search.Name
		if name == "" && search.Section != "" {
			name = fmt.Sprintf("== %s ==", search.Section)
		}
		if name == "" {
			return fmt.Errorf("search entry missing name")
		}

		input := SavedSearchInput{
			Name:  name,
			Query: query,
		}

		if search.Remove {
			if search.ID != "" {
				if err := s.client.DeleteSavedSearch(ctx, search.ID); err != nil {
					return fmt.Errorf("delete %s: %w", search.Name, err)
				}
				search.ID = ""
				search.Remove = false
				updated = true
			}
			continue
		}

		if s.reset {
			if search.ID != "" {
				if err := s.client.DeleteSavedSearch(ctx, search.ID); err != nil {
					return fmt.Errorf("reset delete %s: %w", search.Name, err)
				}
				search.ID = ""
				updated = true
			}
			continue
		}

		if s.recreate && search.ID != "" {
			if err := s.client.DeleteSavedSearch(ctx, search.ID); err != nil {
				return fmt.Errorf("force delete %s: %w", search.Name, err)
			}
			search.ID = ""
			updated = true
		}

		if search.ID == "" {
			id, err := s.client.CreateSavedSearch(ctx, input)
			if err != nil {
				return fmt.Errorf("create %s: %w", search.Name, err)
			}
			search.ID = id
			updated = true
		} else {
			if err := s.client.UpdateSavedSearch(ctx, search.ID, input); err != nil {
				return fmt.Errorf("update %s: %w", search.Name, err)
			}
		}

		time.Sleep(1 * time.Second)
	}

	if updated {
		if err := SaveConfig(configPath, cfg); err != nil {
			return err
		}
	}

	return nil
}
