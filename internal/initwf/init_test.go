package initwf

import (
	"regexp"
	"testing"
)

// The generated workflow must use the `owner/action@<immutable sha>` form.
// A bare SHA or a missing action name silently breaks `cifuse init` output.
func TestWorkflowUsesValidPinnedActions(t *testing.T) {
	workflow := workflow()
	uses := regexp.MustCompile(`(?m)^\s+uses: ([^@\s]+)@([0-9a-f]{40}) # v[0-9.]+$`).FindAllStringSubmatch(workflow, -1)
	if len(uses) != 2 {
		t.Fatalf("expected 2 pinned actions in generated workflow, got %d:\n%s", len(uses), workflow)
	}
	for _, match := range uses {
		if match[1] == "" || match[1] == "github/codeql-action" {
			t.Fatalf("action name missing in %q", match[0])
		}
	}
	if !regexp.MustCompile(`uses: github/codeql-action/upload-sarif@`).MatchString(workflow) {
		t.Fatal("generated workflow must use codeql-action/upload-sarif")
	}
}
