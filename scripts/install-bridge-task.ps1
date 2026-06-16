# install-bridge-task.ps1 — Install claude-bridge as Windows scheduled task
# Run: powershell -ExecutionPolicy Bypass -File scripts/install-bridge-task.ps1

$ErrorActionPreference = "Stop"
$taskName = "cc-connect-claude-bridge"
$bridgeExe = "$env:USERPROFILE\.cc-connect\claude-proxy\bin\claude-bridge.exe"
$bridgeDir = "$env:USERPROFILE\.cc-connect\claude-proxy"
$sessionKey = "feishu:oc_9837e218cd51ec1fa5f14ec230441973:ou_2f843da75285efd296cec87ed116c1b4"

Write-Host "=== Installing bridge as scheduled task ==="

# Remove existing task
schtasks /Delete /TN $taskName /F 2>$null | Out-Null

# Create task: runs at user logon, repeats every 5 minutes
$cmd = "cmd.exe /c `"set BRIDGE_SESSION=$sessionKey && set PATH=%USERPROFILE%\AppData\Roaming\npm;%PATH% && `"$bridgeExe`" >> `"$bridgeDir\logs\bridge.log`" 2>&1`""

schtasks /Create /TN $taskName /TR $cmd /SC ONLOGON /RL LIMITED /F

Write-Host "Task '$taskName' installed."
Write-Host "  Runs at logon, restarts if killed."
Write-Host ""
Write-Host "To remove: .\scripts\uninstall-bridge-task.ps1"
Write-Host "Check status: schtasks /Query /TN $taskName"
