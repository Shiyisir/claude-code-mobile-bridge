# repair-config.ps1 — Refresh config.json real_bin after VS Code Claude Code extension update
# Run: powershell -ExecutionPolicy Bypass -File scripts/repair-config.ps1
#
# Use when VS Code shows "Claude Code process exited with code 1" after an extension update.
# This script scans for the latest extension version and updates real_bin — no rebuild needed.

$ErrorActionPreference = "Stop"
$proxyRoot = "$env:USERPROFILE\.cc-connect\claude-proxy"
$vsCodeExt = "$env:USERPROFILE\.vscode\extensions"
$configFile = "$proxyRoot\config.json"

Write-Host "=== repair-config ==="
Write-Host "Scanning for latest Claude Code extension..."

# Scan for all installed versions, pick latest by semantic version
$extDirs = Get-ChildItem "$vsCodeExt\anthropic.claude-code-*-win32-x64" -Directory -ErrorAction SilentlyContinue

if (-not $extDirs) {
    Write-Error "No Claude Code extension found at: $vsCodeExt\anthropic.claude-code-*-win32-x64"
    Write-Host "Install Claude Code extension in VS Code first."
    exit 1
}

$sorted = $extDirs | Sort-Object {
    if ($_.Name -match 'anthropic\.claude-code-(\d+)\.(\d+)\.(\d+)-win32-x64') {
        [version]::new([int]$matches[1], [int]$matches[2], [int]$matches[3])
    } else {
        [version]"0.0.0"
    }
}
$latestExt = $sorted[-1]
$realClaude = "$($latestExt.FullName)\resources\native-binary\claude.exe"

if (-not (Test-Path $realClaude)) {
    Write-Error "claude.exe not found at expected path: $realClaude"
    exit 1
}

Write-Host "  Latest extension: $($latestExt.Name)"
Write-Host "  real_bin: $realClaude"

# Read existing config, update only real_bin
if (Test-Path $configFile) {
    $config = Get-Content $configFile -Raw | ConvertFrom-Json
    $oldBin = $config.real_bin
    $config.real_bin = $realClaude.Replace('\', '\\')
    $config | ConvertTo-Json | Set-Content $configFile -Encoding UTF8
    Write-Host "  Old: $oldBin"
    Write-Host "  New: $($config.real_bin)"
} else {
    # Config doesn't exist, create minimal config
    $config = @{
        real_bin = $realClaude.Replace('\', '\\')
        enable_ws = $false
        enable_json_parse = $true
        parse_stderr = $false
        drop_unknown_events = $true
    } | ConvertTo-Json
    [System.IO.File]::WriteAllText($configFile, $config)
    Write-Host "  Created new config: $configFile"
}
Write-Host "  Config updated."

# Verify
Write-Host "Verifying proxy..."
$proxyExe = "$proxyRoot\bin\claude.exe"
if (-not (Test-Path $proxyExe)) {
    Write-Warning "proxy binary not found at $proxyExe — run install-proxy.ps1 to rebuild"
    exit 0
}

# Use cmd /c to avoid PowerShell wrapping stderr as NativeCommandError
$versionOutput = cmd /c "$proxyExe --version 2>&1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAILED: proxy --version returned exit code $LASTEXITCODE"
    Write-Host "  Output: $versionOutput"
    Write-Host "  Try running install-proxy.ps1 for a full reinstall."
    exit 1
}
Write-Host "  OK: $($versionOutput -join ' ')"

Write-Host ""
Write-Host "Repair complete. Reload VS Code (Ctrl+Shift+P -> Reload Window)."
