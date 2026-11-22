package subtitles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "github.com/HolyArrs/holyarrs/app/backend/internal/core/model"
	"github.com/HolyArrs/holyarrs/app/backend/internal/core/secrets"
)

func TestDeepLAndLibreTranslateServiceFlow(t *testing.T) {
	ctx := context.Background()
	f := newFix(t)
	v, err := secrets.Open(ctx, f.st, make([]byte, secrets.KeySize))
	if err != nil {
		t.Fatal(err)
	}

	deeplSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "DeepL-Auth-Key fake-deepl:fx" {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
		var req struct {
			Text []string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		trs := make([]map[string]string, len(req.Text))
		for i, s := range req.Text {
			trs[i] = map[string]string{"text": "ES:" + s}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": trs})
	}))
	defer deeplSrv.Close()

	s := &Service{Store: f.st, Vault: v, TranslateHTTP: deeplSrv.Client()}
	tr, err := s.SaveTranslator(ctx, TranslatorInput{Kind: TranslatorDeepL, URL: deeplSrv.URL, Key: "fake-deepl:fx"}, domain.System)
	if err != nil {
		t.Fatalf("SaveTranslator(deepl): %v", err)
	}
	if tr.Kind != TranslatorDeepL || !tr.HasKey || !tr.Ready() || tr.Name() != "DeepL" {
		t.Fatalf("unexpected deepl translator: %+v", tr)
	}
	if res := s.TestTranslator(ctx); !res.OK {
		t.Fatalf("TestTranslator(deepl) failed: %+v", res)
	}

	ltSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Q []string `json:"q"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		out := make([]string, len(req.Q))
		for i, s := range req.Q {
			out[i] = "LT:" + s
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translatedText": out})
	}))
	defer ltSrv.Close()

	s.TranslateHTTP = ltSrv.Client()
	tr, err = s.SaveTranslator(ctx, TranslatorInput{Kind: TranslatorLibreTranslate, URL: ltSrv.URL, ClearKey: true}, domain.System)
	if err != nil {
		t.Fatalf("SaveTranslator(libretranslate): %v", err)
	}
	if tr.Kind != TranslatorLibreTranslate || tr.HasKey || !tr.Ready() || tr.Name() != "LibreTranslate" {
		t.Fatalf("unexpected libretranslate translator: %+v", tr)
	}
	if res := s.TestTranslator(ctx); !res.OK {
		t.Fatalf("TestTranslator(libretranslate) failed: %+v", res)
	}
}
