//go:build e2e

package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLI_Smoke_RealCustomersCSV(t *testing.T) {
	// Adjust this path to where your real CSV lives relative to the repo root.
	// for differnt file wantSnippets need to be fixed.
	csvPath := filepath.Clean("./customers.csv")
	if _, err := os.Stat(csvPath); err != nil {
		t.Skipf("customers.csv not found at %q (skipping optional e2e smoke)", csvPath)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.csv")

	cmd := exec.Command("go", "run", ".", "-path", csvPath, "-out", out)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout // should be empty when -out is used
	cmd.Stderr = &stderr // slog writes here

	if err := cmd.Run(); err != nil {
		t.Fatalf("go run failed: %v\nstderr:\n%s", err, stderr.String())
	}

	// Basic sanity checks (keep it un-brittle).
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected output file at %q: %v", out, err)
	}
	if fi, _ := os.Stat(out); fi.Size() == 0 {
		t.Fatalf("expected non-empty output file at %q", out)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("summary")) {
		t.Errorf("expected summary log in stderr; got:\n%s", stderr.String())
	}

	//This is strictly works only for the  "./customers.csv"
	// for differnt file the values need to be adjusted.
	wantSnippets := [][]byte{
		[]byte("total_rows=3004"),
		[]byte("bad_rows=2"),
		[]byte("unique_domains=501"),
	}
	for _, s := range wantSnippets {
		if !bytes.Contains(stderr.Bytes(), s) {
			t.Fatalf("expected summary to contain %q; got:\n%s", s, stderr.String())
		}
	}
}
