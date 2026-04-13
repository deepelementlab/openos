# Hot reload helper: re-run tests on save using watchexec or entr (install separately)
param(
  [string]$Path = "../internal"
)
Write-Host "Install watchexec, then: watchexec -w $Path go test ./..."
Write-Host "Or use Tilt + live_update for containerized dev."
