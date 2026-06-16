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
