package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunFindsMissingGuardrails(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	workflow := "name: test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps: []\n"
	if err := os.WriteFile(filepath.Join(directory, "test.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 3 {
		t.Fatalf("got %d findings, want 3: %#v", len(report.Findings), report.Findings)
	}
}

func TestRunPassesProtectedWorkflow(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	workflow := "name: test\npermissions:\n  contents: read\nconcurrency:\n  group: ${{ github.workflow }}-${{ github.ref }}\n  cancel-in-progress: true\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    timeout-minutes: 10\n    steps: []\n"
	if err := os.WriteFile(filepath.Join(directory, "test.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.HasFailures() {
		t.Fatalf("unexpected findings: %#v", report.Findings)
	}
}
