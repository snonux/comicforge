package comic

import "strings"

// localizedBrandName returns the comic's subtitle/series name in the selected script.
func localizedBrandName(language, script string) string {
	if strings.EqualFold(script, "Cyrillic") || strings.EqualFold(language, "Bulgarian") {
		return "КомиксФордж Приключения"
	}
	return "ComicForge Adventures"
}
