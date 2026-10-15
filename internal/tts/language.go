package tts

import "strings"

// speechLanguageCode maps common language labels to speech language hints.
// Unknown labels leave the optional hint unset so Gemini can infer the language.
// Codes follow https://ai.google.dev/api/generate-content#SpeechConfig,
// with the existing Bulgarian default retained for compatibility.
func speechLanguageCode(language string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(language), "_", "-")) {
	case "", "bulgarian", "bg", "bg-bg":
		return "bg-BG"
	case "english", "en", "en-us":
		return "en-US"
	case "en-gb":
		return "en-GB"
	case "german", "de", "de-de":
		return "de-DE"
	case "french", "fr", "fr-fr":
		return "fr-FR"
	case "spanish", "es", "es-es":
		return "es-ES"
	case "italian", "it", "it-it":
		return "it-IT"
	case "portuguese", "pt", "pt-br":
		return "pt-BR"
	case "russian", "ru", "ru-ru":
		return "ru-RU"
	case "japanese", "ja", "ja-jp":
		return "ja-JP"
	case "korean", "ko", "ko-kr":
		return "ko-KR"
	case "chinese", "mandarin", "zh", "zh-cn", "cmn", "cmn-cn":
		return "cmn-CN"
	case "arabic", "ar", "ar-xa":
		return "ar-XA"
	case "hindi", "hi", "hi-in":
		return "hi-IN"
	case "turkish", "tr", "tr-tr":
		return "tr-TR"
	default:
		return ""
	}
}
