package registry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPRegistry_PushPull(t *testing.T) {
	destRoot := t.TempDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		local := filepath.Join(destRoot, filepath.FromSlash(r.URL.Path))
		switch r.Method {
		case http.MethodPut:
			_ = os.MkdirAll(filepath.Dir(local), 0o755)
			b, _ := io.ReadAll(r.Body)
			_ = os.WriteFile(local, b, 0o644)
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

	pkg := filepath.Join(t.TempDir(), "bundle")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "manifest.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "layers.json"), []byte(`{"layer_digests":[]}`), 0o644))

	hr := NewHTTPRegistry(srv.URL)
	require.NoError(t, hr.Push(t.Context(), pkg, "my/app", "v1"))
	out := t.TempDir()
	require.NoError(t, hr.Pull(t.Context(), "my/app", "v1", out))
	b, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	require.NoError(t, err)
	require.Equal(t, "{}", string(b))
}

func TestHTTPRegistry_PushNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			http.Error(w, "fail", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	pkg := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "f.txt"), []byte("x"), 0o644))
	hr := NewHTTPRegistry(srv.URL)
	err := hr.Push(context.Background(), pkg, "n", "v")
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestHTTPRegistry_PullMissingManifest(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	hr := NewHTTPRegistry(srv.URL)
	err := hr.Pull(context.Background(), "n", "v", t.TempDir())
	require.Error(t, err)
}

func TestHTTPRegistry_PushSendsBearer(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	pkg := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "a"), []byte("1"), 0o644))
	hr := NewHTTPRegistry(srv.URL)
	hr.Token = "secret"
	require.NoError(t, hr.Push(context.Background(), pkg, "n", "v"))
	require.Equal(t, "Bearer secret", auth)
}
