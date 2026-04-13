package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RegistryHTTP identifies the HTTP remote backend.
const RegistryHTTP RegistryType = "http"

// HTTPRegistry is a minimal REST client for team-shared AAP storage.
type HTTPRegistry struct {
	BaseURL string
	Client  *http.Client
	Token   string // optional Bearer token
}

// NewHTTPRegistry trims trailing slash from baseURL (e.g. https://registry.example.com).
func NewHTTPRegistry(baseURL string) *HTTPRegistry {
	return &HTTPRegistry{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (r *HTTPRegistry) authHeader(req *http.Request) {
	if r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}
}

func (r *HTTPRegistry) urlPrefix(name, version string) string {
	return fmt.Sprintf("%s/v1/packages/%s/%s", r.BaseURL, safeName(name), safeName(version))
}

// Push uploads every file under pkgDir preserving relative paths (manifest.json, layers.json, sigs, …).
func (r *HTTPRegistry) Push(ctx context.Context, pkgDir, name, version string) error {
	if r.BaseURL == "" {
		return fmt.Errorf("http registry: empty base URL")
	}
	return filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(pkgDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := r.urlPrefix(name, version) + "/" + rel
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(data))
		if err != nil {
			return err
		}
		r.authHeader(req)
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := r.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("http registry: put %s: %s: %s", rel, resp.Status, string(b))
		}
		return nil
	})
}

// Pull downloads manifest.json and layers.json (and any signature files if present) into destDir.
func (r *HTTPRegistry) Pull(ctx context.Context, name, version, destDir string) error {
	if r.BaseURL == "" {
		return fmt.Errorf("http registry: empty base URL")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	prefix := r.urlPrefix(name, version)
	for _, rel := range []string{"manifest.json", "layers.json", "signature.json"} {
		if err := r.pullFile(ctx, prefix+"/"+rel, filepath.Join(destDir, filepath.FromSlash(rel))); err != nil {
			if rel == "signature.json" {
				continue // optional
			}
			return err
		}
	}
	return nil
}

func (r *HTTPRegistry) pullFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	r.authHeader(req)
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http registry: get %s: %s: %s", url, resp.Status, string(b))
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
