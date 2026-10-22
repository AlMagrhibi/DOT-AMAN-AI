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
