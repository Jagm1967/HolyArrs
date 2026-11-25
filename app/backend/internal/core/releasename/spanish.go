package releasename

func init() {
	// Regional Spanish tags that do not collide with ordinary title words.
	langWords["castellano"] = "es-es"
	langWords["es-es"] = "es-es"
	langWords["es-la"] = "es-la"
	langWords["es-mx"] = "es-la"
	langWords["es-419"] = "es-la"

	// Spanish words checked only after the title/year so titles stay intact.
	moreLangWords["espanol"] = "es"
	moreLangWords["español"] = "es"
	moreLangWords["esp"] = "es"
	moreLangWords["cast"] = "es-es"
	moreLangWords["latino"] = "es-la"
	moreLangWords["lat"] = "es-la"
}

// isSubtitleNote reports whether token i is immediately followed by a subtitle
// marker ("Castellano Subs", "Latino Sub"), so it is not read as an audio track.
func isSubtitleNote(low []string, i int) bool {
	if i+1 >= len(low) {
		return false
	}
	switch low[i+1] {
	case "sub", "subs", "subtitle", "subtitles", "subtitulado", "subtitulos":
		return true
	}
	return false
}

// guardedMoreLangOK checks short ambiguous tokens ("cast", "lat") that
// need a tag before them so title words are not misread as audio languages.
func guardedMoreLangOK(t string, low []string, i int) bool {
	if isSubtitleNote(low, i) {
		return false
	}
	switch t {
	case "cast", "lat":
		return i > 0 && !plainWord(low[i-1])
	}
	return true
}
