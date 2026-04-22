package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	version "codeberg.org/snonux/comicforge/internal"
	"codeberg.org/snonux/comicforge/internal/comic"
	"codeberg.org/snonux/comicforge/internal/config"
	"codeberg.org/snonux/comicforge/internal/image"
	"codeberg.org/snonux/comicforge/internal/provider"
	textprovider "codeberg.org/snonux/comicforge/internal/text"
	"codeberg.org/snonux/comicforge/internal/tts"
)

type commandDeps struct {
	loadConfig       func(string) (*config.Config, error)
	newTextProvider  func(*config.Config) (provider.TextProvider, error)
	newImageProvider func(*config.Config) (provider.ImageProvider, error)
	newTTSProvider   func(*config.Config, string) (provider.TTSProvider, error)
	newRunner        func(*comic.RunnerConfig) comic.StoryRunner
}

type cliFlags struct {
	vocab            string
	prompt           string
	configPath       string
	promptsDir       string
	outputDir        string
	style            string
	theme            string
	slug             string
	narratorVoice    string
	textModel        string
	imageModel       string
	imageTextModel   string
	ttsModel         string
	textProvider     string
	imageProvider    string
	ttsProvider      string
	narrateEnabled   bool
	ultraRealistic   bool
	noUltraRealistic bool
	version          bool
}

func defaultCommandDeps() commandDeps {
	return commandDeps{
		loadConfig: config.Load,
		newTextProvider: func(cfg *config.Config) (provider.TextProvider, error) {
			return buildTextProvider(cfg)
		},
		newImageProvider: func(cfg *config.Config) (provider.ImageProvider, error) {
			return buildImageProvider(cfg)
		},
		newTTSProvider: func(cfg *config.Config, voice string) (provider.TTSProvider, error) {
			return buildTTSProvider(cfg, voice)
		},
		newRunner: func(cfg *comic.RunnerConfig) comic.StoryRunner {
			return comic.NewRunner(cfg)
		},
	}
}

func newRootCommand() *cobra.Command {
	return newRootCommandWithDeps(defaultCommandDeps())
}

func newRootCommandWithDeps(deps commandDeps) *cobra.Command {
	flags := cliFlags{}
	cmd := &cobra.Command{
		Use:           "comicforge",
		Short:         "ComicForge generates and manages comic output",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.version {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), version.Version)
				return nil
			}
			if cmd.Flags().Changed("prompt") && strings.TrimSpace(flags.prompt) == "" {
				return fmt.Errorf("--prompt is required when set")
			}
			if strings.TrimSpace(flags.prompt) != "" && strings.TrimSpace(flags.vocab) != "" {
				return fmt.Errorf("only one of --prompt and --vocab may be set")
			}
			if strings.TrimSpace(flags.prompt) != "" {
				return runPromptCommand(cmd.Context(), cmd, deps, flags)
			}
			if strings.TrimSpace(flags.vocab) == "" {
				return fmt.Errorf("--vocab is required unless --prompt is set")
			}
			return runCommand(cmd.Context(), cmd, deps, flags)
		},
	}

	cmd.Flags().BoolVarP(&flags.version, "version", "v", false, "print version information")
	cmd.Flags().StringVar(&flags.vocab, "vocab", "", "path to the vocabulary input file")
	cmd.Flags().StringVar(&flags.prompt, "prompt", "", "manual prompt for generating a single image")
	cmd.Flags().StringVar(&flags.configPath, "config", "", "config file (default: search ~/.config/comicforge, $HOME, and .)")
	cmd.Flags().StringVar(&flags.promptsDir, "prompts-dir", "", "directory containing prompt templates")
	cmd.Flags().StringVar(&flags.outputDir, "output", ".", "root output directory for generated comic data")
	cmd.Flags().StringVar(&flags.style, "style", "", "comic art style override")
	cmd.Flags().StringVar(&flags.theme, "theme", "", "story theme override")
	cmd.Flags().BoolVar(&flags.ultraRealistic, "ultra-realistic", false, "force photorealistic rendering")
	cmd.Flags().BoolVar(&flags.noUltraRealistic, "no-ultra-realistic", false, "disable photorealistic rendering")
	cmd.Flags().BoolVar(&flags.narrateEnabled, "narrate", false, "generate narration after the comic")
	cmd.Flags().StringVar(&flags.narratorVoice, "narrator-voice", "", "Gemini voice for narration")
	cmd.Flags().StringVar(&flags.slug, "slug", "", "force the output slug for generated assets")
	cmd.Flags().StringVar(&flags.textProvider, "text-provider", provider.Gemini, "text provider to use")
	cmd.Flags().StringVar(&flags.imageProvider, "image-provider", provider.Gemini, "image provider to use")
	cmd.Flags().StringVar(&flags.ttsProvider, "tts-provider", provider.Gemini, "text-to-speech provider to use")
	cmd.Flags().StringVar(&flags.textModel, "text-model", "", "text model override")
	cmd.Flags().StringVar(&flags.imageModel, "image-model", "", "image model override")
	cmd.Flags().StringVar(&flags.imageTextModel, "image-text-model", "", "image text model override")
	cmd.Flags().StringVar(&flags.ttsModel, "tts-model", "", "text-to-speech model override")

	return cmd
}

