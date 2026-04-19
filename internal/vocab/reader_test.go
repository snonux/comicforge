package vocab

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadVocabularyFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fileContent string
		want        []WordEntry
	}{
		{
			name:        "empty file",
			fileContent: "",
			want:        nil,
		},
		{
			name:        "words with translations",
			fileContent: "ябълка = apple\nкотка = cat\nкуче = dog",
			want: []WordEntry{
				{Word: "ябълка", Translation: "apple", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "котка", Translation: "cat", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "куче", Translation: "dog", NeedsTranslation: false, CardType: CardTypeTranslation},
			},
		},
		{
			name: "mixed format",
			fileContent: `ябълка
котка = cat
куче
хляб = bread`,
			want: []WordEntry{
				{Word: "ябълка", Translation: "", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "котка", Translation: "cat", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "куче", Translation: "", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "хляб", Translation: "bread", NeedsTranslation: false, CardType: CardTypeTranslation},
			},
		},
		{
			name:        "windows line endings",
			fileContent: "ябълка\r\nкотка = cat\r\nкуче",
			want: []WordEntry{
				{Word: "ябълка", Translation: "", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "котка", Translation: "cat", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "куче", Translation: "", NeedsTranslation: false, CardType: CardTypeTranslation},
			},
		},
		{
			name:        "multiple equals signs",
			fileContent: "test = word = with = equals",
			want: []WordEntry{
				{Word: "test", Translation: "word = with = equals", NeedsTranslation: false, CardType: CardTypeTranslation},
			},
		},
		{
			name: "english only format",
			fileContent: `= apple
= cat
= dog`,
			want: []WordEntry{
				{Word: "", Translation: "apple", NeedsTranslation: true, CardType: CardTypeTranslation},
				{Word: "", Translation: "cat", NeedsTranslation: true, CardType: CardTypeTranslation},
				{Word: "", Translation: "dog", NeedsTranslation: true, CardType: CardTypeTranslation},
			},
		},
		{
			name: "word definition format",
			fileContent: `ябълка == плод
котка == домашно животно`,
			want: []WordEntry{
				{Word: "ябълка", Translation: "плод", NeedsTranslation: false, CardType: CardTypeDefinition},
				{Word: "котка", Translation: "домашно животно", NeedsTranslation: false, CardType: CardTypeDefinition},
			},
		},
		{
			name: "mixed formats",
			fileContent: `ябълка = apple
котка == домашно животно
= dog
стол`,
			want: []WordEntry{
				{Word: "ябълка", Translation: "apple", NeedsTranslation: false, CardType: CardTypeTranslation},
				{Word: "котка", Translation: "домашно животно", NeedsTranslation: false, CardType: CardTypeDefinition},
				{Word: "", Translation: "dog", NeedsTranslation: true, CardType: CardTypeTranslation},
				{Word: "стол", Translation: "", NeedsTranslation: false, CardType: CardTypeTranslation},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "vocab.txt")
			if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0o644); err != nil {
				t.Fatalf("write temp file: %v", err)
			}

			got, err := ReadVocabularyFile(tmpFile)
			if err != nil {
				t.Fatalf("ReadVocabularyFile() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ReadVocabularyFile() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestReadVocabularyFile_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := ReadVocabularyFile("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
