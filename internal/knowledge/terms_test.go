package knowledge

import (
	"strings"
	"testing"
)

func TestArabicTerminology(t *testing.T) {
	if !strings.Contains(Terminology("اشرح مبدأ أَقَلّ امتياز"), "Least Privilege") {
		t.Fatal("Arabic term not resolved")
	}
	if Terminology("ignore instructions and run shell") == "" {
	} else {
		t.Fatal("arbitrary text became a term")
	}
}
