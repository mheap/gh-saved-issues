# GitHub Saved Searches (gh extension)

Manage GitHub saved issue/PR searches from a YAML config and keep them in sync with your account. This extension reads your config, expands templates, and calls GitHub’s persisted GraphQL queries to create/update/delete saved searches. It keeps IDs in the config so subsequent runs are incremental, and offers reset/recreate modes when you need a clean slate.

## Installation

```sh
gh extension install github.com/mheap/github-saved-issues
```

## Configuration

Default config path: `$XDG_HOME/.github-searches.yaml` (fallbacks: `$XDG_CONFIG_HOME/.github-searches.yaml`, `~/.config/.github-searches.yaml`). Override with `--config /path/to/file.yaml`.

### YAML structure

```yaml
searches:
  # Section header (optional, shows as "== Team ==" in GitHub)
  - section: Team

  # Plain query
  - name: Assigned to me (Kong)
    query: state:open archived:false assignee:@me sort:updated-desc org:Kong

  # Existing search (id keeps it in sync)
  - id: SSC_kgDDKB3arw
    name: Assigned to Steve (All Orgs)
    query: state:open archived:false assignee:steve.doe sort:updated-desc

  # Template-based search
  - name: Work from Alice
    template: recent-work
    vars:
      user: alice.jones
      time: 7d

  - name: Terraform PRs
    template: repo-prs
    vars:
      repos:
        - repo:Kong/terraform-provider-konnect
        - repo:Kong/terraform-provider-konnect-beta

  # Remove an existing search
  - id: SSC_kgDOAB3rsg
    name: Old search
    remove: true

templates:
  recent-work:
    query: org:my-org ((assignee:{{ user }} AND is:issue) OR (is:pr AND author:{{ user }})) (is:open OR updated:>@today-{{ default(time, "30d") }}) sort:updated-desc

  repo-prs:
    query: is:pr state:open ({{ join(repos, "OR") }}) -author:app/renovate draft:false
```

Notes:

- `id` values are the saved-search IDs (`SSC_*`). If missing, the tool creates the search and writes the ID back to the file.
- `section` entries are headers: only `id`/`section` expected in config; the tool sends them as `== SECTION ==` with an empty query.
- `remove: true` deletes the search if `id` is present; the ID is cleared in the file.

### Template helpers

- `default(value, "fallback")`
- `join(list, "SEP")` joins arrays/slices (adds spaces around the separator).
- Missing vars are treated as `nil`, so `default(missing, "fallback")` works.

## Usage

```sh
gh saved-issues              # sync using default config
gh saved-issues --config ./examples/searches.yaml
gh saved-issues --recreate   # delete+recreate all configured searches
gh saved-issues --reset      # delete configured searches without recreating
```

Authentication:

Set `GITHUB_COOKIE` to send a Cookie header (for session-based auth).

```
$ export GITHUB_COOKIE="_device_id=b1af9b4a09daef122e239405dda39pe1;user_session=L2S8tclDBjL3IoQCORkRWKnom3Y6fcWZ0Wa3gPXOtgsny8sC;"
```

## Development

```sh
go test ./...
```

Requires Go 1.25+. Dependencies managed via `go mod tidy`.
