# Key package coverage: either summarize existing profile (env COVERAGE_FILE) or run focused tests.
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

if ($env:COVERAGE_FILE) {
    $Out = $env:COVERAGE_FILE
    if (-not (Test-Path $Out)) { throw "coverage file not found: $Out" }
    Write-Host "=== Using existing $Out ==="
} else {
    $Out = if ($env:COVERAGE_KEY_OUT) { $env:COVERAGE_KEY_OUT } else { "coverage.key.out" }
    $pkgs = @("./cmd/aos/...", "./internal/builder/...", "./pkg/runtime/facade/...")
    go test @pkgs -coverprofile=$Out -covermode=atomic
    Write-Host "=== Total for selected packages ==="
    go tool cover -func=$Out | Select-Object -Last 1
}

Write-Host "=== Key packages (line coverage by file) ==="
go tool cover -func=$Out | Select-String -Pattern "github.com/agentos/aos/cmd/aos/|github.com/agentos/aos/internal/builder/|github.com/agentos/aos/pkg/runtime/facade/"
