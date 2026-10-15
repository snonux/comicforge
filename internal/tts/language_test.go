package tts

import "testing"

func TestSpeechLanguageCode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		language string
		want     string
	}{
		{"", "bg-BG"},
		{"   ", "bg-BG"},
		{"Bulgarian", "bg-BG"},
		{"bg", "bg-BG"},
		{"BG_bg", "bg-BG"},
		{"English", "en-US"},
		{"en", "en-US"},
		{" EN_us ", "en-US"},
		{"en-GB", "en-GB"},
		{"German", "de-DE"},
		{"fr", "fr-FR"},
		{"Spanish", "es-ES"},
		{"it-IT", "it-IT"},
		{"Portuguese", "pt-BR"},
		{"pt-PT", ""},
		{"Russian", "ru-RU"},
		{"Japanese", "ja-JP"},
		{"ko", "ko-KR"},
		{"Chinese", "cmn-CN"},
		{"Mandarin", "cmn-CN"},
		{"zh", "cmn-CN"},
		{"zh-CN", "cmn-CN"},
		{"cmn", "cmn-CN"},
		{"cmn-CN", "cmn-CN"},
		{"Arabic", "ar-XA"},
		{"ar", "ar-XA"},
		{"ar-XA", "ar-XA"},
		{"ar-EG", ""},
		{"hi-IN", "hi-IN"},
		{"Turkish", "tr-TR"},
		{"Unknown language", ""},
		{"xx-XX", ""},
	} {
		t.Run(tc.language, func(t *testing.T) {
			t.Parallel()
			if got := speechLanguageCode(tc.language); got != tc.want {
				t.Fatalf("speechLanguageCode(%q) = %q, want %q", tc.language, got, tc.want)
			}
		})
	}
}
