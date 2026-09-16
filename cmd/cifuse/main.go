package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Polymerthcedric/cifuse/internal/audit"
	"github.com/Polymerthcedric/cifuse/internal/config"
	"github.com/Polymerthcedric/cifuse/internal/initwf"
)

var version = "v0.2.0"

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		return
	}
	switch os.Args[1] {
	case "audit":
		os.Exit(runAudit(os.Args[2:]))
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "version":
		fmt.Fprintln(os.Stdout, "cifuse", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `cifuse audits GitHub Actions workflows for guardrail gaps.

Usage:
  cifuse audit [flags] [directory]
  cifuse init [flags] [directory]
  cifuse version

Commands:
  audit    scan .github/workflows for guardrail gaps
  init     scaffold a self-checking cifuse workflow
  version  print the cifuse version

Audit flags:
  --format text|json|sarif   output format (default text)
  --config PATH              config file to load (default .cifuserc.yml)
  --exclude RULE             ignore a rule ID such as CF001 (repeatable)
  --exclude-path PATH        ignore a workflow path (repeatable)

Init flags:
  --force                    overwrite an existing cifuse workflow`)
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func runAudit(args []string) int {
	flags := flag.NewFlagSet("audit", flag.ExitOnError)
	format := flags.String("format", "text", "output format: text, json or sarif")
	configPath := flags.String("config", "", "path to a cifuse config file (default: .cifuserc.yml)")
	var excludedRules stringList
	var excludedPaths stringList
	flags.Var(&excludedRules, "exclude", "rule ID to ignore, e.g. CF001 (repeatable)")
	flags.Var(&excludedPaths, "exclude-path", "workflow path to ignore (repeatable)")
	flags.Parse(args)

	root := "."
	if flags.NArg() > 0 {
		root = flags.Arg(0)
	}

	ignore := audit.Ignore{}
	defaultConfig := filepath.Join(root, config.Filename)
	if isRegularFile(defaultConfig) {
		loaded, err := config.Load(defaultConfig)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cifuse:", err)
			return 2
		}
		ignore = mergeIgnore(ignore, loaded)
	}
	if *configPath != "" {
		loaded, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cifuse:", err)
			return 2
		}
		ignore = mergeIgnore(ignore, loaded)
	}
	ignore.Rules = append(ignore.Rules, excludedRules...)
	ignore.Paths = append(ignore.Paths, excludedPaths...)

	report, err := audit.Run(root, ignore)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cifuse:", err)
		return 2
	}
	switch *format {
	case "text":
		fmt.Print(report.Text())
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, "cifuse:", err)
			return 2
		}
	case "sarif":
		out, err := report.SARIF(version)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cifuse:", err)
			return 2
		}
		fmt.Println(string(out))
	default:
		fmt.Fprintln(os.Stderr, "cifuse: --format must be text, json or sarif")
		return 2
	}
	if report.HasFailures() {
		return 1
	}
	return 0
}

func runInit(args []string) int {
	flags := flag.NewFlagSet("init", flag.ExitOnError)
	force := flags.Bool("force", false, "overwrite an existing cifuse workflow")
	flags.Parse(args)

	root := "."
	if flags.NArg() > 0 {
		root = flags.Arg(0)
	}
	created, err := initwf.Generate(root, *force)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cifuse:", err)
		return 1
	}
	for _, path := range created {
		fmt.Fprintf(os.Stdout, "created %s\n", path)
	}
	return 0
}

func mergeIgnore(primary, extra audit.Ignore) audit.Ignore {
	return audit.Ignore{
		Rules:    append(append([]string{}, primary.Rules...), extra.Rules...),
		Paths:    append(append([]string{}, primary.Paths...), extra.Paths...),
		Findings: append(append([]audit.FindingPattern{}, primary.Findings...), extra.Findings...),
	}
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
