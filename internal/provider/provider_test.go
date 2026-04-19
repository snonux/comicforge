package provider

import "testing"

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	if got, want := NormalizeName(" Gemini "), Gemini; got != want {
		t.Fatalf("NormalizeName() = %q, want %q", got, want)
	}
}

func TestIsKnownName(t *testing.T) {
	t.Parallel()

	if !IsKnownName("gemini") {
		t.Fatal("expected gemini to be recognized")
	}
	if IsKnownName("bogus") {
		t.Fatal("expected bogus provider to be rejected")
	}
}
