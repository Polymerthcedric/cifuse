package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".cifuserc.yml")
	content := "ignore:\n  rules:\n    - CF001\n    - CF003\n  paths:\n    - .github/workflows/ci.yml\n  findings:\n    - path: .github/workflows/main.yml\n      rule: CF005\n      line: 10\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ignore, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ignore.Rules) != 2 || ignore.Rules[0] != "CF001" {
		t.Fatalf("unexpected rules: %v", ignore.Rules)
	}
	if len(ignore.Paths) != 1 || ignore.Paths[0] != ".github/workflows/ci.yml" {
		t.Fatalf("unexpected paths: %v", ignore.Paths)
	}
	if len(ignore.Findings) != 1 || ignore.Findings[0].Rule != "CF005" || ignore.Findings[0].Line != 10 {
		t.Fatalf("unexpected findings: %v", ignore.Findings)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".cifuserc.yml")
	if err := os.WriteFile(path, []byte("ignore: {rules: [}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".cifuserc.yml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	ignore, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ignore.Rules) != 0 {
		t.Fatalf("expected empty rules, got %v", ignore.Rules)
	}
}
