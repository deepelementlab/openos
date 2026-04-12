package packaging

import (
	"strings"

	"github.com/opencontainers/go-digest"
)

// ParseImageReference splits an OCI-style reference into name and digest if present.
// Example: myregistry/agent:v1.0@sha256:abc... -> name, digest form.
func ParseImageReference(ref string) (name string, dgst digest.Digest) {
	if idx := strings.LastIndex(ref, "@"); idx > 0 {
		d := digest.Digest(ref[idx+1:])
		if d.Validate() == nil {
			return ref[:idx], d
		}
	}
	return ref, ""
}
