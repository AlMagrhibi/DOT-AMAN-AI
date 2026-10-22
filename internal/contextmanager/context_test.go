package contextmanager

import (
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/model"
	"strings"
	"testing"
)

func TestMandatoryBudget(t *testing.T) {
	h := make([]model.Message, 100)
	for i := range h {
		h[i] = model.Message{Role: "user", Content: strings.Repeat("ع", 100)}
	}
	m, err := Build("SAFETY", "current", h, "reference", 512, 128)
	if err != nil {
		t.Fatal(err)
	}
	n := 32
	for _, x := range m {
		n += Estimate(x.Content)
	}
	if n > 384 || m[0].Content != "SAFETY" || m[len(m)-1].Content != "current" {
		t.Fatal(n, m)
	}
	if _, err = Build(strings.Repeat("x", 1000), "x", nil, "", 512, 128); err == nil {
		t.Fatal("overflow")
	}
}

func TestTokenizerEstimatorPreservesRequiredMessages(t *testing.T) {
	history := []model.Message{{Role: "user", Content: strings.Repeat("word ", 40)}, {Role: "assistant", Content: "recent"}}
	// A deterministic local tokenizer stand-in: one token per five bytes.
	cheap := func(s string) int { return (len(s)+4)/5 + 1 }
	m, err := BuildWithEstimator("SAFETY", "current", history, "reference", 64, 16, cheap)
	if err != nil {
		t.Fatal(err)
	}
	if m[0].Role != "system" || m[len(m)-1].Content != "current" {
		t.Fatalf("required messages missing: %+v", m)
	}
	for _, x := range m {
		if cheap(x.Content) > 64-16 {
			t.Fatal("message exceeded custom token budget")
		}
	}
}
