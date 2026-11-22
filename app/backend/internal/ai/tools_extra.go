package ai

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/HolyArrs/holyarrs/app/backend/internal/core/parts"
	"github.com/HolyArrs/holyarrs/app/backend/internal/core/webserver"
)

func subtitlesOff(h *Handlers, _ webserver.Caller) string {
	if !h.partOn(parts.Subtitles) {
		return "Subtitles is off"
	}
	return ""
}

// extraTools returns additional MCP tools provided by the fork.
func (h *Handlers) extraTools() []toolSpec {
	return []toolSpec{
		{
			name:   "library_stats",
			title:  "Library stats",
			about:  "Counts and disk usage of movies, shows and music in the Library.",
			schema: object(map[string]any{}),
			off:    mediaOff,
			run:    toolLibraryStats,
		},
		{
			name:   "recent_additions",
			title:  "Recent additions",
			about:  "Home overview showing recently added and upcoming titles.",
			schema: object(map[string]any{}),
			off:    mediaOff,
			run:    toolRecentAdditions,
		},
		{
			name:  "missing_subtitles",
			title: "Missing subtitles",
			about: "List files in the Library that are still missing wanted subtitles.",
			schema: object(map[string]any{
				"language": strProp("Optional language code filter, such as es, es-es or en."),
				"kind":     map[string]any{"type": "string", "enum": []string{"movie", "show"}},
				"limit":    intProp("Maximum number of rows (1 to 100, default 25)."),
			}),
			off: subtitlesOff,
			run: toolMissingSubtitles,
		},
		{
			name:   "search_missing_subtitles",
			title:  "Search missing subtitles",
			action: true,
			about:  "Start a background search for missing subtitles. First call returns a confirm_token; call again with it after agreement.",
			schema: object(map[string]any{"confirm_token": tokProp}),
			off:    needPerm(webserver.PermManageLibrary, subtitlesOff),
			run:    toolSearchMissingSubtitles,
		},
		{
			name:   "pending_requests",
			title:  "Pending requests",
			about:  "List all requests waiting for approval.",
			schema: object(map[string]any{}),
			off:    needPerm(webserver.PermApprove, asksOff),
			run:    toolPendingRequests,
		},
		{
			name:   "approve_request",
			title:  "Approve a request",
			action: true,
			about:  "Approve a pending request by its numeric ask_id. First call returns a confirm_token; call again with it after agreement.",
			schema: object(map[string]any{
				"ask_id":        intProp("Numeric request id from pending_requests."),
				"note":          strProp("Optional short note for the requester."),
				"confirm_token": tokProp,
			}, "ask_id"),
			off: needPerm(webserver.PermApprove, asksOff),
			run: toolApproveRequest,
		},
		{
			name:   "decline_request",
			title:  "Decline a request",
			action: true,
			about:  "Decline a pending request by its numeric ask_id. First call returns a confirm_token; call again with it after agreement.",
			schema: object(map[string]any{
				"ask_id":        intProp("Numeric request id from pending_requests."),
				"note":          strProp("Optional reason shown to the requester."),
				"confirm_token": tokProp,
			}, "ask_id"),
			off: needPerm(webserver.PermApprove, asksOff),
			run: toolDeclineRequest,
		},
	}
}

func toolLibraryStats(h *Handlers, r *http.Request, who webserver.Caller, _ map[string]any) (any, error) {
	return h.read(r, who, "/api/v1/library/stats")
}

func toolRecentAdditions(h *Handlers, r *http.Request, who webserver.Caller, _ map[string]any) (any, error) {
	return h.read(r, who, "/api/v1/home")
}

func toolMissingSubtitles(h *Handlers, r *http.Request, who webserver.Caller, args map[string]any) (any, error) {
	q := url.Values{}
	if lang := str(args, "language"); lang != "" {
		q.Set("language", clean(lang, 16))
	}
	if kind := str(args, "kind"); kind == "movie" || kind == "show" {
		q.Set("kind", kind)
	}
	limit := num(args, "limit")
	if limit < 1 || limit > 100 {
		limit = 25
	}
	q.Set("limit", strconv.FormatInt(limit, 10))
	return h.read(r, who, "/api/v1/subtitles/missing?"+q.Encode())
}

func toolSearchMissingSubtitles(h *Handlers, r *http.Request, who webserver.Caller, args map[string]any) (any, error) {
	summary := map[string]any{"will_do": "Start a background search for missing subtitles."}
	out, goAhead, err := h.confirmStep(who, "search_missing_subtitles", args, summary)
	if err != nil || !goAhead {
		return out, err
	}
	code, v := h.call(r, who, webserver.ScopeWrite, http.MethodPost, "/api/v1/subtitles/missing/search", map[string]any{})
	return h.acted(r, who, "search_missing_subtitles", args, "ai_subtitles_search", "started missing subtitle search for", code, v)
}

func toolPendingRequests(h *Handlers, r *http.Request, who webserver.Caller, _ map[string]any) (any, error) {
	v, err := h.read(r, who, "/api/v1/asks")
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return v, nil
	}
	asks, ok := m["asks"].([]any)
	if !ok {
		return v, nil
	}
	pending := make([]any, 0)
	for _, a := range asks {
		if row, ok := a.(map[string]any); ok {
			if st, _ := row["state"].(string); st == "pending" {
				pending = append(pending, row)
			}
		}
	}
	return map[string]any{"pending": pending, "count": len(pending)}, nil
}

func askIDArg(args map[string]any) (int64, error) {
	switch v := args["ask_id"].(type) {
	case int:
		if v > 0 {
			return int64(v), nil
		}
	case int64:
		if v > 0 {
			return v, nil
		}
	}
	id := num(args, "ask_id")
	if id <= 0 {
		return 0, failure{"send ask_id as a positive integer"}
	}
	return id, nil
}

func toolApproveRequest(h *Handlers, r *http.Request, who webserver.Caller, args map[string]any) (any, error) {
	id, err := askIDArg(args)
	if err != nil {
		return nil, err
	}
	note := clean(str(args, "note"), 200)
	summary := map[string]any{"ask_id": id, "will_do": "Approve request #" + strconv.FormatInt(id, 10) + "."}
	if note != "" {
		summary["note"] = note
	}
	out, goAhead, err := h.confirmStep(who, "approve_request", args, summary)
	if err != nil || !goAhead {
		return out, err
	}
	body := map[string]any{}
	if note != "" {
		body["note"] = note
	}
	code, v := h.call(r, who, webserver.ScopeWrite, http.MethodPost, "/api/v1/asks/"+strconv.FormatInt(id, 10)+"/approve", body)
	return h.acted(r, who, "approve_request", args, "ai_ask_approved", "approved", code, v)
}

func toolDeclineRequest(h *Handlers, r *http.Request, who webserver.Caller, args map[string]any) (any, error) {
	id, err := askIDArg(args)
	if err != nil {
		return nil, err
	}
	note := strings.TrimSpace(clean(str(args, "note"), 200))
	summary := map[string]any{"ask_id": id, "will_do": "Decline request #" + strconv.FormatInt(id, 10) + "."}
	if note != "" {
		summary["note"] = note
	}
	out, goAhead, err := h.confirmStep(who, "decline_request", args, summary)
	if err != nil || !goAhead {
		return out, err
	}
	body := map[string]any{}
	if note != "" {
		body["note"] = note
	}
	code, v := h.call(r, who, webserver.ScopeWrite, http.MethodPost, "/api/v1/asks/"+strconv.FormatInt(id, 10)+"/turn-down", body)
	return h.acted(r, who, "decline_request", args, "ai_ask_declined", "declined", code, v)
}
