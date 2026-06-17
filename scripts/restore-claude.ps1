# restore-claude.ps1 - Remove claude-proxy, restore original VS Code Claude
# Run: powershell -ExecutionPolicy Bypass -File scripts\restore-claude.ps1 -Force

param(
    [switch]$Force
)

$ErrorActionPreference = "Continue"

# --- Suicide protection: refuse to run inside VS Code / Claude Code ---
if ($env:VSCODE_PID -or $env:TERM_PROGRAM -eq "vscode") {
    Write-Host "=== REFUSED ==="
    Write-Host "This script stops Claude/claude-proxy processes."
    Write-Host "Do NOT run it from inside VS Code's integrated terminal or a Claude Code session --"
    Write-Host "it would kill the Claude process that is executing this script."
    Write-Host ""
    Write-Host "Please run from an external PowerShell window:"
    Write-Host "  powershell -ExecutionPolicy Bypass -File F:\Documents\Projects\claude-proxy\scripts\restore-claude.ps1 -Force"
    exit 2
}

$proxyRoot = "$env:USERPROFILE\.cc-connect\claude-proxy"

if (-not $Force) {
    Write-Host "=== restore-claude DRY-RUN ==="
    Write-Host "This is a preview. Run with -Force to actually execute."
    Write-Host ""
    Write-Host "Would stop these processes:"
    $procs = @()
    Get-Process claude -ErrorAction SilentlyContinue | ForEach-Object { $procs += "  claude (PID $($_.Id))" }
    Get-Process "claude-proxy" -ErrorAction SilentlyContinue | ForEach-Object { $procs += "  claude-proxy (PID $($_.Id))" }
    Get-Process "claude-bridge" -ErrorAction SilentlyContinue | ForEach-Object { $procs += "  claude-bridge (PID $($_.Id))" }
    Get-Process bridge -ErrorAction SilentlyContinue | ForEach-Object { $procs += "  bridge (PID $($_.Id))" }
    if ($procs.Count -eq 0) {
        Write-Host "  (none found)"
    } else {
        $procs | ForEach-Object { Write-Host $_ }
    }
    Write-Host ""
    Write-Host "Would NOT remove: $proxyRoot (events/logs preserved)"
    Write-Host "Would NOT touch: VS Code settings (claudeCode.claudeProcessWrapper)"
    Write-Host "Reminder: remove claudeCode.claudeProcessWrapper from VS Code settings.json"
    Write-Host ""
    Write-Host "To execute:"
    Write-Host "  powershell -ExecutionPolicy Bypass -File scripts\restore-claude.ps1 -Force"
    exit 0
}

# --- Real execution ---
Write-Host "=== claude-proxy restore ==="

Write-Host "[1/3] Stopping processes..."
Get-Process claude -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue 2>$null
Get-Process "claude-proxy" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue 2>$null
Get-Process "claude-bridge" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue 2>$null
Get-Process bridge -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue 2>$null

Write-Host "[2/3] Reminder: remove claudeCode.claudeProcessWrapper from VS Code settings.json"
Write-Host "  Ctrl+Shift+P -> Open User Settings (JSON)"
Write-Host '  Delete the line containing "claudeCode.claudeProcessWrapper"'

Write-Host "[3/3] Preserving logs and events at $proxyRoot"
Write-Host "  To fully remove, delete: $proxyRoot"
Write-Host ""
Write-Host "Restore complete. Reload VS Code to use original Claude."
