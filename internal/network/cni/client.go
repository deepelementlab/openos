package cni

import (
	"context"
	"encoding/json"
	"os/exec"
)

// Client invokes the CNI plugin binary with ADD/DEL.
type Client struct {
	PluginPath string
}

// AddResult is a minimal parsed result (IPs etc. filled when integrated).
type AddResult struct {
	RawJSON []byte
}

// Add runs CNI ADD with the given netns path and config.
func (c *Client) Add(ctx context.Context, netns, configJSON string) (*AddResult, error) {
	if c.PluginPath == "" {
		return &AddResult{RawJSON: []byte(`{}`)}, nil
	}
	cmd := exec.CommandContext(ctx, c.PluginPath, "add")
	cmd.Env = append(cmd.Environ(), "CNI_COMMAND=ADD", "CNI_NETNS="+netns)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return &AddResult{RawJSON: out}, nil
}

// Del runs CNI DEL.
func (c *Client) Del(ctx context.Context, netns, configJSON string) error {
	if c.PluginPath == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, c.PluginPath, "del")
	cmd.Env = append(cmd.Environ(), "CNI_COMMAND=DEL", "CNI_NETNS="+netns)
	return cmd.Run()
}

// ParseConfig unmarshals CNI network config JSON.
func ParseConfig(data []byte) (*NetworkConfig, error) {
	var nc NetworkConfig
	if err := json.Unmarshal(data, &nc); err != nil {
		return nil, err
	}
	return &nc, nil
}
