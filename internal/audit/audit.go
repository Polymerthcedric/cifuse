package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Finding struct {
	Rule    string `json:"rule"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type FindingPattern struct {
	Path string `yaml:"path"`
	Rule string `yaml:"rule"`
	Line int    `yaml:"line"`
}

type Ignore struct {
	Rules    []string         `yaml:"rules"`
	Paths    []string         `yaml:"paths"`
	Findings []FindingPattern `yaml:"findings"`
}

type Report struct {
	Workflows int       `json:"workflows"`
	Findings  []Finding `json:"findings"`
}

func (r Report) HasFailures() bool { return len(r.Findings) > 0 }

func (r Report) Text() string {
	var out strings.Builder
	if len(r.Findings) == 0 {
		fmt.Fprintf(&out, "PASS  %d workflow(s) checked; no guardrail gaps found.\n", r.Workflows)
	} else {
		fmt.Fprintf(&out, "FAIL  %d finding(s) across %d workflow(s)\n", len(r.Findings), r.Workflows)
		for _, finding := range r.Findings {
			fmt.Fprintf(&out, "%s  %s:%d  %s\n", finding.Rule, finding.Path, finding.Line, finding.Message)
		}
	}
	out.WriteString("Hardened CI, from scratch? https://starter.fidelco.dev/buy\n")
	return out.String()
}

var jobHeader = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*(?:#.*)?$`)

func Run(root string, ignore Ignore) (Report, error) {
	workflowDirectory := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(workflowDirectory)
	if os.IsNotExist(err) {
		return Report{}, nil
	}
	if err != nil {
		return Report{}, err
	}

	report := Report{}
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml")) {
			continue
		}
		path := filepath.Join(workflowDirectory, entry.Name())
		findings, err := auditFile(root, path)
		if err != nil {
			return Report{}, err
		}
		report.Workflows++
		report.Findings = append(report.Findings, findings...)
	}
	report.Findings = filterFindings(report.Findings, ignore)
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Path == report.Findings[j].Path {
			return report.Findings[i].Line < report.Findings[j].Line
		}
		return report.Findings[i].Path < report.Findings[j].Path
	})
	return report, nil
}

func filterFindings(findings []Finding, ignore Ignore) []Finding {
	if len(ignore.Rules) == 0 && len(ignore.Paths) == 0 && len(ignore.Findings) == 0 {
		return findings
	}
	filtered := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if contains(ignore.Rules, finding.Rule) {
			continue
		}
		if contains(ignore.Paths, finding.Path) {
			continue
		}
		if matchesPattern(finding, ignore.Findings) {
			continue
		}
		filtered = append(filtered, finding)
	}
	return filtered
}

func matchesPattern(finding Finding, patterns []FindingPattern) bool {
	for _, pattern := range patterns {
		if pattern.Path != "" && pattern.Path != finding.Path {
			continue
		}
		if pattern.Rule != "" && pattern.Rule != finding.Rule {
			continue
		}
		if pattern.Line > 0 && pattern.Line != finding.Line {
			continue
		}
		return true
	}
	return false
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func auditFile(root, path string) ([]Finding, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	workflowConcurrency := false
	workflowPermissions := false
	inJobs := false
	jobsLine := 0
	var currentJob *job
	jobs := []job{}
	findings := []Finding{}

	for index, raw := range lines {
		lineNumber := index + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if raw == "jobs:" {
			inJobs, jobsLine = true, lineNumber
			continue
		}
		if !inJobs {
			if raw == "concurrency:" || strings.HasPrefix(raw, "concurrency:") {
				workflowConcurrency = true
			}
			if raw == "permissions:" || strings.HasPrefix(raw, "permissions:") {
				workflowPermissions = true
			}
		}
		if strings.Contains(trimmed, "permissions: write-all") {
			findings = append(findings, Finding{"CF004", relativePath, lineNumber, "avoid permissions: write-all; grant only the scopes this job needs"})
		}
		if inJobs {
			if match := jobHeader.FindStringSubmatch(raw); match != nil && match[1] != "permissions" && match[1] != "concurrency" {
				jobs = append(jobs, job{name: match[1], line: lineNumber})
				currentJob = &jobs[len(jobs)-1]
				continue
			}
			if currentJob != nil && strings.HasPrefix(trimmed, "timeout-minutes:") {
				currentJob.hasTimeout = true
			}
		}
	}
	if !workflowConcurrency {
		findings = append(findings, Finding{"CF001", relativePath, 1, "add workflow concurrency with cancel-in-progress to stop duplicate runs consuming CI minutes"})
	}
	if !workflowPermissions {
		findings = append(findings, Finding{"CF002", relativePath, 1, "declare default read-only GITHUB_TOKEN permissions, then elevate only specific jobs"})
	}
	for _, candidate := range jobs {
		if !candidate.hasTimeout {
			findings = append(findings, Finding{"CF003", relativePath, candidate.line, fmt.Sprintf("job %q needs timeout-minutes to cap a stalled run", candidate.name)})
		}
	}
	if inJobs && len(jobs) == 0 {
		findings = append(findings, Finding{"CF005", relativePath, jobsLine, "no jobs found under jobs:"})
	}
	return findings, nil
}

type job struct {
	name       string
	line       int
	hasTimeout bool
}
