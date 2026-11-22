package releasename

import (
	"slices"
	"testing"
)

func TestSpanishVariants(t *testing.T) {
	cases := []struct {
		name  string
		hints Hints
		title string
		langs []string
		group string
	}{
		{"Pelicula.2024.Castellano.1080p.WEB-DL.H264-GRP", Hints{}, "Pelicula", []string{"es-es"}, "GRP"},
		{"Pelicula.2024.1080p.WEB-DL.ES-ES-GRP", Hints{}, "Pelicula", []string{"es-es"}, "GRP"},
		{"Pelicula.2024.1080p.WEB-DL.ES-ES", Hints{}, "Pelicula", []string{"es-es"}, ""},
		{"Pelicula.2024.1080p.DUAL.CAST.x265-GRP", Hints{}, "Pelicula", []string{"es-es"}, "GRP"},
		{"Pelicula.2024.Latino.1080p.WEB-DL.H264-GRP", Hints{}, "Pelicula", []string{"es-la"}, "GRP"},
		{"Pelicula.2024.ES-LA.1080p.WEB-DL-GRP", Hints{}, "Pelicula", []string{"es-la"}, "GRP"},
		{"Pelicula.2024.ES-MX.1080p.WEB-DL-GRP", Hints{}, "Pelicula", []string{"es-la"}, "GRP"},
		{"Pelicula.2024.1080p.DUAL.LAT.x265-GRP", Hints{}, "Pelicula", []string{"es-la"}, "GRP"},
		{"Pelicula.2024.ESP.1080p.WEB-DL-GRP", Hints{}, "Pelicula", []string{"es"}, "GRP"},
		{"Pelicula.2024.Espanol.1080p.WEB-DL-GRP", Hints{}, "Pelicula", []string{"es"}, "GRP"},
		{"Pelicula.2024.Español.1080p.WEB-DL-GRP", Hints{}, "Pelicula", []string{"es"}, "GRP"},
		{"Serie.S01E02.Castellano.1080p.WEB-DL-GRP", Hints{Show: true}, "Serie", []string{"es-es"}, "GRP"},
		{"Serie.S01E02.Latino.1080p.WEB-DL-GRP", Hints{Show: true}, "Serie", []string{"es-la"}, "GRP"},
	}
	for _, c := range cases {
		r := Parse(c.name, c.hints)
		if r.Title.Value != c.title || !slices.Equal(r.Languages, c.langs) || r.Group.Value != c.group {
			t.Errorf("%s: title=%q langs=%v group=%q; want title=%q langs=%v group=%q",
				c.name, r.Title.Value, r.Languages, r.Group.Value, c.title, c.langs, c.group)
		}
	}
	if !ReadsLanguage("es-es") || !ReadsLanguage("es-la") || !ReadsLanguage("es") {
		t.Error("ReadsLanguage should know es, es-es and es-la")
	}
}

func TestSpanishVariants_NoFalsePositives(t *testing.T) {
	cases := []struct {
		name  string
		title string
	}{
		{"Cast.Away.2000.1080p.BluRay.x264-GRP", "Cast Away"},
		{"El.Ritmo.Latino.2020.1080p.WEB-DL-GRP", "El Ritmo Latino"},
		{"Pelicula.2024.1080p.VOSE.WEB-DL-GRP", "Pelicula"},
		{"Pelicula.2024.1080p.SUBESP.WEB-DL-GRP", "Pelicula"},
		{"Pelicula.2024.1080p.Castellano.Subs-GRP", "Pelicula"},
		{"Pelicula.2024.1080p.Latino.Sub-GRP", "Pelicula"},
	}
	for _, c := range cases {
		r := Parse(c.name, Hints{})
		if r.Title.Value != c.title || len(r.Languages) != 0 {
			t.Errorf("%s: title=%q langs=%v; want title=%q langs=[]", c.name, r.Title.Value, r.Languages, c.title)
		}
	}
}
