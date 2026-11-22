package mediafiles

import (
	"testing"

	"github.com/HolyArrs/holyarrs/app/backend/internal/core/releasename"
)

func TestSpanishRegionalPickRules(t *testing.T) {
	if !langMatches("es", "es-es") || !langMatches("es", "es-la") || !langMatches("es-es", "es-es") {
		t.Fatal("expected base es and exact es-es to match")
	}
	if langMatches("es-es", "es-la") || langMatches("es-es", "es") || langMatches("es-la", "es-es") {
		t.Fatal("regional mismatch should not match")
	}
	if LanguageName("es-es") != "Spanish (Spain)" || LanguageName("es-la") != "Spanish (Latin America)" {
		t.Fatalf("names %q %q", LanguageName("es-es"), LanguageName("es-la"))
	}

	rules := PickRulesBody{
		Steps:        []Step{{Quality: "1080p_web", Allowed: true}},
		GoodEnoughAt: "1080p_web",
		Protocol:     ProtocolRule{Prefer: "either"},
		Order:        "quality_then_points",
		ProperPolicy: ProperPolicy{Mode: "take_any"},
		Language:     LanguageRule{Wanted: []string{"es-es"}, OriginalOK: false},
	}
	if err := CheckPickRules("Castellano", []string{"movie"}, rules); err != nil {
		t.Fatalf("CheckPickRules rejected es-es: %v", err)
	}

	cast := releasename.Parse("Pelicula.2024.Castellano.1080p.WEB-DL-GRP", releasename.Hints{})
	lat := releasename.Parse("Pelicula.2024.Latino.1080p.WEB-DL-GRP", releasename.Hints{})
	title := &TitleRef{OriginalLanguage: "en"}

	vCast := Verdict{}
	vCast.language(cast, rules, title)
	if len(vCast.Reasons) != 1 || vCast.Reasons[0].Effect != "pass" {
		t.Fatalf("castellano should pass es-es rule: %+v", vCast.Reasons)
	}

	vLat := Verdict{}
	vLat.language(lat, rules, title)
	if len(vLat.Reasons) != 1 || vLat.Reasons[0].Effect != "fail" {
		t.Fatalf("latino should fail es-es rule: %+v", vLat.Reasons)
	}

	// Generic "es" rule still accepts both Castellano and Latino.
	rulesAnyEs := rules
	rulesAnyEs.Language.Wanted = []string{"es"}
	vAny := Verdict{}
	vAny.language(cast, rulesAnyEs, title)
	if len(vAny.Reasons) != 1 || vAny.Reasons[0].Effect != "pass" {
		t.Fatalf("castellano should pass generic es rule: %+v", vAny.Reasons)
	}
}
