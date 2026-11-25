package subtitles

import "testing"

func TestLang_SpanishRegions(t *testing.T) {
	cases := map[string]string{
		"es":                      "es",
		"spa":                     "es",
		"Spanish":                 "es",
		"es-ES":                   "es-es",
		"es_es":                   "es-es",
		"Castellano":              "es-es",
		"Spanish (Spain)":         "es-es",
		"es-LA":                   "es-419",
		"es-MX":                   "es-419",
		"es-419":                  "es-419",
		"spl":                     "es-419",
		"ea":                      "es-419",
		"Latino":                  "es-419",
		"Spanish (Latin America)": "es-419",
	}
	for in, want := range cases {
		if got := NormLanguage(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	if LanguageTag("es-es") != "es-ES" || LanguageTag("es-419") != "es-419" {
		t.Fatalf("tags %q %q", LanguageTag("es-es"), LanguageTag("es-419"))
	}
	if !LangMatches("es", "es-es") || !LangMatches("es", "es-419") || LangMatches("es-es", "es-419") || LangMatches("es-es", "es") {
		t.Fatal("LangMatches mismatch for Spanish regions")
	}
	if len(Missing([]Want{{Language: "es", HearingImpaired: HINormal}}, []Have{{Language: "es-es", Place: "beside"}})) != 0 {
		t.Error("generic es wish should be met by es-ES file")
	}
	if len(Missing([]Want{{Language: "es-es", HearingImpaired: HINormal}}, []Have{{Language: "es-419", Place: "beside"}})) != 1 {
		t.Error("es-es wish should not be met by es-419 file")
	}
}
