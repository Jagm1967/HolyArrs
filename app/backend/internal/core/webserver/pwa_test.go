package webserver

import (
	"mime"
	"strings"
	"testing"
)

func TestPWAManifestSupport(t *testing.T) {
	if !IsAsset("/manifest.webmanifest") {
		t.Fatal("/manifest.webmanifest must be treated as a public static asset")
	}
	if ct := mime.TypeByExtension(".webmanifest"); !strings.HasPrefix(ct, "application/manifest+json") {
		t.Fatalf("webmanifest content-type = %q, want application/manifest+json", ct)
	}
	html := []byte(`<link rel="manifest" href="/manifest.webmanifest" />`)
	got := string(WithBase(html, "/holyarrs"))
	if !strings.Contains(got, `href="/holyarrs/manifest.webmanifest"`) {
		t.Fatalf("WithBase did not rewrite manifest href: %s", got)
	}
}
