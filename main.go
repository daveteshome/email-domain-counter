package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/daveteshome/email-domain-counter/customerimporter"
	"github.com/daveteshome/email-domain-counter/exporter"
)

const (
	exitOK    = 0
	exitFatal = 1
)

type Options struct {
	path                   string
	outFile                string
	emailHeader            string
	allowSingleLabelDomain bool
}

func readOptions() Options {
	var o Options

	flag.StringVar(&o.path, "path", "", "Path to the file with customer data (required)")
	flag.StringVar(&o.outFile, "out", "", "Optional: output file path (stdout if empty)")
	flag.StringVar(&o.emailHeader, "email-header", "email", "Email column header (case-insensitive)")
	flag.BoolVar(&o.allowSingleLabelDomain, "allow-single-label-domain", false, "Accept domains without a dot (e.g., user@corp)")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %s -path=<file> [-out=<file>] [-email-header=<name>] [--allow-single-label-domain]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
		//How to run hint:
		fmt.Fprintln(flag.CommandLine.Output(), `
			Examples:
			# Run and print to console.
			go run . -path "./customers.csv

			# Save to a file
			go run . -path "./customers.csv -out ./result.csv

			# Show help
			go run . -h
		`)
	}

	flag.Parse()
	return o
}

func main() {
	opts := readOptions()

	if opts.path == "" {
		slog.Error("input is required: pass -path=<file>.")
		flag.Usage()
		os.Exit(exitFatal)
	}

	// Fail early if the file does not exist or is a directory
	info, err := os.Stat(opts.path)
	if err != nil {
		slog.Error("cannot access input file", "path", opts.path, "error", err)
		os.Exit(exitFatal)
	}
	if info.IsDir() {
		slog.Error("input path is a directory, expected a file", "path", opts.path)
		os.Exit(exitFatal)
	}

	imp := customerimporter.New(customerimporter.Config{
		Path:                   opts.path,
		EmailHeader:            opts.emailHeader,
		AllowSingleLabelDomain: opts.allowSingleLabelDomain,
	})

	result, err := imp.ImportDomainData()
	if err != nil {
		slog.Error("failed to import", "error", err)
		os.Exit(exitFatal)
	}

	if opts.outFile == "" {
		if err := exporter.WriteCSV(os.Stdout, result.Data); err != nil {
			slog.Error("failed writing to stdout", "error", err)
			os.Exit(exitFatal)
		}
	} else {
		exp := exporter.NewCustomerExporter(opts.outFile)
		if err := exp.ExportData(result.Data); err != nil {
			slog.Error("failed writing file", "out", opts.outFile, "error", err)
			os.Exit(exitFatal)
		}
	}

	slog.Info("summary",
		"file", opts.path,
		"total_rows", result.Stats.TotalRows,
		"bad_rows", result.Stats.BadRows,
		"unique_domains", result.Stats.UniqueDomains,
		"sorted", "count desc, domain asc",
		"single_label_allowed", opts.allowSingleLabelDomain,
	)

	os.Exit(exitOK)
}
