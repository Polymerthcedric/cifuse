package audit

import (
	"encoding/json"
	"sort"
)

var ruleDocs = map[string]struct {
	Name        string
	Description string
}{
	"CF001": {"missing-concurrency", "Workflow is missing a concurrency group with cancel-in-progress, so duplicate runs can consume CI minutes."},
	"CF002": {"undeclared-default-permissions", "Workflow does not declare default read-only GITHUB_TOKEN permissions before elevating specific jobs."},
	"CF003": {"missing-job-timeout", "A job has no timeout-minutes cap, so a stalled run can hang indefinitely."},
	"CF004": {"write-all-permissions", "permissions: write-all grants a broader token than the job needs."},
	"CF005": {"empty-jobs", "The jobs: key contains no jobs, so the workflow does nothing."},
}

type sarifDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Organization   string      `json:"organization"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ShortDescription sarifText `json:"shortDescription"`
	FullDescription  sarifText `json:"fullDescription"`
	HelpURI          string    `json:"helpUri"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func (r Report) SARIF(version string) ([]byte, error) {
	rules := sortedRuleIDs(r.Findings)
	ruleDefs := make([]sarifRule, 0, len(rules))
	for _, id := range rules {
		doc := ruleDocs[id]
		ruleDefs = append(ruleDefs, sarifRule{
			ID:               id,
			Name:             doc.Name,
			ShortDescription: sarifText{Text: doc.Description},
			FullDescription:  sarifText{Text: doc.Description},
			HelpURI:          "https://github.com/Polymerthcedric/cifuse#rules",
		})
	}

	results := make([]sarifResult, 0, len(r.Findings))
	for _, finding := range r.Findings {
		results = append(results, sarifResult{
			RuleID:  finding.Rule,
			Level:   "error",
			Message: sarifText{Text: finding.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: finding.Path},
					Region:           sarifRegion{StartLine: finding.Line},
				},
			}},
		})
	}

	doc := sarifDocument{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "cifuse",
				Organization:   "Polymerthcedric",
				Version:        version,
				InformationURI: "https://github.com/Polymerthcedric/cifuse",
				Rules:          ruleDefs,
			}},
			Results: results,
		}},
	}
	return json.MarshalIndent(doc, "", "  ")
}

func sortedRuleIDs(findings []Finding) []string {
	seen := map[string]bool{}
	for _, finding := range findings {
		seen[finding.Rule] = true
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
