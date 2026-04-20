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
		return "кирилица"
	case strings.EqualFold(script, "Latin"):
		return "латиница"
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
		return "английски"
	case strings.EqualFold(language, "German"):
		return "немски"
	case strings.EqualFold(language, "French"):
		return "френски"
	default:
		return language
	}
}

func localizedStylePrompt(style, language, script string) string {
	if !strings.EqualFold(script, "Cyrillic") && !strings.EqualFold(language, "Bulgarian") {
		return style
	}
	switch style {
	case "classic American comic book with bold ink outlines, halftone dots, and primary colors":
		return "класически американски комикс с дебели мастилени контури, халф-тонови точки и основни цветове"
	case "Japanese manga with clean linework, expressive eyes, and speed lines":
		return "японска манга с чисти линии, изразителни очи и скоростни линии"
	case "retro 1960s pop art in the style of Roy Lichtenstein with thick outlines and Ben-Day dots":
		return "ретро поп арт от 60-те години с дебели контури и точки в стил Бен-Дей"
	case "watercolor illustration with soft washes, delicate linework, and pastel tones":
		return "акварелна илюстрация с меки петна, фина линия и пастелни тонове"
	case "European bande dessinée with detailed backgrounds, clear lines, and rich flat colors":
		return "европейски bande dessinée с детайлни фонове, ясни линии и наситени плоски цветове"
	case "noir black-and-white graphic novel with heavy shadows and high contrast":
		return "ноар черно-бял графичен роман с тежки сенки и силен контраст"
	case "children's picture book with bright, friendly illustrations and thick outlines":
		return "детска книжка с ярки, приветливи илюстрации и дебели контури"
	case "painterly oil-on-canvas comic with loose brushwork and vivid impressionist colors":
		return "живописен комикс с маслени щрихи и ярки импресионистични цветове"
	case "cyberpunk neon art with glowing outlines, dark backgrounds, and electric accent colors":
		return "киберпънк неонова визия с светещи контури, тъмни фонове и електрически акцентни цветове"
	case "ultra-realistic DSLR photography, cinematic 35mm lens, natural lighting, hyper-detailed textures":
		return "ултрареалистична фотография с кинематографичен 35 мм обектив, естествено осветление и силно детайлни текстури"
	case "cinematic still photography, golden-hour lighting, shallow depth of field, photojournalism quality":
		return "кинематографична фотография с светлина на златен час, плитка дълбочина на полето и качеството на фоторепортаж"
	case "hyper-realistic photography, studio-quality lighting, sharp focus, true-to-life colours and textures":
		return "хиперреалистична фотография със студийно осветление, остър фокус и естествени цветове и текстури"
	default:
		return style
	}
}
