package grpc

// CommandResult is the outcome of executing a command in an agent (non-protobuf helper).
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}
