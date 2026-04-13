package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const minimalAgentfile = `{
  "apiVersion": "aos.io/v1",
  "kind": "AgentPackage",
  "metadata": {"name": "cli_demo", "version": "0.1.0"},
  "steps": [{"type": "run", "command": "echo ok", "cache": true}]
}`

func newTestRoot(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	t.Setenv("AOS_NO_BANNER", "1")
	var out, errBuf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&errBuf)
	return root, &out, &errBuf
}

func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestCLI_BuildPushPullRunDryRun_LocalRegistry(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	agentPath := filepath.Join(dir, "Agentfile.json")
	require.NoError(t, os.WriteFile(agentPath, []byte(minimalAgentfile), 0o644))
	aapOut := filepath.Join(dir, "aap-out")
	regRoot := filepath.Join(dir, "registry")

	root, out, _ := newTestRoot(t)
	root.SetArgs([]string{"build", "-f", agentPath, "-o", aapOut})
	require.NoError(t, root.ExecuteContext(context.Background()))
	require.FileExists(t, filepath.Join(aapOut, "manifest.json"))
	require.FileExists(t, filepath.Join(aapOut, "layers.json"))
	require.Contains(t, out.String(), "Built AAP")

	out.Reset()
	errBuf := &bytes.Buffer{}
	root = newRootCmd()
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"push",
		"--registry", regRoot,
		"--name", "cli_demo", "--version", "0.1.0",
		"--package-dir", aapOut,
	})
	require.NoError(t, root.ExecuteContext(context.Background()))

	out.Reset()
	errBuf = &bytes.Buffer{}
	root = newRootCmd()
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"pull",
		"--registry", regRoot,
		"--name", "cli_demo", "--version", "0.1.0",
	})
	require.NoError(t, root.ExecuteContext(context.Background()))
	pulled := strings.TrimSpace(out.String())
	require.DirExists(t, pulled)

	out.Reset()
	errBuf = &bytes.Buffer{}
	root = newRootCmd()
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"run", "cli_demo:0.1.0",
		"--registry", regRoot,
		"--dry-run",
	})
	require.NoError(t, root.ExecuteContext(context.Background()))
	require.Contains(t, out.String(), "dry-run")
	require.Contains(t, out.String(), "cli_demo")
}

func TestCLI_PushPullHTTPRegistry(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	agentPath := filepath.Join(dir, "Agentfile.json")
	require.NoError(t, os.WriteFile(agentPath, []byte(minimalAgentfile), 0o644))
	aapOut := filepath.Join(dir, "aap-http")

	destRoot := t.TempDir()
	var sawAuth bool
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		local := filepath.Join(destRoot, filepath.ToSlash(r.URL.Path))
		switch r.Method {
		case http.MethodPut:
			require.NoError(t, os.MkdirAll(filepath.Dir(local), 0o755))
			b, _ := io.ReadAll(r.Body)
			require.NoError(t, os.WriteFile(local, b, 0o644))
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			b, err := os.ReadFile(local)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(b)
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	root, out, _ := newTestRoot(t)
	root.SetArgs([]string{"build", "-f", agentPath, "-o", aapOut})
	require.NoError(t, root.ExecuteContext(context.Background()))

	out.Reset()
	errBuf := &bytes.Buffer{}
	root = newRootCmd()
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"push",
		"--registry-url", srv.URL,
		"--registry-token", "test-token",
		"--name", "http_demo", "--version", "1.0.0",
		"--package-dir", aapOut,
	})
	require.NoError(t, root.ExecuteContext(context.Background()))
	require.True(t, sawAuth, "expected Authorization header on HTTP push")

	pullDest := filepath.Join(dir, "pulled")
	require.NoError(t, os.MkdirAll(pullDest, 0o755))
	out.Reset()
	errBuf = &bytes.Buffer{}
	root = newRootCmd()
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"pull",
		"--registry-url", srv.URL,
		"--registry-token", "test-token",
		"--name", "http_demo", "--version", "1.0.0",
		"--dest", pullDest,
	})
	require.NoError(t, root.ExecuteContext(context.Background()))
	require.FileExists(t, filepath.Join(pullDest, "manifest.json"))
}
