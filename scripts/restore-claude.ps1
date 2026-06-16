# restore-claude.ps1 — Remove claude-proxy, restore original VS Code Claude
# Run: powershell -ExecutionPolicy Bypass -File scripts/restore-claude.ps1

$ErrorActionPreference = "Stop"
$proxyRoot = "$env:USERPROFILE\.cc-connect\claude-proxy"

Write-Host "=== claude-proxy restore ==="

# 1. Kill proxy processes
Write-Host "[1/3] Stopping processes..."
taskkill /F /IM claude-proxy.exe 2>$null | Out-Null
taskkill /F /IM claude-bridge.exe 2>$null | Out-Null
taskkill /F /IM bridge.exe 2>$null | Out-Null

# 2. Remove VS Code setting
Write-Host "[2/3] Reminder: remove claudeCode.claudeProcessWrapper from VS Code settings.json"
Write-Host "  Ctrl+Shift+P -> Open User Settings (JSON)"
Write-Host "  Delete the line containing `"claudeCode.claudeProcessWrapper`""

# 3. Optional: remove proxy binaries
Write-Host "[3/3] Preserving logs and events at $proxyRoot"
Write-Host "  To fully remove: Remove-Item -Recurse -Force `"$proxyRoot`""
Write-Host ""
Write-Host "Restore complete. Reload VS Code to use original Claude."
