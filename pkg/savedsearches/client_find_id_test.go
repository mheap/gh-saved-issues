package savedsearches

import "testing"

func TestFindShortcutIDPrefersMatchingName(t *testing.T) {
	data := map[string]any{
		"createDashboardSearchShortcut": map[string]any{
			"dashboard": map[string]any{
				"shortcuts": map[string]any{
					"nodes": []any{
						map[string]any{"id": "SSC_1", "name": "Other"},
						map[string]any{"id": "SSC_2", "name": "Target"},
					},
				},
			},
		},
	}

	id, ok := findShortcutID(data, "Target")
	if !ok || id != "SSC_2" {
		t.Fatalf("expected SSC_2, got %s", id)
	}
}
