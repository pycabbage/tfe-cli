# AGENTS.md

## Commands

```bash
go build -v ./...                              # build
go test -v ./...                                # run all tests
go test -v ./internal/tfc/                      # single package
go test -v -run TestUploadState_Success ./internal/tfc/  # single test
go tool golangci-lint run --fix ./...           # lint (default config, no .golangci.yml)
```

Pre-commit hook (lefthook) runs test + lint automatically on `*.go` changes.

CI runs `go build -v ./...` then `go test -v ./...` on every PR.

## Architecture

```
main.go → cmd.Execute()
  rootCmd.PersistentPreRunE:
    config.Load()  → reads TFC_API_TOKEN, TFC_ORGANIZATION, TFC_WORKSPACE_NAME (env + .env)
    tfc.New(cfg)   → wraps *tfe.Client (hashicorp/go-tfe v1.103.0)

cmd/           Cobra commands. Package-level `var client *tfc.Client` set in root.go.
internal/tfc/  Facade over go-tfe SDK. Client struct wraps *tfe.Client with workspace cache.
internal/config/  Env loading with custom .env parser (no external lib).
internal/output/  PrintTable / PrintKV helpers.
```

Workspace caching: `GetWorkspace()` caches; `Lock()`/`Unlock()` invalidate (`c.workspace = nil`).

## Critical Convention

**ALL TFE/TFC API calls must go through `hashicorp/go-tfe`. Never use `net/http` directly.**

The go-tfe SDK provides `StateVersions.Upload()` which combines Create + PUT raw state + Re-read in one call. Use it instead of manual HTTP.

## go-tfe SDK Signatures (v1.103.0)

Key non-obvious signatures:

```go
// List takes options only (no workspaceID arg); filter via Organization/Workspace fields
StateVersions.List(ctx, &tfe.StateVersionListOptions{
    Organization: org, Workspace: ws, ListOptions: tfe.ListOptions{PageSize: 10},
})

// Upload combines Create + binary upload + Read; returns finalized state version
StateVersions.Upload(ctx, workspaceID, tfe.StateVersionUploadOptions{
    StateVersionCreateOptions: tfe.StateVersionCreateOptions{Lineage: ..., MD5: ..., Serial: ...},
    RawState: data,
})
```

## Testing

- Standard `testing` package, no assertion library.
- Mock TFE API with `httptest.NewServer(http.StripPrefix("/api/v2", mux))`.
- Test helpers in `internal/tfc/client_test.go`: `newTestClient`, `writeJSON`, `workspaceResponse`.
- All test names, assertion messages, and user-facing strings must be in English.

### Mock JSON:API Date Format

The `hashicorp/jsonapi` library (used by go-tfe) strictly requires ISO8601 with trailing `Z`:

```go
// WRONG — time.RFC3339 can produce "+00:00" which fails jsonapi parsing
"created-at": time.Now().Format(time.RFC3339)

// CORRECT
"created-at": time.Now().UTC().Format("2006-01-02T15:04:05Z")
```

## Code Style

- All comments, test messages, and user-facing strings must be in English.
- Error wrapping: `fmt.Errorf("context: %w", err)`.
- No comments on code (no `// do something` above obvious lines).
- Cobra `RunE` (not `Run`) on all commands; `SilenceUsage: true` + `SilenceErrors: true` on root.

## Environment

Required env vars (or `.env` file, auto-discovered walking up from CWD):

| Variable | Purpose |
|---|---|
| `TFC_API_TOKEN` | HCP Terraform API token |
| `TFC_ORGANIZATION` | Organization name |
| `TFC_WORKSPACE_NAME` | Workspace name |

Env vars take precedence over `.env`. The TFE endpoint is hardcoded to `https://app.terraform.io`.
```
