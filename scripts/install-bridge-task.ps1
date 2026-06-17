# install-bridge-task.ps1 — DEPRECATED since v0.1.2
# Bridge auto-start has been moved to the Windows Startup folder (start-cc.bat).
# Scheduled tasks are no longer used — no admin required, no battery policy issues.
# This script is kept for historical reference only; do NOT use in new installs.
# install-bridge-task.ps1 — Install claude-bridge as Windows scheduled task

$taskName = "cc-connect-claude-bridge"
$batchFile = "$env:USERPROFILE\.cc-connect\start-bridge.bat"

Write-Host "=== Installing bridge task ==="
Write-Host "Task: $taskName"
Write-Host "Run:  $batchFile"

# Create task
schtasks /Create /TN $taskName /TR $batchFile /SC ONLOGON /RL LIMITED /F

if ($LASTEXITCODE -eq 0) {
    Write-Host "Task installed successfully."
    Write-Host "Check: schtasks /Query /TN $taskName"
} else {
    Write-Host "Install failed. Trying with elevated prompt..."
}
