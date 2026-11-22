package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// LibreTranslate talks to a LibreTranslate server (self-hosted or hosted).
// Key is optional and only sent when configured.
type LibreTranslate struct {
	URL, Key string
	HTTP     *http.Client
	Timeout  time.Duration
}

func libreEndpoint(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(b, "/translate") {
		return b
	}
	return b + "/translate"
}

func libreLang(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	if c == "" {
		return "auto"
	}
	base, _, _ := strings.Cut(c, "-")
	return base
}

// Translate posts the batch texts to /translate and reads the array (or single
// string) reply back in order.
func (l *LibreTranslate) Translate(ctx context.Context, b Batch) ([]Line, error) {
	if len(b.Lines) == 0 {
		return nil, nil
	}
	texts := make([]string, len(b.Lines))
	for i, x := range b.Lines {
		texts[i] = x.Text
	}
	payload := map[string]any{
		"q":      texts,
		"source": libreLang(b.From),
		"target": libreLang(b.To),
		"format": "text",
	}
	if l.Key != "" {
		payload["api_key"] = l.Key
	}
	body, _ := json.Marshal(payload)
	out, err := send(ctx, l.HTTP, http.MethodPost, libreEndpoint(l.URL), nil, body, l.Timeout, l.Key)
	if err != nil {
		return nil, err
	}
	var resp struct {
		TranslatedText json.RawMessage
	}
	if err := json.Unmarshal(out, &resp); err != nil || len(resp.TranslatedText) == 0 {
		return nil, &Error{Kind: KindUnreadable, Msg: "the LibreTranslate answer held no translated lines"}
	}
	var list []string
	if err := json.Unmarshal(resp.TranslatedText, &list); err != nil {
		var single string
		if len(b.Lines) == 1 && json.Unmarshal(resp.TranslatedText, &single) == nil {
			list = []string{single}
		} else {
			return nil, &Error{Kind: KindUnreadable, Msg: "the LibreTranslate answer could not be read"}
		}
	}
	if len(list) == 0 {
		return nil, &Error{Kind: KindUnreadable, Msg: "the LibreTranslate answer held no translated lines"}
	}
	got := make([]Line, 0, min(len(list), len(b.Lines)))
	for i, txt := range list {
		if i >= len(b.Lines) {
			break
		}
		got = append(got, Line{N: b.Lines[i].N, Text: txt})
	}
	return got, nil
}
