package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWorkflow(t *testing.T, root, name, content string) {
	t.Helper()
	directory := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunFindsMissingGuardrails(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, "test.yml", "name: test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps: []\n")
	report, err := Run(root, Ignore{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 3 {
		t.Fatalf("got %d findings, want 3: %#v", len(report.Findings), report.Findings)
	}
}

func TestRunPassesProtectedWorkflow(t *testing.T) {
	root := t.TempDir()
	workflow := "name: test\npermissions:\n  contents: read\nconcurrency:\n  group: ${{ github.workflow }}-${{ github.ref }}\n  cancel-in-progress: true\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    timeout-minutes: 10\n    steps: []\n"
	writeWorkflow(t, root, "test.yml", workflow)
	report, err := Run(root, Ignore{})
	if err != nil {
		t.Fatal(err)
	}
	if report.HasFailures() {
		t.Fatalf("unexpected findings: %#v", report.Findings)
	}
}

func TestRunIgnoresRulesAndPaths(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, "main.yml", "name: test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps: []\n")
	writeWorkflow(t, root, "legacy.yml", "name: test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    timeout-minutes: 10\n    steps: []\n")
	report, err := Run(root, Ignore{
		Rules: []string{"CF001", "CF002", "CF003"},
		Paths: []string{".github/workflows/legacy.yml"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("got findings %#v, want none", report.Findings)
	}
}

func TestRunIgnoresSpecificFinding(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, "main.yml", "name: test\npermissions:\n  contents: read\nconcurrency:\n  group: x\n  cancel-in-progress: true\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    timeout-minutes: 10\n    steps: []\n  test:\n    runs-on: ubuntu-latest\n    steps: []\n")
	report, err := Run(root, Ignore{
		Findings: []FindingPattern{
			{Path: ".github/workflows/main.yml", Rule: "CF003", Line: 13},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("got findings %#v, want none", report.Findings)
	}
}

func TestSARIFShape(t *testing.T) {
	report := Report{
		Workflows: 1,
		Findings: []Finding{
			{Rule: "CF001", Path: ".github/workflows/main.yml", Line: 1, Message: "add workflow concurrency"},
			{Rule: "CF003", Path: ".github/workflows/main.yml", Line: 5, Message: "job needs timeout"},
		},
	}
	out, err := report.SARIF("v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name    string `json:"name"`
					Version string `json:"version"`
					Rules   []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID string `json:"ruleId"`
				Level  string `json:"level"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Version != "2.1.0" || len(doc.Runs) != 1 {
		t.Fatalf("unexpected SARIF envelope: %s", string(out))
	}
	driver := doc.Runs[0].Tool.Driver
	if driver.Name != "cifuse" || driver.Version != "v0.2.0" {
		t.Fatalf("unexpected driver: %s %s", driver.Name, driver.Version)
	}
	if len(driver.Rules) != 2 || len(doc.Runs[0].Results) != 2 {
		t.Fatalf("unexpected rules/results: %s", string(out))
	}
	if doc.Runs[0].Results[0].Level != "error" {
		t.Fatalf("expected error level, got %s", doc.Runs[0].Results[0].Level)
	}
}

func TestGeneratedWorkflowPassesItsOwnAudit(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowPath, 0o755); err != nil {
		t.Fatal(err)
	}
	workflow := `name: cifuse

on:
  pull_request:
  push:
    branches: [main]

concurrency:
  group: cifuse-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read
  security-events: write

jobs:
  guardrails:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps: []
`
	if err := os.WriteFile(filepath.Join(workflowPath, "cifuse.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Run(root, Ignore{})
	if err != nil {
		t.Fatal(err)
	}
	if report.HasFailures() {
		t.Fatalf("generated workflow should pass its own audit, got %#v", report.Findings)
	}
}

func TestTextOutputHasCTALine(t *testing.T) {
	report := Report{Workflows: 1, Findings: []Finding{{Rule: "CF001", Path: "x.yml", Line: 1, Message: "m"}}}
	if text := report.Text(); !strings.Contains(text, "starter.fidelco.dev/buy") {
		t.Fatalf("text output should point at the starter: %q", text)
	}
}
