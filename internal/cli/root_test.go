package cli

import (
	"bytes"
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
