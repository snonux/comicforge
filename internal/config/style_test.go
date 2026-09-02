package config

import (
	"reflect"
	"testing"
)

func TestStyleCategoryTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want string
	}{
		{"comic", "Comic"},
		{"action_90s", "Action 90s"},
		{"pixel_art", "Pixel Art"},
		{"ink_wash", "Ink Wash"},
		{"neon_noir", "Neon Noir"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := StyleCategoryTitle(tt.key); got != tt.want {
			t.Fatalf("StyleCategoryTitle(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestStyleConfigKeysOrder(t *testing.T) {
	t.Parallel()

	styles := StyleConfig{
		"clay":      {"c"},
		"comic":     {"a"},
		"neon_noir": {"n"},
		"zebra":     {"z"},
		"manga":     {"m"},
	}
	got := styles.Keys()
	want := []string{"comic", "manga", "clay", "neon_noir", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys() = %#v, want %#v", got, want)
	}
}

func TestStyleConfigClone(t *testing.T) {
	t.Parallel()

	src := StyleConfig{"comic": {"a"}}
	dst := src.Clone()
	dst["comic"][0] = "b"
	dst["extra"] = []string{"x"}
	if src["comic"][0] != "a" {
		t.Fatal("Clone shared slice with source")
	}
	if _, ok := src["extra"]; ok {
		t.Fatal("Clone leaked key into source")
	}
	if StyleConfig(nil).Clone() != nil {
		t.Fatal("nil Clone should return nil")
	}
}

func TestStyleConfigKeysNilEmpty(t *testing.T) {
	t.Parallel()

	if got := StyleConfig(nil).Keys(); got != nil {
		t.Fatalf("nil Keys() = %#v, want nil", got)
	}
	if got := (StyleConfig{}).Keys(); got != nil {
		t.Fatalf("empty Keys() = %#v, want nil", got)
	}
}
