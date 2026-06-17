# repair-config.ps1 - Refresh config.json real_bin after VS Code Claude Code extension update
# Run: powershell -ExecutionPolicy Bypass -File scripts/repair-config.ps1
#
# Use when VS Code shows "Claude Code process exited with code 1" after an extension update.
# This script scans for the latest extension version and updates real_bin. No rebuild needed.
# If config.json is corrupted, it will be rebuilt from scratch.

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

# Read existing config, tolerate corruption
$restored = $false
if (Test-Path $configFile) {
    try {
        $raw = Get-Content $configFile -Raw -Encoding UTF8
        if ($raw.Trim() -ne "") {
            $config = $raw | ConvertFrom-Json
            $oldBin = $config.real_bin
            Write-Host "  Old real_bin: $oldBin"
        } else {
            $restored = $true
        }
    } catch {
        Write-Warning "config.json is corrupted (ConvertFrom-Json failed), rebuilding from defaults."
        Write-Warning "Error: $($_.Exception.Message)"
        $restored = $true
    }
} else {
    $restored = $true
}

if ($restored) {
    Write-Host "  Creating new config from scratch."
}

# Build fresh config object (ConvertTo-Json handles escaping automatically)
$config = [ordered]@{
    real_bin            = $realClaude
    enable_json_parse   = $true
    parse_stderr        = $false
    drop_unknown_events = $true
    enable_ws           = $false
}

# Write using Set-Content -Encoding UTF8 (preserves Chinese characters correctly)
$json = $config | ConvertTo-Json -Depth 5
Set-Content -Path $configFile -Value $json -Encoding UTF8 -Force
Write-Host "  config.json written."

# Verify proxy can launch real Claude
Write-Host "Verifying proxy..."
$proxyExe = "$proxyRoot\bin\claude.exe"
if (-not (Test-Path $proxyExe)) {
    Write-Warning "proxy binary not found at $proxyExe - run install-proxy.ps1 to rebuild"
    exit 0
}

$versionOutput = cmd /c ""$proxyExe" --version 2>&1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAILED: proxy --version returned exit code $LASTEXITCODE"
    Write-Host "  Output: $versionOutput"
    Write-Host "  Try running install-proxy.ps1 for a full reinstall."
    exit 1
}
Write-Host "  OK: $($versionOutput -join ' ')"

Write-Host ""
Write-Host "Repair complete. Reload VS Code (Ctrl+Shift+P -> Reload Window)."
