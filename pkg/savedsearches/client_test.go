package savedsearches

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSavedSearch(t *testing.T) {
	var received graphQLRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("expected auth header")
		}
		if r.Header.Get("Cookie") != "a=b" {
			t.Fatalf("expected cookie header")
		}

		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if received.Query != createPersistedID {
			t.Fatalf("expected persisted id %s, got %s", createPersistedID, received.Query)
		}

		resp := graphQLResponse{
			Data: map[string]any{
				"createDashboardSearchShortcut": map[string]any{
					"dashboard": map[string]any{
						"shortcuts": map[string]any{
							"nodes": []any{
								map[string]any{"id": "SSC_test123", "name": "Demo"},
								map[string]any{"id": "SSC_other", "name": "Other"},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := &GraphQLClient{
		httpClient: ts.Client(),
		endpoint:   ts.URL,
		token:      "token",
		cookie:     "a=b",
	}

	id, err := client.CreateSavedSearch(context.Background(), SavedSearchInput{
		Name:  "Demo",
		Query: "state:open",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id != "SSC_test123" {
		t.Fatalf("expected id from response, got %s", id)
	}

	input := received.Variables["input"].(map[string]any)
	if input["name"] != "Demo" || input["query"] != "state:open" {
		t.Fatalf("unexpected input payload: %+v", input)
	}
}

func TestUpdateSavedSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Query != updatePersistedID {
			t.Fatalf("expected %s, got %s", updatePersistedID, req.Query)
		}
		if r.Header.Get("Cookie") != "" {
			t.Fatalf("unexpected cookie header")
		}
		w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer ts.Close()

	client := &GraphQLClient{
		httpClient: ts.Client(),
		endpoint:   ts.URL,
		token:      "token",
	}

	err := client.UpdateSavedSearch(context.Background(), "SSC_123", SavedSearchInput{
		Name:  "Demo",
		Query: "state:open",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSavedSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Query != deletePersistedID {
			t.Fatalf("expected %s, got %s", deletePersistedID, req.Query)
		}
		w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer ts.Close()

	client := &GraphQLClient{
		httpClient: ts.Client(),
		endpoint:   ts.URL,
		token:      "token",
	}

	if err := client.DeleteSavedSearch(context.Background(), "SSC_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer ts.Close()

	client := &GraphQLClient{
		httpClient: ts.Client(),
		endpoint:   ts.URL,
		token:      "token",
	}

	_, err := client.graphQL(context.Background(), "ignored", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}
