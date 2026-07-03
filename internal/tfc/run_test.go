package tfc

import (
	"net/http"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

// runFixture builds a minimal in-memory *tfe.Run for use as the `run`
// argument to FindStateVersionForRun, without a round trip through the API.
func runFixture(id string, createdAt time.Time) *tfe.Run {
	return &tfe.Run{ID: id, CreatedAt: createdAt}
}

func runResponse(id string, createdAt time.Time) any {
	return map[string]any{
		"data": map[string]any{
			"id":   id,
			"type": "runs",
			"attributes": map[string]any{
				"status":            "applied",
				"source":            "tfe-api",
				"message":           "test run",
				"created-at":        createdAt.UTC().Format("2006-01-02T15:04:05Z"),
				"terraform-version": "1.5.0",
			},
		},
	}
}

func runListResponse(ids []string, createdAt time.Time) any {
	data := make([]map[string]any, len(ids))
	for i, id := range ids {
		data[i] = map[string]any{
			"id":   id,
			"type": "runs",
			"attributes": map[string]any{
				"status":            "applied",
				"source":            "tfe-api",
				"message":           "test run",
				"created-at":        createdAt.UTC().Format("2006-01-02T15:04:05Z"),
				"terraform-version": "1.5.0",
			},
		}
	}
	return map[string]any{
		"data": data,
		"meta": map[string]any{
			"pagination": map[string]any{
				"current-page": 1,
				"page-size":    10,
				"total-pages":  1,
				"total-count":  len(ids),
			},
		},
	}
}

func workspaceWithCurrentRunResponse(wsID, runID string, createdAt time.Time) any {
	return map[string]any{
		"data": map[string]any{
			"id":   wsID,
			"type": "workspaces",
			"attributes": map[string]any{
				"name":   "myws",
				"locked": false,
			},
			"relationships": map[string]any{
				"current-run": map[string]any{
					"data": map[string]any{"type": "runs", "id": runID},
				},
			},
		},
		"included": []map[string]any{
			{
				"id":   runID,
				"type": "runs",
				"attributes": map[string]any{
					"status":            "applied",
					"source":            "tfe-api",
					"message":           "current run",
					"created-at":        createdAt.UTC().Format("2006-01-02T15:04:05Z"),
					"terraform-version": "1.5.0",
				},
			},
		},
	}
}

func workspaceNoCurrentRunResponse(wsID string) any {
	return map[string]any{
		"data": map[string]any{
			"id":   wsID,
			"type": "workspaces",
			"attributes": map[string]any{
				"name":   "myws",
				"locked": false,
			},
		},
	}
}

// stateVersionResponseWithRun builds a single state-versions JSON:API data
// item with a "run" relationship linkage (resource identifier only).
func stateVersionResponseWithRun(id string, serial int64, createdAt time.Time, runID string) map[string]any {
	item := map[string]any{
		"id":   id,
		"type": "state-versions",
		"attributes": map[string]any{
			"serial":     serial,
			"created-at": createdAt.UTC().Format("2006-01-02T15:04:05Z"),
			"status":     "finalized",
		},
	}
	if runID != "" {
		item["relationships"] = map[string]any{
			"run": map[string]any{
				"data": map[string]any{"type": "runs", "id": runID},
			},
		}
	}
	return item
}

func stateVersionListPage(items []map[string]any, currentPage, totalPages, totalCount int) any {
	return map[string]any{
		"data": items,
		"meta": map[string]any{
			"pagination": map[string]any{
				"current-page": currentPage,
				"page-size":    100,
				"total-pages":  totalPages,
				"total-count":  totalCount,
			},
		},
	}
}

func TestListRuns_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/runs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, runListResponse([]string{"run-001", "run-002"}, time.Now()))
	})
	c := newTestClient(t, mux)

	runs, err := c.ListRuns(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("run count: got %d, want 2", len(runs))
	}
	if runs[0].ID != "run-001" {
		t.Errorf("first ID: got %q, want %q", runs[0].ID, "run-001")
	}
}

func TestGetLatestRun_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("include") != "current_run" {
			t.Errorf("include param: got %q, want %q", r.URL.Query().Get("include"), "current_run")
		}
		writeJSON(w, 200, workspaceWithCurrentRunResponse("ws-001", "run-current", time.Now()))
	})
	c := newTestClient(t, mux)

	run, err := c.GetLatestRun(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != "run-current" {
		t.Errorf("ID: got %q, want %q", run.ID, "run-current")
	}
	if run.Message != "current run" {
		t.Errorf("Message: got %q, want %q", run.Message, "current run")
	}
}

