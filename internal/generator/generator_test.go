package generator_test

import (
	"strings"
	"testing"

	"github.com/Sneypsleep90/go-url-shortner/internal/generator"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	code, err := generator.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 10 {
		t.Fatalf("code length = %d, want 10", len(code))
	}
	for _, char := range code {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_", char) {
			t.Fatalf("code contains invalid character %q", char)
		}
	}
}
