// Package config provides application configuration loading and prompt template
// helpers for ComicForge.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/viper"

	"codeberg.org/snonux/comicforge/internal/image"
	"codeberg.org/snonux/comicforge/internal/provider"
	"codeberg.org/snonux/comicforge/internal/text"
	"codeberg.org/snonux/comicforge/internal/tts"
	"codeberg.org/snonux/comicforge/prompts"
)

const (
	// DefaultPromptsDir is the fallback directory for Go template prompt files.
	DefaultPromptsDir = "./prompts"
)

// Config holds ComicForge settings loaded from YAML, environment variables, and defaults.
type Config struct {
	Provider  ProviderConfig  `mapstructure:"provider" yaml:"provider"`
	API       APIConfig       `mapstructure:"api" yaml:"api"`
	Models    ModelConfig     `mapstructure:"models" yaml:"models"`
	Comic     ComicConfig     `mapstructure:"comic" yaml:"comic"`
	Language  LanguageConfig  `mapstructure:"language" yaml:"language"`
	Story     StoryConfig     `mapstructure:"story" yaml:"story"`
	Styles    StyleConfig     `mapstructure:"styles" yaml:"styles"`
	Narration NarrationConfig `mapstructure:"narration" yaml:"narration"`
	PDF       PDFConfig       `mapstructure:"pdf" yaml:"pdf"`

	PromptsDir string `mapstructure:"prompts_dir" yaml:"prompts_dir"`
}

var (
	_ provider.TextConfig  = (*Config)(nil)
	_ provider.ImageConfig = (*Config)(nil)
	_ provider.TTSConfig   = (*Config)(nil)
	_ text.Config          = (*Config)(nil)
	_ image.Config         = (*Config)(nil)
	_ tts.Config           = (*Config)(nil)
)

// ProviderConfig stores the selected provider name for each capability.
type ProviderConfig struct {
	Text  string `mapstructure:"text" yaml:"text"`
	Image string `mapstructure:"image" yaml:"image"`
	TTS   string `mapstructure:"tts" yaml:"tts"`
}

// APIConfig stores API keys and related secrets.
type APIConfig struct {
	GoogleAPIKey string `mapstructure:"google_api_key" yaml:"google_api_key"`
}

// ModelConfig stores the model IDs used by each capability.
type ModelConfig struct {
	Text      string `mapstructure:"text" yaml:"text"`
	Image     string `mapstructure:"image" yaml:"image"`
	ImageText string `mapstructure:"image_text" yaml:"image_text"`
	TTS       string `mapstructure:"tts" yaml:"tts"`
}

// ComicConfig stores comic generation knobs.
type ComicConfig struct {
	StoryPages           int    `mapstructure:"story_pages" yaml:"story_pages"`
	GalleryPages         int    `mapstructure:"gallery_pages" yaml:"gallery_pages"`
	PanelsPerPage        int    `mapstructure:"panels_per_page" yaml:"panels_per_page"`
	AspectRatio          string `mapstructure:"aspect_ratio" yaml:"aspect_ratio"`
	PromptMaxChars       int    `mapstructure:"prompt_max_chars" yaml:"prompt_max_chars"`
	PageMaxRetries       int    `mapstructure:"page_max_retries" yaml:"page_max_retries"`
	PageRetryBaseSeconds int    `mapstructure:"page_retry_base_seconds" yaml:"page_retry_base_seconds"`
}

// LanguageConfig stores language and script labels used by prompts.
type LanguageConfig struct {
	Input  string `mapstructure:"input" yaml:"input"`
	Output string `mapstructure:"output" yaml:"output"`
	Story  string `mapstructure:"story_language" yaml:"story_language"`
	Script string `mapstructure:"script" yaml:"script"`
}

// StoryConfig stores story prompt knobs.
type StoryConfig struct {
	Genres          []string `mapstructure:"genres" yaml:"genres"`
	RealisticWeight float64  `mapstructure:"realistic_weight" yaml:"realistic_weight"`
}

