package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd("1.2.3")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "1.2.3" {
		t.Fatalf("version output = %q, want %q", got, "1.2.3")
	}
}

func TestProfilesList(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd("test")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"profiles", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "web") || !strings.Contains(got, "queue") {
		t.Fatalf("profiles list missing expected profiles: %q", got)
	}
}

func TestProfilesShow(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd("test")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"profiles", "show", "web"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "Services:") || !strings.Contains(got, "Failure modes:") {
		t.Fatalf("profiles show missing detail: %q", got)
	}
}

func TestValidateRejectsBadOutputFormat(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(reportPath, []byte(`{
	  "run_id": "sf_seed_1",
	  "services": ["api-gateway"],
	  "sample_trace_ids": ["abc123"]
	}`), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	buf := new(bytes.Buffer)
	cmd := NewRootCmd("test")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"validate", "tempo",
		"--report-file", reportPath,
		"--output", "xml",
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "output must be text or json") {
		t.Fatalf("execute err=%v output=%s", err, buf.String())
	}
}
