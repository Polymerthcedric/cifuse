package initwf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Polymerthcedric/cifuse/internal/config"
)

const WorkflowName = "cifuse.yml"

// Action pins below are shared with scripts/ and the README so templates and
// docs never drift from what cifuse itself would recommend.
const pinnedVersion = "v0.2.0"

const workflowTemplate = `name: cifuse

on:
  pull_request:
  push:
    branches:
      - main

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
    steps:
      - name: Checkout
        uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4
      - name: Install cifuse
        run: curl -fsSL https://raw.githubusercontent.com/Polymerthcedric/cifuse/main/scripts/install.sh | bash -s -- -v %s
      - name: Audit workflows
        run: "$HOME/.local/bin/cifuse" audit --format sarif . > cifuse.sarif
      - name: Upload SARIF to code scanning
        uses: github/codeql-action/c20e34f438d671fc35777cc9820dd7adf8252874 # v3
        with:
          sarif_file: cifuse.sarif
`

func workflow() string {
	return fmt.Sprintf(workflowTemplate, pinnedVersion)
}

const configTemplate = `# cifuse configuration (https://github.com/Polymerthcedric/cifuse)
# ignore:
#   rules:
#     - CF001
#   paths:
#     - .github/workflows/legacy.yml
#   findings:
#     - path: .github/workflows/archive.yml
#       rule: CF003
`

func Generate(root string, force bool) ([]string, error) {
	workflowsDir := filepath.Join(root, ".github", "workflows")
	workflowPath := filepath.Join(workflowsDir, WorkflowName)
	if _, err := os.Stat(workflowPath); err == nil && !force {
		return nil, fmt.Errorf("%s already exists; use --force to overwrite", workflowPath)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	created := []string{}
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(workflowPath, []byte(workflow()), 0o644); err != nil {
		return nil, err
	}
	created = append(created, workflowPath)

	configPath := filepath.Join(root, config.Filename)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(configTemplate), 0o644); err != nil {
			return nil, err
		}
		created = append(created, configPath)
	}
	return created, nil
}
