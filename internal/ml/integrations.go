package ml

// MLflowRunRef ties an AOS agent to an MLflow experiment.
type MLflowRunRef struct {
	Experiment string
	RunID      string
}

// KubeflowPipelineRef ties workload to a KFP pipeline name/version.
type KubeflowPipelineRef struct {
	Name    string
	Version string
}
