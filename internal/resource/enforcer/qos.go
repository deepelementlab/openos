package enforcer

// QoSClass mirrors Kubernetes-style classes for agent resource guarantees.
type QoSClass string

const (
	QoSGuaranteed QoSClass = "Guaranteed"
	QoSBurstable  QoSClass = "Burstable"
	QoSBestEffort QoSClass = "BestEffort"
)

// QoSSpec describes requests and limits used to classify QoS.
type QoSSpec struct {
	CPURequestNano int64
	CPULimitNano   int64
	MemoryRequest  int64
	MemoryLimit    int64
}

// Classify returns the QoS class from requests/limits semantics.
func Classify(s QoSSpec) QoSClass {
	if s.CPURequestNano == 0 && s.MemoryRequest == 0 {
		return QoSBestEffort
	}
	if s.CPULimitNano > 0 && s.MemoryLimit > 0 &&
		s.CPURequestNano == s.CPULimitNano && s.MemoryRequest == s.MemoryLimit {
		return QoSGuaranteed
	}
	return QoSBurstable
}
