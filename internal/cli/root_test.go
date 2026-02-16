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
