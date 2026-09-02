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

	"codeberg.org/snonux/comicforge/internal/comic"
	"codeberg.org/snonux/comicforge/internal/provider"
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
)

// NameCheck reports whether a provider name is known for a capability.
type NameCheck func(name string) bool

// LoadOption customizes configuration loading.
type LoadOption func(*loadOptions)

type loadOptions struct {
	textKnown  NameCheck
	imageKnown NameCheck
	ttsKnown   NameCheck
}

// WithProviderNameChecks injects validators used to reject unknown provider names.
// A nil check skips validation for that capability so config stays free of
// concrete provider-package dependencies.
func WithProviderNameChecks(text, image, tts NameCheck) LoadOption {
	return func(o *loadOptions) {
		o.textKnown = text
		o.imageKnown = image
		o.ttsKnown = tts
	}
}

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
		Styles: StyleConfig(comic.DefaultStyleLines()),
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
func Load(configPath string, opts ...LoadOption) (*Config, error) {
	cfg := DefaultConfig()

	options := loadOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

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
	if err := cfg.validate(options); err != nil {
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

func (c *Config) validate(opts loadOptions) error {
	if opts.textKnown != nil && !opts.textKnown(c.Provider.Text) {
		return fmt.Errorf("unknown text provider: %s", c.Provider.Text)
	}
	if opts.imageKnown != nil && !opts.imageKnown(c.Provider.Image) {
		return fmt.Errorf("unknown image provider: %s", c.Provider.Image)
	}
	if opts.ttsKnown != nil && !opts.ttsKnown(c.Provider.TTS) {
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
