package comic

import (
	"fmt"
	"strings"
)

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

func localizedLanguageName(language, script string) string {
	if strings.EqualFold(script, "Cyrillic") || strings.EqualFold(language, "Bulgarian") {
		return "български"
	}
	switch {
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

// localizedStylePrompt returns the style prompt in the active script.
// In Cyrillic/Bulgarian mode every style must have a co-located localization;
// missing entries panic so English cannot silently leak into Cyrillic output.
func localizedStylePrompt(style, language, script string) string {
	if !needsCyrillicStyle(language, script) {
		return style
	}
	if style == "" {
		return style
	}
	localized, ok := cyrillicStyleByEnglish[style]
	if !ok {
		panic(fmt.Sprintf("comic: no Cyrillic localization for style %q", style))
	}
	return localized
}
