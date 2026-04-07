package containerd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	contentapi "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageManager handles image operations for containerd runtime.
type ImageManager struct {
	client *containerd.Client
}

func NewImageManager(client *containerd.Client) *ImageManager {
	return &ImageManager{client: client}
}

func (m *ImageManager) PullImage(ctx context.Context, imageRef string, opts ...containerd.RemoteOpt) (containerd.Image, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference cannot be empty")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	if img, err := m.GetImage(ctx, imageRef); err == nil && img != nil {
		return img, nil
	}

	defaultOpts := []containerd.RemoteOpt{
		containerd.WithPullUnpack,
		containerd.WithPullLabels(map[string]string{
			"source":    "agentos",
			"pulled_at": time.Now().Format(time.RFC3339),
		}),
	}
	allOpts := append(defaultOpts, opts...)

	image, err := m.client.Pull(ctx, imageRef, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %s: %w", imageRef, err)
	}
	return image, nil
}

func (m *ImageManager) GetImage(ctx context.Context, imageRef string) (containerd.Image, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference cannot be empty")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	image, err := m.client.GetImage(ctx, imageRef)
	if err != nil {
		if dgst, parseErr := digest.Parse(imageRef); parseErr == nil {
			image, err = m.client.GetImage(ctx, dgst.String())
			if err != nil {
				return nil, err
			}
			return image, nil
		}
		return nil, err
	}
	return image, nil
}

func (m *ImageManager) ListImages(ctx context.Context, filters ...string) ([]containerd.Image, error) {
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)
	imageList, err := m.client.ListImages(ctx, filters...)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return imageList, nil
}

func (m *ImageManager) RemoveImage(ctx context.Context, imageRef string) error {
	if imageRef == "" {
		return fmt.Errorf("image reference cannot be empty")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	image, err := m.GetImage(ctx, imageRef)
	if err != nil {
		return nil
	}
	if err := m.client.ImageService().Delete(ctx, image.Name()); err != nil {
		return fmt.Errorf("failed to remove image %s: %w", imageRef, err)
	}
	return nil
}

func (m *ImageManager) ImageInfo(ctx context.Context, imageRef string) (*ImageInfo, error) {
	image, err := m.GetImage(ctx, imageRef)
	if err != nil {
		return nil, err
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	configDesc, err := image.Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	manifest, err := images.Manifest(ctx, m.client.ContentStore(), image.Target(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get image manifest: %w", err)
	}
	_ = manifest

	p, err := readBlob(ctx, m.client.ContentStore(), configDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to read image config: %w", err)
	}

	var imgConfig ocispec.Image
	if err := json.Unmarshal(p, &imgConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image config: %w", err)
	}

	return &ImageInfo{
		Name:       image.Name(),
		Digest:     image.Target().Digest.String(),
		Size:       image.Target().Size,
		Arch:       imgConfig.Architecture,
		OS:         imgConfig.OS,
		Author:     imgConfig.Author,
		Entrypoint: imgConfig.Config.Entrypoint,
		Cmd:        imgConfig.Config.Cmd,
		WorkingDir: imgConfig.Config.WorkingDir,
		Env:        imgConfig.Config.Env,
	}, nil
}

func (m *ImageManager) PushImage(ctx context.Context, imageRef string, opts ...containerd.RemoteOpt) error {
	if imageRef == "" {
		return fmt.Errorf("image reference cannot be empty")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	image, err := m.GetImage(ctx, imageRef)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	if err := m.client.Push(ctx, imageRef, image.Target(), opts...); err != nil {
		return fmt.Errorf("failed to push image %s: %w", imageRef, err)
	}
	return nil
}

// ImageInfo contains metadata about a container image.
type ImageInfo struct {
	Name       string   `json:"name"`
	Digest     string   `json:"digest"`
	Size       int64    `json:"size"`
	Arch       string   `json:"architecture"`
	OS         string   `json:"os"`
	Author     string   `json:"author,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	Cmd        []string `json:"cmd,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
	Env        []string `json:"env,omitempty"`
}

func readBlob(ctx context.Context, store contentapi.Store, desc ocispec.Descriptor) ([]byte, error) {
	ra, err := store.ReaderAt(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer ra.Close()
	p := make([]byte, ra.Size())
	_, err = ra.ReadAt(p, 0)
	return p, err
}