func TestGetLatestRun_NoCurrentRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceNoCurrentRunResponse("ws-001"))
	})
	c := newTestClient(t, mux)

	_, err := c.GetLatestRun(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetRun_EmptyIDDelegatesToLatest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceWithCurrentRunResponse("ws-001", "run-current", time.Now()))
	})
	c := newTestClient(t, mux)

	run, err := c.GetRun(t.Context(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != "run-current" {
		t.Errorf("expected latest for empty ID: got %q", run.ID)
	}
}

func TestGetRun_LatestKeywordDelegatesToLatest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceWithCurrentRunResponse("ws-001", "run-current", time.Now()))
	})
	c := newTestClient(t, mux)

	run, err := c.GetRun(t.Context(), "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != "run-current" {
		t.Errorf("expected latest for 'latest' keyword: got %q", run.ID)
	}
}

func TestGetRun_SpecificID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/runs/run-abc123", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, runResponse("run-abc123", time.Now()))
	})
	c := newTestClient(t, mux)

	run, err := c.GetRun(t.Context(), "run-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != "run-abc123" {
		t.Errorf("ID: got %q, want %q", run.ID, "run-abc123")
	}
}

func TestFindStateVersionForRun_MatchOnFirstPage(t *testing.T) {
	runCreatedAt := time.Now().Add(-time.Hour)
	items := []map[string]any{
		stateVersionResponseWithRun("sv-003", 3, time.Now(), ""),
		stateVersionResponseWithRun("sv-002", 2, time.Now(), "run-target"),
		stateVersionResponseWithRun("sv-001", 1, runCreatedAt.Add(-time.Hour), ""),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionListPage(items, 1, 1, len(items)))
	})
	c := newTestClient(t, mux)

	run := runFixture("run-target", runCreatedAt)
	sv, err := c.FindStateVersionForRun(t.Context(), run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv == nil {
		t.Fatal("expected a state version, got nil")
	}
	if sv.ID != "sv-002" {
		t.Errorf("ID: got %q, want %q", sv.ID, "sv-002")
	}
}

func TestFindStateVersionForRun_NoMatchCutoff(t *testing.T) {
	runCreatedAt := time.Now()
	// All state versions predate the run and none match: cutoff should
	// trigger on the very first item without needing a second page.
	items := []map[string]any{
		stateVersionResponseWithRun("sv-010", 10, runCreatedAt.Add(-time.Hour), ""),
		stateVersionResponseWithRun("sv-009", 9, runCreatedAt.Add(-2*time.Hour), ""),
	}

	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("page[number]") != "" && r.URL.Query().Get("page[number]") != "1" {
			t.Errorf("unexpected page requested: %s", r.URL.Query().Get("page[number]"))
		}
		writeJSON(w, 200, stateVersionListPage(items, 1, 5, 500))
	})
	c := newTestClient(t, mux)

	run := runFixture("run-none", runCreatedAt)
	sv, err := c.FindStateVersionForRun(t.Context(), run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv != nil {
		t.Errorf("expected nil state version, got %+v", sv)
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 page fetch, got %d", calls)
	}
}

func TestFindStateVersionForRun_NaturalExhaustion(t *testing.T) {
	runCreatedAt := time.Now().Add(-24 * time.Hour)
	// A single page of items, all newer than the run and none matching;
	// pagination reports this is the only page (total-pages: 1).
	items := []map[string]any{
		stateVersionResponseWithRun("sv-002", 2, time.Now(), ""),
		stateVersionResponseWithRun("sv-001", 1, time.Now(), ""),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionListPage(items, 1, 1, len(items)))
	})
	c := newTestClient(t, mux)

	run := runFixture("run-none", runCreatedAt)
	sv, err := c.FindStateVersionForRun(t.Context(), run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv != nil {
		t.Errorf("expected nil state version, got %+v", sv)
	}
}

// TestFindStateVersionForRun_ScanLimit is intentionally skipped as a unit
// test: exercising the real 20-page/2000-item cap would require either an
// awkward exported knob to shrink the limit for tests, or spinning up 20
// fake pages of 100 items each with no exit condition ever satisfied. That
// is disproportionate to the value of a unit test here; the scan-limit path
// is covered by manual/live verification against the real workspace instead
// (see task report).
