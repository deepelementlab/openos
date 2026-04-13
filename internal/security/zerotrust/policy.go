// Package zerotrust defines mTLS and identity expectations for AOS services.
package zerotrust

// ServiceIdentity is SPIFFE-like ID for workloads.
type ServiceIdentity struct {
	Cluster   string
	Namespace string
	Service   string
}

// String renders a URI for policy checks.
func (s ServiceIdentity) String() string {
	return "spiffe://" + s.Cluster + "/ns/" + s.Namespace + "/svc/" + s.Service
}

// TrustBundle holds roots for verifying peer certificates (placeholder).
type TrustBundle struct {
	PEM [][]byte
}
