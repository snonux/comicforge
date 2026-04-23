package comic

import "testing"

func TestIsDINA4ClassPDFPresentation(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"print", true},
		{"PRINT", true},
		{" book ", true},
		{"book", true},
		{"none", false},
		{"", false},
		{"fullbleed", false},
	}
	for _, tt := range tests {
		if got := IsDINA4ClassPDFPresentation(tt.in); got != tt.want {
			t.Errorf("IsDINA4ClassPDFPresentation(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsBookPDFPresentation(t *testing.T) {
	if !IsBookPDFPresentation("book") {
		t.Fatal("IsBookPDFPresentation(book) = false")
	}
	if IsBookPDFPresentation("print") {
		t.Fatal("IsBookPDFPresentation(print) = true, want false")
	}
	if IsBookPDFPresentation("none") {
		t.Fatal("IsBookPDFPresentation(none) = true, want false")
	}
}
