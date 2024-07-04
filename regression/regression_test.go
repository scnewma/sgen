package regression

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

var binaryLocation = "../bin/sgen"

func TestMain(m *testing.M) {
	p, err := filepath.Abs(binaryLocation)
	if err != nil {
		panic(fmt.Sprintf("could not convert binary location to absolute path: %v", err))
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		panic(fmt.Sprintf("binary does not exist: %v", err))
	}

	binaryLocation = p
	os.Exit(m.Run())
}

func TestRegression(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		goldenFile string
	}{
		{
			name:       "file: no default template renders json",
			args:       []string{"names-no-default-file"},
			goldenFile: "default-template-json-file.golden",
		},
		{
			name:       "command: no default template renders json",
			args:       []string{"--sync", "names-no-default-command"},
			goldenFile: "default-template-json-command.golden",
		},
		{
			name:       "file: provide default template",
			args:       []string{"names-file"},
			goldenFile: "default-template-file.golden",
		},
		{
			name:       "command: provide default template",
			args:       []string{"--sync", "names-command"},
			goldenFile: "default-template-command.golden",
		},
		{
			name:       "file: named template",
			args:       []string{"names-file", "--template-name=bulleted"},
			goldenFile: "bulleted-template-file.golden",
		},
		{
			name:       "command: named template",
			args:       []string{"--sync", "names-command", "--template-name=bulleted"},
			goldenFile: "bulleted-template-command.golden",
		},
		{
			name:       "file: CLI template",
			args:       []string{"names-file", "--template={{.name | repeat 3}}"},
			goldenFile: "cli-template-file.golden",
		},
		{
			name:       "command: CLI template",
			args:       []string{"--sync", "names-command", "--template={{.name | repeat 3}}"},
			goldenFile: "cli-template-command.golden",
		},
	}

	configDir, err := filepath.Abs("./testdata/sgen")
	if err != nil {
		t.Fatalf("could not find ./testdata directory: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheDir := t.TempDir()

			cmd := exec.Command(binaryLocation, tt.args...)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "SGEN_CONFIG_DIR="+configDir)
			cmd.Env = append(cmd.Env, "SGEN_CACHE_DIR="+cacheDir)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("error running command: %v\nStdout:\n%s\nStderr:\n%s\n", err, stdout.String(), stderr.String())
			}

			assert.Assert(t, golden.String(stdout.String(), tt.goldenFile))
		})
	}
}
