// exporter/exporter_test.go
package exporter

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveteshome/email-domain-counter/customerimporter"
)

func TestWriteCSV_Basics(t *testing.T) {
	type tc struct {
		name string
		data []customerimporter.DomainData
		want string
	}

	tests := []tc{
		{
			name: "Header_only",
			data: nil,
			want: "domain,number_of_customers\n",
		},
		{
			name: "Multiple_rows",
			data: []customerimporter.DomainData{
				{Domain: "a.com", CustomerQuantity: 3},
				{Domain: "b.com", CustomerQuantity: 1},
			},
			want: "domain,number_of_customers\n" +
				"a.com,3\n" +
				"b.com,1\n",
		},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		if err := WriteCSV(&buf, tt.data); err != nil {
			t.Fatalf("[%s] WriteCSV error: %v", tt.name, err)
		}
		if got := buf.String(); got != tt.want {
			t.Fatalf("[%s] CSV mismatch:\n--got--\n%q\n--want--\n%q", tt.name, got, tt.want)
		}
	}
}

func TestExportData_WritesToNestedDirs(t *testing.T) {
	type tc struct {
		name string
		data []customerimporter.DomainData
		want string
	}
	tests := []tc{
		{
			name: "Header_only",
			data: nil,
			want: "domain,number_of_customers\n",
		},
		{
			name: "With_rows",
			data: []customerimporter.DomainData{
				{Domain: "livejournal.com", CustomerQuantity: 12},
				{Domain: "microsoft.com", CustomerQuantity: 22},
			},
			want: "domain,number_of_customers\n" +
				"livejournal.com,12\n" +
				"microsoft.com,22\n",
		},
	}

	for _, tt := range tests {
		tmp := t.TempDir()
		out := filepath.Join(tmp, "deep", "nested", "out.csv")

		exp := NewCustomerExporter(out) // takes string path
		if err := exp.ExportData(tt.data); err != nil {
			t.Fatalf("[%s] ExportData error: %v", tt.name, err)
		}

		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("[%s] read out: %v", tt.name, err)
		}
		if string(b) != tt.want {
			t.Fatalf("[%s] file content mismatch:\n--got--\n%q\n--want--\n%q", tt.name, string(b), tt.want)
		}
	}
}

func TestExportData_EmptyPath_ReturnsError(t *testing.T) {
	exp := NewCustomerExporter("")
	err := exp.ExportData([]customerimporter.DomainData{})
	if err == nil {
		t.Fatalf("expected error for empty path, got nil")
	}
	_ = err
}

type failingWriter struct {
	n int
	i int
}

func (f *failingWriter) Write(p []byte) (int, error) {
	if f.i >= f.n {
		return 0, errors.New("boom")
	}
	f.i++
	return len(p), nil
}

func TestWriteCSV_PropagatesWriterErrors(t *testing.T) {
	data := []customerimporter.DomainData{
		{Domain: "x.com", CustomerQuantity: 1},
	}

	w := &failingWriter{n: 0}
	err := WriteCSV(w, data)
	if err == nil {
		t.Fatalf("expected non-nil error from WriteCSV, got nil")
	}
}

type shortWriter struct{ done bool }

func (s *shortWriter) Write(p []byte) (int, error) {
	if !s.done {
		s.done = true
		return len(p) / 2, io.ErrShortWrite
	}
	return 0, errors.New("boom")
}

func TestWriteCSV_ShortWriteIsError(t *testing.T) {
	data := []customerimporter.DomainData{{Domain: "x.com", CustomerQuantity: 1}}
	err := WriteCSV(&shortWriter{}, data)
	if err == nil {
		t.Fatalf("expected error from short write, got nil")
	}
}
