package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// DeepL talks to the DeepL REST API (Free https://api-free.deepl.com or Pro
// https://api.deepl.com). Key is sent in the Authorization header as a
// DeepL-Auth-Key token.
type DeepL struct {
	URL, Key string
	HTTP     *http.Client
	Timeout  time.Duration
}

const deeplMaxBatch = 50

func deeplEndpoint(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	switch {
	case strings.HasSuffix(b, "/v2/translate"):
		return b
	case strings.HasSuffix(b, "/v2"):
		return b + "/translate"
	default:
		return b + "/v2/translate"
	}
}

func deeplTarget(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "es-es", "es":
		return "ES"
	case "es-la":
		return "ES-419"
	case "pt-br":
		return "PT-BR"
	case "pt":
		return "PT-PT"
	case "en":
		return "EN-US"
	case "zh":
		return "ZH-HANS"
	default:
		return strings.ToUpper(strings.TrimSpace(code))
	}
}

func deeplSource(code string) string {
	base, _, _ := strings.Cut(strings.TrimSpace(code), "-")
	return strings.ToUpper(base)
}

// Translate sends up to 50 lines per DeepL call and maps translations back in
// strict 1:1 order so cue numbering never drifts.
func (d *DeepL) Translate(ctx context.Context, b Batch) ([]Line, error) {
	if len(b.Lines) == 0 {
		return nil, nil
	}
	hdr := map[string]string{}
	if d.Key != "" {
		hdr["Authorization"] = "DeepL-Auth-Key " + d.Key
	}
	addr := deeplEndpoint(d.URL)
	src := deeplSource(b.From)
	dst := deeplTarget(b.To)
	var all []Line
	for i := 0; i < len(b.Lines); i += deeplMaxBatch {
		part := b.Lines[i:min(i+deeplMaxBatch, len(b.Lines))]
		texts := make([]string, len(part))
		for k, l := range part {
			texts[k] = l.Text
		}
		payload := map[string]any{
			"text":                texts,
			"target_lang":         dst,
			"preserve_formatting": true,
		}
		if src != "" {
			payload["source_lang"] = src
		}
		body, _ := json.Marshal(payload)
		out, err := send(ctx, d.HTTP, http.MethodPost, addr, hdr, body, d.Timeout, d.Key)
		if err != nil {
			return nil, err
		}
		var resp struct {
			Translations []struct {
				Text string
			}
		}
		if err := json.Unmarshal(out, &resp); err != nil || len(resp.Translations) == 0 {
			return nil, &Error{Kind: KindUnreadable, Msg: "the DeepL answer held no translated lines"}
		}
		for k, tr := range resp.Translations {
			if k >= len(part) {
				break
			}
			all = append(all, Line{N: part[k].N, Text: tr.Text})
		}
	}
	return all, nil
}
