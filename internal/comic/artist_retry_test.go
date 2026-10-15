package comic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type retryImageProvider func(context.Context) error

func (retryImageProvider) Name() string       { return "retry-test" }
func (retryImageProvider) IsAvailable() error { return nil }
func (p retryImageProvider) GenerateImage(ctx context.Context, _, _ string) error {
	return p(ctx)
}

func TestGenerateWithRetryStopsBeforeCanceledAttempt(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%t", deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			} else {
				cancel()
			}
			defer cancel()
			calls := 0
			a := newRetryTestArtist(t, func(context.Context) error {
				calls++
				return errors.New("provider failure")
			})
			a.sleep = func(context.Context, time.Duration) error {
				t.Fatal("unexpected retry wait")
				return nil
			}
			err := a.generateWithRetry(ctx, "prompt", "unused.png", "cover", nil)
			if !errors.Is(err, ctx.Err()) || calls != 0 {
				t.Fatalf("error = %v, calls = %d; want %v and no calls", err, calls, ctx.Err())
			}
		})
	}
}

func TestGenerateWithRetryStopsWhenCanceledDuringAPI(t *testing.T) {
	for _, providerErr := range []error{context.Canceled, errors.New("provider failure"), nil} {
		t.Run(fmt.Sprint(providerErr), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			a := newRetryTestArtist(t, func(callCtx context.Context) error {
				calls++
				cancel()
				<-callCtx.Done()
				return providerErr
			})
			a.validateImage = func(context.Context, string, string, string) error {
				t.Fatal("unexpected validation after cancellation")
				return nil
			}
			a.sleep = func(context.Context, time.Duration) error {
				t.Fatal("unexpected retry wait after cancellation")
				return nil
			}
			err := a.generateWithRetry(ctx, "prompt", filepath.Join(t.TempDir(), "out.png"), "cover", nil)
			if !errors.Is(err, context.Canceled) || calls != 1 {
				t.Fatalf("error = %v, calls = %d; want cancellation and one call", err, calls)
			}
		})
	}
}

func TestRenderPageRemovesUnvalidatedOutputAfterCancellation(t *testing.T) {
	for _, providerErr := range []error{nil, context.Canceled} {
		t.Run(fmt.Sprint(providerErr), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			outputDir := t.TempDir()
			outputFile := filepath.Join(outputDir, "cover.png")
			calls := 0
			a := newRetryTestArtist(t, func(context.Context) error {
				calls++
				if err := os.WriteFile(outputFile, []byte("unvalidated image"), 0o644); err != nil {
					return err
				}
				if calls == 1 {
					cancel()
					return providerErr
				}
				return nil
			})
			a.outputDir = outputDir
			validations := 0
			a.validateImage = func(context.Context, string, string, string) error {
				validations++
				return nil
			}
			if _, err := a.renderPage(ctx, "cover", coverPromptTemplate, nil, "cover", nil); !errors.Is(err, context.Canceled) {
				t.Fatalf("renderPage() error = %v, want cancellation", err)
			}
			if _, err := os.Stat(outputFile); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unvalidated output remains: stat error = %v", err)
			}
			if calls != 1 || validations != 0 {
				t.Fatalf("calls = %d, validations = %d; want 1 and 0", calls, validations)
			}
			if _, err := a.renderPage(context.Background(), "cover", coverPromptTemplate, nil, "cover", nil); err != nil {
				t.Fatalf("resumed renderPage() error = %v", err)
			}
			if calls != 2 || validations != 1 {
				t.Fatalf("resumed calls = %d, validations = %d; want 2 and 1", calls, validations)
			}
		})
	}
}

func TestGenerateWithRetryCancellationInterruptsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	a := newRetryTestArtist(t, func(context.Context) error {
		calls++
		return errors.New("transient failure")
	})
	a.pageRetryBase = time.Hour
	sleeper := a.sleep
	waiting := make(chan struct{})
	a.sleep = func(ctx context.Context, pause time.Duration) error {
		close(waiting)
		return sleeper(ctx, pause)
	}
	result := make(chan error, 1)
	go func() {
		result <- a.generateWithRetry(ctx, "prompt", "unused.png", "cover", nil)
	}()
	select {
	case <-waiting:
	case <-time.After(2 * time.Second):
		t.Fatal("retry did not enter backoff")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("error = %v, calls = %d; want cancellation and one call", err, calls)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not interrupt backoff")
	}
}

