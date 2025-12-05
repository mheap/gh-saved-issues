package savedsearches

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
)

const (
	defaultEndpoint = "https://github.com/_graphql"

	createPersistedID = "c06c5627e09922bd28c6d34ff91d0530"
	updatePersistedID = "379dbe4cf68c3485e48df2f699f5ae75"
	deletePersistedID = "2939ea7192de2c6284da481de6737322"
)

// SavedSearchInput represents the information sent to GitHub.
type SavedSearchInput struct {
	Name        string
	Query       string
	Description string
}

// Client describes the operations needed by the syncer.
type Client interface {
	CreateSavedSearch(ctx context.Context, input SavedSearchInput) (string, error)
	UpdateSavedSearch(ctx context.Context, id string, input SavedSearchInput) error
	DeleteSavedSearch(ctx context.Context, id string) error
}

// GraphQLClient performs GraphQL requests against github.com.
type GraphQLClient struct {
	httpClient *http.Client
	endpoint   string
	token      string
	cookie     string
}

// NewGraphQLClient builds a client using GH authentication.
func NewGraphQLClient(ctx context.Context, endpoint string) (*GraphQLClient, error) {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token == "" {
		token, _ = auth.TokenForHost("github.com")
	}

	if token == "" {
		return nil, errors.New("no GitHub token available; set GH_TOKEN or gh auth login")
	}

	cookie := os.Getenv("GH_COOKIE")
	if cookie == "" {
		cookie = os.Getenv("GITHUB_COOKIE")
	}

	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	return &GraphQLClient{
		httpClient: http.DefaultClient,
		endpoint:   endpoint,
		token:      token,
		cookie:     cookie,
	}, nil
}

// CreateSavedSearch creates a new shortcut and returns the id.
func (c *GraphQLClient) CreateSavedSearch(ctx context.Context, input SavedSearchInput) (string, error) {
	vars := map[string]any{
		"input": map[string]any{
			"color":      "GRAY",
			"icon":       "BOOKMARK",
			"name":       input.Name,
			"query":      input.Query,
			"searchType": "ISSUES",
		},
	}

	if input.Description != "" {
		vars["input"].(map[string]any)["description"] = input.Description
	}

	data, err := c.graphQL(ctx, createPersistedID, vars)
	if err != nil {
		return "", err
	}

	id, ok := findShortcutID(data, input.Name)
	if !ok {
		return "", errors.New("create saved search: id not found in response")
	}

	return id, nil
}

// UpdateSavedSearch updates an existing shortcut.
func (c *GraphQLClient) UpdateSavedSearch(ctx context.Context, id string, input SavedSearchInput) error {
	vars := map[string]any{
		"input": map[string]any{
			"color":             "GRAY",
			"description":       input.Description,
			"icon":              "BOOKMARK",
			"name":              input.Name,
			"query":             input.Query,
			"scopingRepository": nil,
			"shortcutId":        id,
		},
	}

	_, err := c.graphQL(ctx, updatePersistedID, vars)
	return err
}

// DeleteSavedSearch removes a shortcut.
func (c *GraphQLClient) DeleteSavedSearch(ctx context.Context, id string) error {
	vars := map[string]any{
		"input": map[string]any{
			"shortcutId": id,
		},
	}

	_, err := c.graphQL(ctx, deletePersistedID, vars)
	return err
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data   map[string]any `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *GraphQLClient) graphQL(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	payload := graphQLRequest{Query: query, Variables: variables}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}

	req.Header.Set("github-verified-fetch", "true")
	req.Header.Set("origin", "https://github.com")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post graphql: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("graphql status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed graphQLResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse graphql response: %w", err)
	}

	if len(parsed.Errors) > 0 {
		var messages []string
		for _, e := range parsed.Errors {
			messages = append(messages, e.Message)
		}
		return nil, fmt.Errorf("graphql error: %s", strings.Join(messages, "; "))
	}

	return parsed.Data, nil
}

// findShortcutID reads the saved search id from the known response shape.
func findShortcutID(data any, targetName string) (string, bool) {
	root, ok := data.(map[string]any)
	if !ok {
		return "", false
	}

	create, ok := root["createDashboardSearchShortcut"].(map[string]any)
	if !ok {
		return "", false
	}

	dashboard, ok := create["dashboard"].(map[string]any)
	if !ok {
		return "", false
	}

	shortcuts, ok := dashboard["shortcuts"].(map[string]any)
	if !ok {
		return "", false
	}

	nodes, ok := shortcuts["nodes"].([]any)
	if !ok {
		return "", false
	}

	for _, n := range nodes {
		node, ok := n.(map[string]any)
		if !ok {
			continue
		}
		name, _ := node["name"].(string)
		if name != targetName {
			continue
		}
		if id, _ := node["id"].(string); strings.HasPrefix(id, "SSC_") {
			return id, true
		}
	}

	return "", false
}
