# install-proxy.ps1 - Install claude-proxy as VS Code Claude wrapper
# Run: powershell -ExecutionPolicy Bypass -File scripts\install-proxy.ps1 -Force

param(
    [switch]$Force
)

$ErrorActionPreference = "Stop"

# --- Suicide protection: refuse to run inside VS Code / Claude Code ---
if ($env:VSCODE_PID -or $env:TERM_PROGRAM -eq "vscode") {
    Write-Host "=== REFUSED ==="
    Write-Host "This script replaces the active Claude entry point."
    Write-Host "Do NOT run it from inside VS Code's integrated terminal or a Claude Code session --"
    Write-Host "it may break the currently running Claude process."
    Write-Host ""
    Write-Host "Please run from an external PowerShell window:"
    Write-Host "  powershell -ExecutionPolicy Bypass -File F:\Documents\Projects\claude-proxy\scripts\install-proxy.ps1 -Force"
    exit 2
}

$proxyRoot = "$env:USERPROFILE\.cc-connect\claude-proxy"
$proxyBin = "$proxyRoot\bin"
$npmDir = "$env:APPDATA\npm"
$vsCodeExt = "$env:USERPROFILE\.vscode\extensions"

if (-not $Force) {
    Write-Host "=== install-proxy DRY-RUN ==="
    Write-Host "This is a preview. Run with -Force to actually build and install."
    Write-Host ""
    Write-Host "Would scan for latest VS Code Claude Code extension..."
    $extDirs = Get-ChildItem "$vsCodeExt\anthropic.claude-code-*-win32-x64" -Directory -ErrorAction SilentlyContinue
    if ($extDirs) {
        $sorted = $extDirs | Sort-Object {
            if ($_.Name -match 'anthropic\.claude-code-(\d+)\.(\d+)\.(\d+)-win32-x64') {
                [version]::new([int]$matches[1], [int]$matches[2], [int]$matches[3])
            } else { [version]"0.0.0" }
        }
        $latestExt = $sorted[-1]
        Write-Host "  Latest extension: $($latestExt.Name)"
        Write-Host "  real_bin would be: $($latestExt.FullName)\resources\native-binary\claude.exe"
    } else {
        Write-Host "  No extension found, would try npm global"
    }
    Write-Host ""
    Write-Host "Would build to: $proxyBin"
    Write-Host "  claude.exe (proxy)"
    Write-Host "  claude-bridge.exe"
    Write-Host "Would write config to: $proxyRoot\config.json"
    Write-Host "Would NOT modify VS Code settings"
    Write-Host ""
    Write-Host "To execute:"
    Write-Host "  powershell -ExecutionPolicy Bypass -File scripts\install-proxy.ps1 -Force"
    exit 0
}

# --- Real execution ---
Write-Host "=== claude-proxy installer ==="

# 1. Find latest real Claude binary
Write-Host "[1/5] Locating latest VS Code Claude Code extension..."
$extDirs = Get-ChildItem "$vsCodeExt\anthropic.claude-code-*-win32-x64" -Directory -ErrorAction SilentlyContinue
if ($extDirs) {
    $sorted = $extDirs | Sort-Object {
        if ($_.Name -match 'anthropic\.claude-code-(\d+)\.(\d+)\.(\d+)-win32-x64') {
            [version]::new([int]$matches[1], [int]$matches[2], [int]$matches[3])
        } else { [version]"0.0.0" }
    }
    $latestExt = $sorted[-1]
    $realClaude = "$($latestExt.FullName)\resources\native-binary\claude.exe"
    if (Test-Path $realClaude) {
        Write-Host "  Latest: $($latestExt.Name)"
        Write-Host "  real_bin: $realClaude"
    } else {
        Write-Host "  Latest extension found but claude.exe missing: $realClaude"
        $realClaude = $null
    }
}
if (-not $realClaude) {
    Write-Host "  VS Code extension not found, trying npm global..."
    $fallbacks = @(
        "$npmDir\claude.exe",
        "$npmDir\node_modules\@anthropic-ai\claude-code\bin\claude.exe"
    )
    foreach ($f in $fallbacks) {
        if (Test-Path $f) { $realClaude = $f; Write-Host "  Found: $realClaude"; break }
    }
}
if (-not $realClaude) {
    Write-Error "Cannot find real claude.exe. Install Claude Code first."
    exit 1
}

# 2. Create directories
Write-Host "[2/5] Creating directories..."
New-Item -ItemType Directory -Force $proxyBin | Out-Null
New-Item -ItemType Directory -Force "$proxyRoot\runtime" | Out-Null
New-Item -ItemType Directory -Force "$proxyRoot\logs" | Out-Null
New-Item -ItemType Directory -Force "$proxyRoot\events" | Out-Null

# 3. Build proxy
Write-Host "[3/5] Building claude-proxy..."
Push-Location "$PSScriptRoot\.."
$env:GOPROXY = "https://goproxy.cn,direct"
go build -o "$proxyBin\claude.exe" .\cmd\claude-proxy\
go build -o "$proxyBin\claude-bridge.exe" .\cmd\claude-bridge\
Pop-Location
Write-Host "  Built: $proxyBin\claude.exe"
Write-Host "  Built: $proxyBin\claude-bridge.exe"

# 4. Write config (ConvertTo-Json handles escaping; do NOT manually Replace backslashes)
Write-Host "[4/5] Writing config..."
$config = [ordered]@{
    real_bin            = $realClaude
    enable_json_parse   = $true
    parse_stderr        = $false
    drop_unknown_events = $true
    enable_ws           = $false
}
$json = $config | ConvertTo-Json -Depth 5
Set-Content -Path "$proxyRoot\config.json" -Value $json -Encoding UTF8 -Force
Write-Host "  Config: $proxyRoot\config.json"

# 5. Verify
Write-Host "[5/5] Verifying proxy..."
$versionOutput = cmd /c ""$proxyBin\claude.exe" --version 2>&1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAILED: proxy --version returned exit code $LASTEXITCODE"
    Write-Host "  Output: $versionOutput"
    Write-Host "  Rolling back: restoring original Claude..."
    & "$PSScriptRoot\restore-claude.ps1" -Force
    exit 1
}
Write-Host "  Proxy verified: $($versionOutput -join ' ')"

Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. In VS Code: Ctrl+Shift+P -> Open User Settings (JSON)"
Write-Host '     Add: "claudeCode.claudeProcessWrapper": "' + $proxyBin.Replace('\','\\') + '\\claude.exe"'
Write-Host "  2. Reload VS Code: Ctrl+Shift+P -> Reload Window"
Write-Host ""
Write-Host "Restore original:  .\scripts\restore-claude.ps1 -Force"
Write-Host "Repair after update: .\scripts\repair-config.ps1"