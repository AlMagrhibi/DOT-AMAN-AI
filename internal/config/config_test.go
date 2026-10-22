package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEndpoint(t *testing.T) {
	for _, x := range []string{"https://example.com", "http://localhost:8081", "http://127.0.0.1.evil", "http://user@127.0.0.1", "http://127.0.0.1/x", "http://192.168.1.1"} {
		if ValidateEndpoint(x) == nil {
			t.Fatal(x)
		}
	}
	for _, x := range []string{"http://127.0.0.1:8081", "http://[::1]:8081"} {
		if err := ValidateEndpoint(x); err != nil {
			t.Fatal(err)
		}
	}
}
func TestUnknownConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.toml")
	if err := os.WriteFile(p, []byte("unknown=true"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Fatal("accepted unknown setting")
	}
}
