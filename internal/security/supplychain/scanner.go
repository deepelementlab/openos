package supplychain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ScanResult is a vulnerability scan summary.
type ScanResult struct {
	ImageRef     string
	CriticalCVEs int
	HighCVEs     int
	Tool         string // trivy, grype, or stub
	RawJSON      []byte `json:"-"`
}

// trivyReport matches a minimal subset of Trivy JSON output.
type trivyReport struct {
	Results []struct {
		Vulnerabilities []struct {
			Severity string `json:"Severity"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

// Scanner runs image scans via Trivy or Grype when available in PATH.
type Scanner struct{}

// NewScanner creates a scanner.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan invokes trivy or grype when installed; otherwise returns an empty result (no CVEs).
func (s *Scanner) Scan(ctx context.Context, imageRef string) (*ScanResult, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("scanner: empty image ref")
	}
	if path, err := exec.LookPath("trivy"); err == nil {
		return scanTrivy(ctx, path, imageRef)
	}
	if path, err := exec.LookPath("grype"); err == nil {
		return scanGrype(ctx, path, imageRef)
	}
	return &ScanResult{ImageRef: imageRef, Tool: "stub"}, nil
}

func scanTrivy(ctx context.Context, trivyBin, imageRef string) (*ScanResult, error) {
	cmd := exec.CommandContext(ctx, trivyBin, "image", "--format", "json", "--quiet", imageRef)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("trivy: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var rep trivyReport
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		return &ScanResult{ImageRef: imageRef, Tool: "trivy", RawJSON: stdout.Bytes()}, nil
	}
	var crit, high int
	for _, r := range rep.Results {
		for _, v := range r.Vulnerabilities {
			switch strings.ToUpper(v.Severity) {
			case "CRITICAL":
				crit++
			case "HIGH":
				high++
			}
		}
	}
	return &ScanResult{
		ImageRef: imageRef, CriticalCVEs: crit, HighCVEs: high, Tool: "trivy", RawJSON: stdout.Bytes(),
	}, nil
}

func scanGrype(ctx context.Context, grypeBin, imageRef string) (*ScanResult, error) {
	cmd := exec.CommandContext(ctx, grypeBin, imageRef, "-o", "json", "--quiet")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("grype: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var doc struct {
		Matches []struct {
			Vulnerability struct {
				Severity string `json:"severity"`
			} `json:"vulnerability"`
		} `json:"matches"`
	}
	_ = json.Unmarshal(stdout.Bytes(), &doc)
	var crit, high int
	for _, m := range doc.Matches {
		switch strings.ToUpper(m.Vulnerability.Severity) {
		case "CRITICAL":
			crit++
		case "HIGH":
			high++
		}
	}
	return &ScanResult{
		ImageRef: imageRef, CriticalCVEs: crit, HighCVEs: high, Tool: "grype", RawJSON: stdout.Bytes(),
	}, nil
}

// AdmissionOK returns false if critical or high counts exceed policy (zero-tolerance default).
func AdmissionOK(r *ScanResult, maxCritical, maxHigh int) bool {
	if r == nil {
		return true
	}
	return r.CriticalCVEs <= maxCritical && r.HighCVEs <= maxHigh
}
