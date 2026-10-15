package comic

import "strings"

// localizedBrandName returns the comic's subtitle/series name in the selected script.
func localizedBrandName(language, script string) string {
	if strings.EqualFold(script, "Cyrillic") || strings.EqualFold(language, "Bulgarian") {
		return "КомиксФордж Приключения"
	}
	return "ComicForge Adventures"
}

func localizedScriptName(script string) string {
	switch {
	case strings.EqualFold(script, "Cyrillic"):
		return "Cyrillic"
	case strings.EqualFold(script, "Latin"):
		return "Latin"
	default:
		return script
	}
}

// localizedLanguageName names the requested language independently of its script.
func localizedLanguageName(language, _ string) string {
	switch {
	case strings.EqualFold(language, "Bulgarian"):
		return "български"
	case strings.EqualFold(language, "English"):
		return "English"
	case strings.EqualFold(language, "German"):
		return "German"
	case strings.EqualFold(language, "French"):
		return "French"
	default:
		return language
	}
}

func needsCyrillicStyle(language, script string) bool {
	return strings.EqualFold(script, "Cyrillic") || strings.EqualFold(language, "Bulgarian")
}

// localizedStylePrompt localizes built-in style instructions when available.
// Custom styles remain prompt instructions; the image templates separately
// constrain visible text to the requested language and script.
func localizedStylePrompt(style, language, script string) string {
	if !needsCyrillicStyle(language, script) {
		return style
	}
	if style == "" {
		return style
	}
	if localized, ok := cyrillicStyleByEnglish[style]; ok {
		return localized
	}
	return style
}
