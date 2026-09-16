package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Polymerthcedric/cifuse/internal/audit"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "audit" {
		fmt.Fprintln(os.Stderr, "usage: cifuse audit [--format text|json] [directory]")
		os.Exit(2)
	}

	flags := flag.NewFlagSet("audit", flag.ExitOnError)
	format := flags.String("format", "text", "output format: text or json")
	flags.Parse(os.Args[2:])
	path := "."
	if flags.NArg() > 0 {
		path = flags.Arg(0)
	}

	report, err := audit.Run(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cifuse:", err)
		os.Exit(2)
	}
	if *format == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, "cifuse:", err)
			os.Exit(2)
		}
	} else if *format == "text" {
		fmt.Print(report.Text())
	} else {
		fmt.Fprintln(os.Stderr, "cifuse: --format must be text or json")
		os.Exit(2)
	}
	if report.HasFailures() {
		os.Exit(1)
	}
}
