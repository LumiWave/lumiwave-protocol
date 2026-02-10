package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintRootUsage(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	printRootUsage()

	_ = w.Close()
	outBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("stdout read failed: %v", err)
	}
	out := string(outBytes)

	checks := []string{
		"LumiWave Go Client Sample",
		"wallet subcommands",
		"balance subcommands",
		"tx subcommands",
		"send-privkey",
		"chain subcommands",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Fatalf("usage output missing %q\noutput=%s", c, out)
		}
	}
}
