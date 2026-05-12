package auth

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestConfigRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OSM_CONFIG_PATH", filepath.Join(dir, "config.json"))

	if _, err := LoadConfig(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist, got %v", err)
	}
	in := &PersistedConfig{ClientID: "abcd1234"}
	if err := SaveConfig(in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.ClientID != "abcd1234" {
		t.Errorf("got %q, want abcd1234", out.ClientID)
	}
}