func TestGenerateWithRetryStopsWhenParentDeadlineExpires(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	calls := 0
	a := newRetryTestArtist(t, func(callCtx context.Context) error {
		calls++
		<-callCtx.Done()
		return callCtx.Err()
	})
	a.sleep = func(context.Context, time.Duration) error {
		t.Fatal("unexpected retry wait after parent deadline")
		return nil
	}
	err := a.generateWithRetry(ctx, "prompt", "unused.png", "cover", nil)
	if !errors.Is(err, context.DeadlineExceeded) || calls > 1 {
		t.Fatalf("error = %v, calls = %d; want parent deadline and at most one call", err, calls)
	}
}

func TestGenerateWithRetryChecksCancellationBeforeNextAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	a := newRetryTestArtist(t, func(context.Context) error {
		calls++
		return errors.New("transient failure")
	})
	a.sleep = func(context.Context, time.Duration) error {
		cancel()
		return nil
	}
	err := a.generateWithRetry(ctx, "prompt", "unused.png", "cover", nil)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("error = %v, calls = %d; want cancellation and one call", err, calls)
	}
}

func TestGenerateWithRetryRetriesTransientAttemptErrors(t *testing.T) {
	for _, attemptErr := range []error{errors.New("transient failure"), context.DeadlineExceeded, context.Canceled} {
		t.Run(attemptErr.Error(), func(t *testing.T) {
			calls := 0
			a := newRetryTestArtist(t, func(context.Context) error {
				calls++
				if calls < 3 {
					return fmt.Errorf("attempt failed: %w", attemptErr)
				}
				return nil
			})
			var pauses []time.Duration
			a.sleep = func(ctx context.Context, pause time.Duration) error {
				pauses = append(pauses, pause)
				return ctx.Err()
			}
			err := a.generateWithRetry(context.Background(), "prompt", filepath.Join(t.TempDir(), "out.png"), "cover", nil)
			if err != nil || calls != 3 {
				t.Fatalf("error = %v, calls = %d; want success after three calls", err, calls)
			}
			if len(pauses) != 2 || pauses[0] != time.Second || pauses[1] != 2*time.Second {
				t.Fatalf("retry pauses = %v, want [1s 2s]", pauses)
			}
		})
	}
}

func TestGenerateWithRetrySupportsNilContextAndSleeper(t *testing.T) {
	calls := 0
	a := newRetryTestArtist(t, func(ctx context.Context) error {
		if ctx == nil {
			t.Fatal("provider received nil context")
		}
		calls++
		if calls == 1 {
			return errors.New("transient failure")
		}
		return nil
	})
	a.sleep = nil
	a.pageRetryBase = time.Nanosecond
	//nolint:staticcheck // Preserve the existing nil-context compatibility path.
	if err := a.generateWithRetry(nil, "prompt", "unused.png", "cover", nil); err != nil || calls != 2 {
		t.Fatalf("error = %v, calls = %d; want success after two calls", err, calls)
	}
}

func TestGenerateWithRetryStopsWhenCanceledDuringValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	a := newRetryTestArtist(t, func(context.Context) error {
		calls++
		return nil
	})
	a.validateImage = func(context.Context, string, string, string) error {
		cancel()
		return context.Canceled
	}
	a.sleep = func(context.Context, time.Duration) error {
		t.Fatal("unexpected retry wait after cancellation")
		return nil
	}
	err := a.generateWithRetry(ctx, "prompt", filepath.Join(t.TempDir(), "out.png"), "cover", nil)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("error = %v, calls = %d; want cancellation and one call", err, calls)
	}
}

func TestWaitForImageRetryFinishes(t *testing.T) {
	if err := waitForImageRetry(context.Background(), 0); err != nil {
		t.Fatalf("waitForImageRetry() = %v", err)
	}
}

func newRetryTestArtist(t *testing.T, generate retryImageProvider) *Artist {
	t.Helper()
	a := NewArtist(&ArtistConfig{
		ImageProvider:  generate,
		Prompts:        fakePromptRenderer{},
		PageMaxRetries: 3,
		PageRetryBase:  time.Second,
	})
	a.validateImage = nil
	return a
}