// StyleConfig maps style category keys (YAML keys under styles:) to prompt line pools.
// Known keys include comic, realistic, cartoon, action_90s, manga, horror,
// watercolor, pulp, mecha, pixel_art, ink_wash, and clay; extra keys are allowed.
type StyleConfig map[string][]string

// StyleCategoryOrder is the stable display order for built-in style categories.
var StyleCategoryOrder = []string{
	"comic",
	"realistic",
	"cartoon",
	"action_90s",
	"manga",
	"horror",
	"watercolor",
	"pulp",
	"mecha",
	"pixel_art",
	"ink_wash",
	"clay",
}

// StyleCategoryTitle returns a human-readable label for a style category key.
func StyleCategoryTitle(key string) string {
	switch key {
	case "action_90s":
		return "Action 90s"
	case "pixel_art":
		return "Pixel Art"
	case "ink_wash":
		return "Ink Wash"
	default:
		if key == "" {
			return ""
		}
		parts := strings.Split(key, "_")
		for i, p := range parts {
			if p == "" {
				continue
			}
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
		return strings.Join(parts, " ")
	}
}

// Clone returns a copy of the style map with copied line slices.
func (s StyleConfig) Clone() StyleConfig {
	if s == nil {
		return nil
	}
	out := make(StyleConfig, len(s))
	for k, v := range s {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// Keys returns category keys in StyleCategoryOrder first, then any extras sorted.
func (s StyleConfig) Keys() []string {
	if len(s) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(s))
	keys := make([]string, 0, len(s))
	for _, key := range StyleCategoryOrder {
		if _, ok := s[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var extras []string
	for key := range s {
		if !seen[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	return append(keys, extras...)
}

// NarrationConfig stores narration prompt knobs.
type NarrationConfig struct {
	Voices     []string `mapstructure:"voices" yaml:"voices"`
	ChunkWords int      `mapstructure:"chunk_words" yaml:"chunk_words"`
}

// PDFConfig controls final PDF assembly (ImageMagick).
type PDFConfig struct {
	// Density is the ImageMagick -density value (DPI hint for the PDF).
	Density int `mapstructure:"density" yaml:"density"`
	// JPEGQuality is 0 for default encoding, or 1–100 to JPEG-compress the PDF (smaller files).
	JPEGQuality int `mapstructure:"jpeg_quality" yaml:"jpeg_quality"`
	// Presentation is "none", "print" (matte + ISO A4 portrait pages), or "book" (aged tilt + ISO A4 portrait pages).
	Presentation string `mapstructure:"presentation" yaml:"presentation"`
}

// DefaultConfig returns a configuration populated with the initial Gemini-first defaults.
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderConfig{
			Text:  provider.Gemini,
			Image: provider.Gemini,
			TTS:   provider.Gemini,
		},
		API: APIConfig{},
		Models: ModelConfig{
			Text:      "gemini-2.5-flash",
			Image:     "gemini-3.1-flash-image-preview",
			ImageText: "gemini-2.5-flash",
			TTS:       "gemini-2.5-flash-preview-tts",
		},
		Comic: ComicConfig{
			StoryPages:           5,
			GalleryPages:         5,
			PanelsPerPage:        4,
			AspectRatio:          "16:9",
			PromptMaxChars:       900,
			PageMaxRetries:       5,
			PageRetryBaseSeconds: 15,
		},
		Language: LanguageConfig{
			Input:  "Vocabulary",
			Output: "Story",
			Story:  "Bulgarian",
			Script: "Cyrillic",
		},
		Story: StoryConfig{
			Genres: []string{
				"a warm slice-of-life story",
				"a heartfelt family drama",
				"an exciting science-fiction adventure",
			},
			RealisticWeight: 0.4,
		},
		Styles: StyleConfig{
			"comic": {
				"classic comic book with bold ink outlines",
				"graphic novel with dramatic shadows",
			},
			"realistic": {
				"ultra-realistic DSLR photography, cinematic 35mm lens",
				"cinematic realism with natural light",
			},
			"cartoon": {
				"extremely cartoonish 1930s rubber-hose funny-animal comic style, original mouse-and-duck-era mascot energy, pie-cut eyes, button noses or beaks, white gloves, oversized shoes, noodle arms, squash-and-stretch bodies, round heads, huge expressions, flat candy colors, simple gag staging, absolutely no realism",
				"vintage theatrical funny-animal cartoon comic style, redesign people as original anthropomorphic mascot characters with beaks or round snouts, pie-cut eyes, white gloves, oversized shoes, elastic limbs, cheerful slapstick poses, bold clean outlines, bright flat backgrounds, no realistic anatomy",
			},
			"action_90s": {
				"1990s superhero comic splash-page style like a bold caped hero punching a masked villain in a city, huge muscles, spandex costumes, flowing cape, explosive impact burst, speed lines, smoke, rubble, halftone print texture, heavy black inks, saturated primary colors",
				"classic 90s action superhero cover style, oversized yellow masthead energy, caped spandex hero, armored villain, clenched fists, dramatic foreshortening, city skyline, explosions, speech balloons, thick ink outlines, Ben-Day dots",
			},
			"manga": {
				"true monochrome Japanese shonen manga page, black ink only with grey screentone, no western color rendering, large expressive anime eyes, sharp spiky hair silhouettes, sweat drops, impact bursts, speed-line backgrounds, exaggerated emotion close-ups, dynamic diagonal manga panel language",
				"authentic serialized manga action art, crisp black linework, dense screentone gradients, white paper highlights, hand-drawn Japanese comic energy, chibi reaction inset where appropriate, dramatic sound-effect shapes, cinematic close-ups, no painterly western comic coloring",
			},
			"horror": {
				"vintage horror comic style, eerie shadows, gothic atmosphere, unsettling monsters, dramatic candlelit contrast, suspenseful panel staging",
				"creepy supernatural horror comic style, fog, haunted architecture, anxious expressions, heavy inks, sickly green and crimson accents",
			},
			"watercolor": {
				"pure transparent watercolor storybook comic style, visible cold-press paper grain, watery pigment blooms, uneven wash edges, granulation, soft bleeding colors, pale layered washes, loose pencil underdrawing, no digital airbrush, no oil-paint texture, no glossy comic coloring",
				"hand-painted watercolor children's adventure style, translucent pastel washes, wet-on-wet skies, dry-brush texture, white paper highlights, delicate ink-and-pencil outlines, puddled pigment edges, airy negative space, gentle low-contrast rendering",
			},
			"pulp": {
				"retro pulp adventure cover style, painted poster energy, dramatic rim light, saturated inks, worn paper grain, bold composition, vintage print imperfections",
				"mid-century pulp sci-fi comic style, chromatic airbrushed highlights, atomic-age props, bold diagonals, distressed halftone texture, high adventure mood",
			},
			"mecha": {
				"high-detail mecha comic style, hard-surface armor plates, mechanical joints, cockpit glass reflections, technical line art, industrial lighting",
				"anime mecha action page style, towering robots, dynamic perspective, panel-like motion energy, sparks and smoke, crisp cel-shaded rendering",
			},
			"pixel_art": {
				"16-bit pixel-art comic scene, limited palette, clean pixel clusters, deliberate dithering, retro game-era perspective, sharp pixel edges",
				"retro pixel-art storyboard style, chunky sprites, high-contrast palette ramps, tiled-environment texture, no anti-aliasing, classic console look",
			},
			"ink_wash": {
				"ink-wash brush painting style, sumi-e inspired strokes, expressive black ink gradients, controlled bleed on paper, minimal but dramatic composition",
				"monochrome ink-and-wash comic style, calligraphic brush lines, soft gray washes, negative-space storytelling, textured rice-paper feel",
			},
			"clay": {
				"stop-motion clay diorama style, hand-sculpted clay characters, thumbprint texture, miniature set lighting, tactile handcrafted look",
				"plasticine comic tableau style, sculpted figures, soft studio shadows, practical miniature props, playful clay surface detail",
			},
		},
		Narration: NarrationConfig{
			Voices: []string{
				"Charon",
				"Fenrir",
			},
			ChunkWords: 100,
		},
		PDF: PDFConfig{
			Density:      150,
			JPEGQuality:  0,
			Presentation: "none",
		},
		PromptsDir: DefaultPromptsDir,
	}
}

// Load reads configuration from YAML, environment variables, and defaults.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetEnvPrefix("COMICFORGE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	setDefaults(v, cfg)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		if homeDir, err := HomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".config", "comicforge"))
			v.AddConfigPath(homeDir)
		}
		v.AddConfigPath(".")
		v.SetConfigName("config")
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if configPath != "" || !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	applyEnvFallbacks(cfg)
	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// TextProviderName returns the configured text provider name.
func (c *Config) TextProviderName() string {
	return provider.NormalizeName(c.Provider.Text)
}

// ImageProviderName returns the configured image provider name.
func (c *Config) ImageProviderName() string {
	return provider.NormalizeName(c.Provider.Image)
}

// GoogleAPIKey returns the configured Google API key.
func (c *Config) GoogleAPIKey() string {
	if c == nil {
		return ""
	}
	return c.API.GoogleAPIKey
}

// TextModel returns the configured text model name.
func (c *Config) TextModel() string {
	if c == nil {
		return ""
	}
	return c.Models.Text
}

// ImageModel returns the configured image model name.
func (c *Config) ImageModel() string {
	if c == nil {
		return ""
	}
	return c.Models.Image
}

// ImageTextModel returns the configured image-text model name.
func (c *Config) ImageTextModel() string {
	if c == nil {
		return ""
	}
	return c.Models.ImageText
}

// ComicAspectRatio returns the configured comic image aspect ratio.
func (c *Config) ComicAspectRatio() string {
	if c == nil {
		return ""
	}
	return c.Comic.AspectRatio
}

// TTSModel returns the configured text-to-speech model name.
func (c *Config) TTSModel() string {
	if c == nil {
		return ""
	}
	return c.Models.TTS
}

// TTSProviderName returns the configured TTS provider name.
func (c *Config) TTSProviderName() string {
	return provider.NormalizeName(c.Provider.TTS)
}

// PromptDir returns the configured prompts directory or the default fallback.
func (c *Config) PromptDir() string {
	if c == nil || strings.TrimSpace(c.PromptsDir) == "" {
		return DefaultPromptsDir
	}
	return strings.TrimSpace(c.PromptsDir)
}

// PromptPath joins the configured prompts directory with the requested template file.
func (c *Config) PromptPath(name string) string {
	return filepath.Join(c.PromptDir(), name)
}

// LoadPromptTemplate parses a Go text/template prompt file from the configured prompts directory.
func (c *Config) LoadPromptTemplate(name string) (*template.Template, error) {
	path := c.PromptPath(name)
	content, err := os.ReadFile(path)
	if err == nil {
		return parsePromptTemplate(path, content)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read prompt %q: %w", path, err)
	}

	content, err = prompts.Read(name)
	if err != nil {
		return nil, fmt.Errorf("read embedded prompt %q: %w", name, err)
	}

	return parsePromptTemplate(filepath.Base(name), content)
}

// RenderPrompt executes a prompt template with the provided data.
func (c *Config) RenderPrompt(name string, data any) (string, error) {
	tmpl, err := c.LoadPromptTemplate(name)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("render prompt %q: %w", name, err)
	}

	return builder.String(), nil
}

func parsePromptTemplate(name string, content []byte) (*template.Template, error) {
	tmpl, err := template.New(filepath.Base(name)).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse prompt %q: %w", name, err)
	}

	return tmpl, nil
}

func (c *Config) normalize() {
	c.Provider.Text = provider.NormalizeName(c.Provider.Text)
	c.Provider.Image = provider.NormalizeName(c.Provider.Image)
	c.Provider.TTS = provider.NormalizeName(c.Provider.TTS)

	if c.Provider.Text == "" {
		c.Provider.Text = provider.Gemini
	}
	if c.Provider.Image == "" {
		c.Provider.Image = provider.Gemini
	}
	if c.Provider.TTS == "" {
		c.Provider.TTS = provider.Gemini
	}

	if c.PromptsDir == "" {
		c.PromptsDir = DefaultPromptsDir
	}

	if c.PDF.Density <= 0 {
		c.PDF.Density = 150
	}
	if c.PDF.JPEGQuality < 0 {
		c.PDF.JPEGQuality = 0
	}
	if c.PDF.JPEGQuality > 100 {
		c.PDF.JPEGQuality = 100
	}
	c.PDF.Presentation = strings.ToLower(strings.TrimSpace(c.PDF.Presentation))
	if c.PDF.Presentation == "" {
		c.PDF.Presentation = "none"
	}
}

func (c *Config) validate() error {
	if !text.DefaultRegistry().Has(c.Provider.Text) {
		return fmt.Errorf("unknown text provider: %s", c.Provider.Text)
	}
	if !image.DefaultRegistry().Has(c.Provider.Image) {
		return fmt.Errorf("unknown image provider: %s", c.Provider.Image)
	}
	if !tts.DefaultRegistry().Has(c.Provider.TTS) {
		return fmt.Errorf("unknown TTS provider: %s", c.Provider.TTS)
	}

	switch c.PDF.Presentation {
	case "none", "print", "book":
	default:
		return fmt.Errorf("unknown pdf.presentation %q (use none, print, or book)", c.PDF.Presentation)
	}

	return nil
}

func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault("provider.text", cfg.Provider.Text)
	v.SetDefault("provider.image", cfg.Provider.Image)
	v.SetDefault("provider.tts", cfg.Provider.TTS)

	v.SetDefault("api.google_api_key", cfg.API.GoogleAPIKey)

	v.SetDefault("models.text", cfg.Models.Text)
	v.SetDefault("models.image", cfg.Models.Image)
	v.SetDefault("models.image_text", cfg.Models.ImageText)
	v.SetDefault("models.tts", cfg.Models.TTS)

	v.SetDefault("comic.story_pages", cfg.Comic.StoryPages)
	v.SetDefault("comic.gallery_pages", cfg.Comic.GalleryPages)
	v.SetDefault("comic.panels_per_page", cfg.Comic.PanelsPerPage)
	v.SetDefault("comic.aspect_ratio", cfg.Comic.AspectRatio)
	v.SetDefault("comic.prompt_max_chars", cfg.Comic.PromptMaxChars)
	v.SetDefault("comic.page_max_retries", cfg.Comic.PageMaxRetries)
	v.SetDefault("comic.page_retry_base_seconds", cfg.Comic.PageRetryBaseSeconds)

	v.SetDefault("language.input", cfg.Language.Input)
	v.SetDefault("language.output", cfg.Language.Output)
	v.SetDefault("language.story_language", cfg.Language.Story)
	v.SetDefault("language.script", cfg.Language.Script)

	v.SetDefault("story.genres", cfg.Story.Genres)
	v.SetDefault("story.realistic_weight", cfg.Story.RealisticWeight)

	for key, lines := range cfg.Styles {
		v.SetDefault("styles."+key, lines)
	}

	v.SetDefault("narration.voices", cfg.Narration.Voices)
	v.SetDefault("narration.chunk_words", cfg.Narration.ChunkWords)

	v.SetDefault("pdf.density", cfg.PDF.Density)
	v.SetDefault("pdf.jpeg_quality", cfg.PDF.JPEGQuality)
	v.SetDefault("pdf.presentation", cfg.PDF.Presentation)

	v.SetDefault("prompts_dir", cfg.PromptsDir)
}

func applyEnvFallbacks(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.API.GoogleAPIKey == "" {
		cfg.API.GoogleAPIKey = os.Getenv("GOOGLE_API_KEY")
	}
}
