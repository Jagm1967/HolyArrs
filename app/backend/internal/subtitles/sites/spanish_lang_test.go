package sites

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/HolyArrs/holyarrs/app/backend/internal/subtitles/sites/fakesite"
)

func TestSiteLang_SpanishRegions(t *testing.T) {
	for in, want := range map[string]string{
		"es": "es", "es-ES": "es-es", "es_es": "es-es", "Castellano": "es-es",
		"es-LA": "es-419", "es-MX": "es-419", "es-419": "es-419", "EA": "es-419", "SPL": "es-419",
	} {
		if got := siteLang(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	if !langWanted([]string{"es"}, "es-es") || !langWanted([]string{"es"}, "es-419") || langWanted([]string{"es-es"}, "es-419") {
		t.Error("langWanted mismatch for Spanish regions")
	}
}

func TestOpenSubtitles_SpanishRegions(t *testing.T) {
	f := osFake()
	f.Subs = append(f.Subs,
		fakesite.Sub{ID: "201", Kind: "movie", Language: "es", Title: "The Lantern", Year: 2010, Text: fakesite.Cue("hola", 1000)},
		fakesite.Sub{ID: "202", Kind: "movie", Language: "ea", Title: "The Lantern", Year: 2010, Text: fakesite.Cue("hola", 1000)},
	)
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	s, err := New("opensubtitles", Config{Name: "OS", URL: srv.URL + "/api/v1", Key: fakeKey, GapMS: 1, AllowLocal: true})
	if err != nil {
		t.Fatal(err)
	}
	esES, err := s.Search(context.Background(), Query{Kind: "movie", Title: "The Lantern", Year: 2010, Languages: []string{"es-es"}})
	if err != nil || len(esES) != 1 || esES[0].Language != "es-es" {
		t.Fatalf("es-es: %+v %v", esES, err)
	}
	esLA, err := s.Search(context.Background(), Query{Kind: "movie", Title: "The Lantern", Year: 2010, Languages: []string{"es-419"}})
	if err != nil || len(esLA) != 1 || esLA[0].Language != "es-419" {
		t.Fatalf("es-419: %+v %v", esLA, err)
	}
}