func runCommand(ctx context.Context, cmd *cobra.Command, deps commandDeps, flags cliFlags) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if flags.noUltraRealistic && flags.ultraRealistic {
		return fmt.Errorf("only one of --ultra-realistic and --no-ultra-realistic may be set")
	}

	cfg, err := deps.loadConfig(flags.configPath)
	if err != nil {
		return err
	}

	applyConfigOverrides(cmd, cfg, flags)
	voice := resolveNarratorVoice(flags, cfg)

	textProvider, err := deps.newTextProvider(cfg)
	if err != nil {
		return fmt.Errorf("build text provider: %w", err)
	}
	imageProvider, err := deps.newImageProvider(cfg)
	if err != nil {
		return fmt.Errorf("build image provider: %w", err)
	}
	var mainTTSProvider provider.TTSProvider
	if flags.narrateEnabled {
		mainTTSProvider, err = deps.newTTSProvider(cfg, voice)
		if err != nil {
			return fmt.Errorf("build TTS provider: %w", err)
		}
	}

	runner := deps.newRunner(&comic.RunnerConfig{
		TextProvider:          textProvider,
		ImageProvider:         imageProvider,
		MainTTSProvider:       mainTTSProvider,
		ConclusionTTSProvider: mainTTSProvider,
		Prompts:               cfg,
		OutputDir:             flags.outputDir,
		Style:                 flags.style,
		ComicStyles:           cfg.Styles.Comic,
		RealisticStyles:       cfg.Styles.Realistic,
		Theme:                 flags.theme,
		Language:              cfg.Language.Story,
		Script:                cfg.Language.Script,
		NarratorVoice:         voice,
		Slug:                  flags.slug,
		NarrateEnabled:        flags.narrateEnabled,
		UltraRealistic:        resolveUltraRealistic(flags),
		RealisticWeight:       cfg.Story.RealisticWeight,
		AspectRatio:           cfg.Comic.AspectRatio,
		PromptMaxChars:        cfg.Comic.PromptMaxChars,
		PageMaxRetries:        cfg.Comic.PageMaxRetries,
		PageRetryBase:         time.Duration(cfg.Comic.PageRetryBaseSeconds) * time.Second,
		ChunkWords:            cfg.Narration.ChunkWords,
		StoryPages:            cfg.Comic.StoryPages,
		GalleryPages:          cfg.Comic.GalleryPages,
		PanelsPerPage:         cfg.Comic.PanelsPerPage,
	})

	return runner.Run(ctx, flags.vocab)
}

