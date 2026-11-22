package ai

import (
	"net/http"
	"testing"

	"github.com/HolyArrs/holyarrs/app/backend/internal/core/parts"
	"github.com/HolyArrs/holyarrs/app/backend/internal/core/webserver"
)

func TestExtraMCPTools(t *testing.T) {
	g := newRig(t)
	g.who.MCPGroups = []string{"see", "add", "control", "settings", "approve"}
	approved := 0
	declined := 0
	subSearch := 0
	g.h.Inner = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/library/stats":
			webserver.WriteJSON(w, http.StatusOK, map[string]any{"movies": 42, "shows": 10})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/home":
			webserver.WriteJSON(w, http.StatusOK, map[string]any{"recent": []string{"Dune"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/subtitles/missing":
			webserver.WriteJSON(w, http.StatusOK, map[string]any{"items": []map[string]any{{"language": r.URL.Query().Get("language")}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/subtitles/missing/search":
			subSearch++
			webserver.WriteJSON(w, http.StatusAccepted, map[string]any{"started": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/asks":
			webserver.WriteJSON(w, http.StatusOK, map[string]any{
				"asks": []map[string]any{
					{"id": 5, "state": "pending", "name": "Movie A"},
					{"id": 6, "state": "done", "name": "Movie B"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/asks/5/approve":
			approved++
			webserver.WriteJSON(w, http.StatusOK, map[string]any{"ask": map[string]any{"id": 5, "name": "Movie A"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/asks/5/turn-down":
			declined++
			webserver.WriteJSON(w, http.StatusOK, map[string]any{"ask": map[string]any{"id": 5, "name": "Movie A"}})
		default:
			webserver.WriteJSON(w, http.StatusOK, map[string]any{})
		}
	})

	// Read-only tools.
	if m, _, isErr := g.tool("library_stats", map[string]any{}); isErr || m["movies"] != float64(42) {
		t.Fatalf("library_stats: %+v err=%v", m, isErr)
	}
	if m, _, isErr := g.tool("recent_additions", map[string]any{}); isErr || m["recent"] == nil {
		t.Fatalf("recent_additions: %+v err=%v", m, isErr)
	}
	if m, _, isErr := g.tool("missing_subtitles", map[string]any{"language": "es-es", "limit": 10}); isErr || m["items"] == nil {
		t.Fatalf("missing_subtitles: %+v err=%v", m, isErr)
	}
	if m, _, isErr := g.tool("pending_requests", map[string]any{}); isErr || m["count"] != 1 {
		t.Fatalf("pending_requests: %+v err=%v", m, isErr)
	}

	// Two-step approve_request.
	step1, _, isErr := g.tool("approve_request", map[string]any{"ask_id": 5})
	if isErr || step1["confirm_token"] == "" || approved != 0 {
		t.Fatalf("approve step 1: %+v err=%v approved=%d", step1, isErr, approved)
	}
	step2, _, isErr := g.tool("approve_request", map[string]any{"ask_id": 5, "confirm_token": step1["confirm_token"]})
	if isErr || step2["done"] != true || approved != 1 {
		t.Fatalf("approve step 2: %+v err=%v approved=%d", step2, isErr, approved)
	}

	// Two-step decline_request.
	d1, _, isErr := g.tool("decline_request", map[string]any{"ask_id": 5, "note": "not available"})
	if isErr || d1["confirm_token"] == "" || declined != 0 {
		t.Fatalf("decline step 1: %+v err=%v", d1, isErr)
	}
	d2, _, isErr := g.tool("decline_request", map[string]any{"ask_id": 5, "note": "not available", "confirm_token": d1["confirm_token"]})
	if isErr || d2["done"] != true || declined != 1 {
		t.Fatalf("decline step 2: %+v err=%v declined=%d", d2, isErr, declined)
	}

	// Two-step search_missing_subtitles.
	s1, _, isErr := g.tool("search_missing_subtitles", map[string]any{})
	if isErr || s1["confirm_token"] == "" || subSearch != 0 {
		t.Fatalf("sub search step 1: %+v err=%v", s1, isErr)
	}
	s2, _, isErr := g.tool("search_missing_subtitles", map[string]any{"confirm_token": s1["confirm_token"]})
	if isErr || s2["done"] != true || subSearch != 1 {
		t.Fatalf("sub search step 2: %+v err=%v subSearch=%d", s2, isErr, subSearch)
	}

	// Subtitles part off hides subtitle tools.
	g.partsOff[parts.Subtitles] = true
	if _, _, isErr := g.tool("missing_subtitles", map[string]any{}); !isErr {
		t.Fatal("missing_subtitles should be unavailable when Subtitles part is off")
	}
}
