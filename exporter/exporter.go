package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/daveteshome/email-domain-counter/customerimporter"
)

var csvHeader = []string{"domain", "number_of_customers"}

type CustomerExporter struct {
	outPath string
}

func NewCustomerExporter(outPath string) *CustomerExporter {
	return &CustomerExporter{outPath: outPath}
}

func (e *CustomerExporter) ExportData(data []customerimporter.DomainData) error {
	if dir := filepath.Dir(e.outPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("ensure dir %q: %w", dir, err)
		}
	}

	f, err := os.Create(e.outPath)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", e.outPath, err)
	}
	defer f.Close()

	if err := WriteCSV(f, data); err != nil {
		return fmt.Errorf("write CSV to %q: %w", e.outPath, err)
	}
	return nil
}

func WriteCSV(w io.Writer, data []customerimporter.DomainData) error {
	cw := csv.NewWriter(w)

	if err := cw.Write(csvHeader); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, d := range data {
		if err := cw.Write([]string{d.Domain, strconv.Itoa(d.CustomerQuantity)}); err != nil {
			return fmt.Errorf("write row for %q: %w", d.Domain, err)
		}
	}

	cw.Flush()
	return cw.Error()
}