func runPromptCommand(ctx context.Context, cmd *cobra.Command, deps commandDeps, flags cliFlags) error {
	if ctx == nil {
		ctx = context.Background()
	}

	cfg, err := deps.loadConfig(flags.configPath)
	if err != nil {
		return err
	}

	applyConfigOverrides(cmd, cfg, flags)
	imageProvider, err := deps.newImageProvider(cfg)
	if err != nil {
		return fmt.Errorf("build image provider: %w", err)
	}

	runner := deps.newRunner(&comic.RunnerConfig{
		ImageProvider:   imageProvider,
		Prompts:         cfg,
		OutputDir:       flags.outputDir,
		Style:           flags.style,
		ComicStyles:     cfg.Styles.Comic,
		RealisticStyles: cfg.Styles.Realistic,
		Theme:           flags.theme,
		Language:        cfg.Language.Story,
		Script:          cfg.Language.Script,
		Slug:            flags.slug,
		UltraRealistic:  resolveUltraRealistic(flags),
		RealisticWeight: cfg.Story.RealisticWeight,
		AspectRatio:     cfg.Comic.AspectRatio,
		PromptMaxChars:  cfg.Comic.PromptMaxChars,
		PageMaxRetries:  cfg.Comic.PageMaxRetries,
		PageRetryBase:   time.Duration(cfg.Comic.PageRetryBaseSeconds) * time.Second,
	})

	return runner.RunPrompt(ctx, flags.prompt)
}

func applyConfigOverrides(cmd *cobra.Command, cfg *config.Config, flags cliFlags) {
	if cfg == nil {
		return
	}
	if cmd.Flags().Changed("text-provider") {
		cfg.Provider.Text = flags.textProvider
	}
	if cmd.Flags().Changed("image-provider") {
		cfg.Provider.Image = flags.imageProvider
	}
	if cmd.Flags().Changed("tts-provider") {
		cfg.Provider.TTS = flags.ttsProvider
	}
	if cmd.Flags().Changed("text-model") {
		cfg.Models.Text = flags.textModel
	}
	if cmd.Flags().Changed("image-model") {
		cfg.Models.Image = flags.imageModel
	}
	if cmd.Flags().Changed("image-text-model") {
		cfg.Models.ImageText = flags.imageTextModel
	}
	if cmd.Flags().Changed("tts-model") {
		cfg.Models.TTS = flags.ttsModel
	}
	if cmd.Flags().Changed("prompts-dir") {
		cfg.PromptsDir = flags.promptsDir
	}
}

func buildTextProvider(cfg *config.Config) (provider.TextProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	p, err := textprovider.DefaultRegistry().NewFromConfig(cfg)
	if err != nil {
		if errors.Is(err, provider.ErrUnknownProvider) {
			return nil, fmt.Errorf("text provider %q is not implemented", cfg.Provider.Text)
		}
		return nil, err
	}
	return p, nil
}

func buildImageProvider(cfg *config.Config) (provider.ImageProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	p, err := image.DefaultRegistry().NewFromConfig(cfg)
	if err != nil {
		if errors.Is(err, image.ErrUnknownProvider) {
			return nil, fmt.Errorf("image provider %q is not implemented", cfg.Provider.Image)
		}
		return nil, err
	}
	sharedProvider, ok := p.(provider.ImageProvider)
	if !ok {
		return nil, fmt.Errorf("image provider %q does not satisfy the shared provider interface", cfg.Provider.Image)
	}
	return sharedProvider, nil
}

func buildTTSProvider(cfg *config.Config, voice string) (provider.TTSProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	p, err := tts.DefaultRegistry().NewFromConfig(cfg, voice)
	if err != nil {
		if errors.Is(err, provider.ErrUnknownProvider) {
			return nil, fmt.Errorf("tts provider %q is not implemented", cfg.Provider.TTS)
		}
		return nil, err
	}
	return p, nil
}

func resolveUltraRealistic(flags cliFlags) *bool {
	switch {
	case flags.ultraRealistic:
		v := true
		return &v
	case flags.noUltraRealistic:
		v := false
		return &v
	default:
		return nil
	}
}

func resolveNarratorVoice(flags cliFlags, cfg *config.Config) string {
	if strings.TrimSpace(flags.narratorVoice) != "" {
		return strings.TrimSpace(flags.narratorVoice)
	}
	if cfg == nil {
		return ""
	}
	if len(cfg.Narration.Voices) == 0 {
		return ""
	}
	return strings.TrimSpace(cfg.Narration.Voices[0])
}
