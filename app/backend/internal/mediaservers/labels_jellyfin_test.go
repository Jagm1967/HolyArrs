package mediaservers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	domain "github.com/HolyArrs/holyarrs/app/backend/internal/core/model"
)

func TestLabelJellyfinAddsTagsAndPreservesExisting(t *testing.T) {
	var mu sync.Mutex
	items := map[string]map[string]any{
		"jf-movie-1": {
			"Id":          "jf-movie-1",
			"Name":        "The Matrix",
			"Type":        "Movie",
			"Overview":    "Keep this field",
			"ProviderIds": map[string]string{"Tmdb": "603"},
			"Tags":        []string{"Favs"},
		},
		"jf-show-1": {
			"Id":          "jf-show-1",
			"Name":        "Demo Show",
			"Type":        "Series",
			"ProviderIds": map[string]string{"Tvdb": "100"},
			"Tags":        []string{},
		},
	}
	posts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Authorization"), tok) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/Users":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"Id": "u-normal", "Policy": map[string]any{"IsAdministrator": false}},
				{"Id": "u-admin", "Policy": map[string]any{"IsAdministrator": true}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/Items":
			list := make([]map[string]any, 0, len(items))
			for _, it := range items {
				list = append(list, it)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": list})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/Users/u-admin/Items/"):
			id := strings.TrimPrefix(r.URL.Path, "/Users/u-admin/Items/")
			it, ok := items[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(it)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/Items/"):
			id := strings.TrimPrefix(r.URL.Path, "/Items/")
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			items[id] = body
			posts++
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	w := newWorld(t)
	movie, show := watchTitles(t, w)
	if _, err := Add(context.Background(), w.s, w.v, Input{Name: "Jellyfin", Kind: "jellyfin", URL: srv.URL, Token: tok}, domain.System); err != nil {
		t.Fatal(err)
	}
	if err := SaveLabelSettings(context.Background(), w.s, LabelSettings{Enabled: true, Label: "requested", Askers: true}, domain.System); err != nil {
		t.Fatal(err)
	}
	l := &Linker{Store: w.s, Vault: w.v, Asked: func(_ context.Context, _ string) ([]AskedTitle, error) {
		return []AskedTitle{{TitleID: movie, Askers: []string{"ana"}}, {TitleID: show, Askers: []string{"bo"}}}, nil
	}}

	got, err := l.LabelPlex(context.Background(), true)
	if err != nil || got != 2 {
		t.Fatalf("first pass: got %d err %v", got, err)
	}
	mu.Lock()
	movieTags, _ := items["jf-movie-1"]["Tags"].([]any)
	movieOverview, _ := items["jf-movie-1"]["Overview"].(string)
	showTags, _ := items["jf-show-1"]["Tags"].([]any)
	firstPosts := posts
	mu.Unlock()

	if movieOverview != "Keep this field" {
		t.Fatalf("existing metadata lost: overview=%q", movieOverview)
	}
	toStrings := func(xs []any) []string {
		out := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if got := toStrings(movieTags); !slices.Equal(got, []string{"Favs", "requested", "ana"}) {
		t.Fatalf("movie tags = %v", got)
	}
	if got := toStrings(showTags); !slices.Equal(got, []string{"requested", "bo"}) {
		t.Fatalf("show tags = %v", got)
	}
	if firstPosts != 2 {
		t.Fatalf("posts = %d, want 2", firstPosts)
	}

	// Second pass must be a no-op since both titles were recorded as tagged.
	got2, err := l.LabelPlex(context.Background(), true)
	if err != nil || got2 != 0 {
		t.Fatalf("second pass: got %d err %v", got2, err)
	}
}
