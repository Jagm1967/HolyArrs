package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeepLTranslateSpanishVariants(t *testing.T) {
	var gotAuth, gotPath, gotTarget, gotSource string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		var req struct {
			Text       []string `json:"text"`
			SourceLang string   `json:"source_lang"`
			TargetLang string   `json:"target_lang"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotSource = req.SourceLang
		gotTarget = req.TargetLang
		trs := make([]map[string]string, len(req.Text))
		for i, s := range req.Text {
			trs[i] = map[string]string{"text": "[" + req.TargetLang + "] " + s}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": trs})
	}))
	defer srv.Close()

	c := &DeepL{URL: srv.URL, Key: "demo-key:fx", HTTP: srv.Client()}
	res, err := Run(context.Background(), c, Job{
		Batch: Batch{
			From: "en", To: "es-la",
			Lines: []Line{{N: 1, Text: "Good morning."}, {N: 2, Text: "See you tomorrow."}},
		},
	})
	if err != nil {
		t.Fatalf("DeepL Run failed: %v", err)
	}
	if gotAuth != "DeepL-Auth-Key demo-key:fx" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotPath != "/v2/translate" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotSource != "EN" || gotTarget != "ES-419" {
		t.Fatalf("langs = %q -> %q", gotSource, gotTarget)
	}
	if res.Translated != 2 || res.Lines[0] != "[ES-419] Good morning." || res.Lines[1] != "[ES-419] See you tomorrow." {
		t.Fatalf("unexpected result: %+v", res)
	}

	if _, err := c.Translate(context.Background(), Batch{From: "en", To: "es-es", Lines: []Line{{N: 1, Text: "Hi"}}}); err != nil {
		t.Fatalf("es-es failed: %v", err)
	}
	if gotTarget != "ES" {
		t.Fatalf("es-es target = %q, want ES", gotTarget)
	}
}

func TestLibreTranslateBatch(t *testing.T) {
	var gotPath, gotSource, gotTarget, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var req struct {
			Q      []string `json:"q"`
			Source string   `json:"source"`
			Target string   `json:"target"`
			APIKey string   `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotSource, gotTarget, gotKey = req.Source, req.Target, req.APIKey
		out := make([]string, len(req.Q))
		for i, s := range req.Q {
			out[i] = "es:" + s
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translatedText": out})
	}))
	defer srv.Close()

	c := &LibreTranslate{URL: srv.URL, Key: "lt-secret", HTTP: srv.Client()}
	res, err := Run(context.Background(), c, Job{
		Batch: Batch{
			From: "en", To: "es-es",
			Lines: []Line{{N: 10, Text: "Hello"}, {N: 11, Text: "World"}},
		},
	})
	if err != nil {
		t.Fatalf("LibreTranslate Run failed: %v", err)
	}
	if gotPath != "/translate" || gotSource != "en" || gotTarget != "es" || gotKey != "lt-secret" {
		t.Fatalf("unexpected request: path=%q src=%q dst=%q key=%q", gotPath, gotSource, gotTarget, gotKey)
	}
	if res.Translated != 2 || res.Lines[0] != "es:Hello" || res.Lines[1] != "es:World" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
