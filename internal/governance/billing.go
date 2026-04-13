package governance

// UsageRecord is one billable unit (agent-hour, egress GB, etc.).
type UsageRecord struct {
	TenantID string
	Metric   string
	Units    float64
	UnitCost float64
}

// AllocateCost splits shared spend by usage share.
func AllocateCost(total float64, usages []UsageRecord) map[string]float64 {
	var sum float64
	for _, u := range usages {
		sum += u.Units
	}
	out := make(map[string]float64)
	if sum == 0 {
		return out
	}
	for _, u := range usages {
		share := u.Units / sum
		out[u.TenantID] += total * share
	}
	return out
}
