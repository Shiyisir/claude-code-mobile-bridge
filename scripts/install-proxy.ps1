# install-proxy.ps1 — Install claude-proxy as VS Code Claude wrapper
# Run: powershell -ExecutionPolicy Bypass -File scripts/install-proxy.ps1

$ErrorActionPreference = "Stop"
$proxyRoot = "$env:USERPROFILE\.cc-connect\claude-proxy"
$proxyBin = "$proxyRoot\bin"
$npmDir = "$env:APPDATA\npm"
$vsCodeExt = "$env:USERPROFILE\.vscode\extensions"

Write-Host "=== claude-proxy installer ==="

# 1. Find real Claude binary
Write-Host "[1/5] Locating real Claude binary..."
$realClaude = $null
$candidates = @(
    "$vsCodeExt\anthropic.claude-code-*\resources\native-binary\claude.exe",
    "$npmDir\claude.exe",
    "$npmDir\node_modules\@anthropic-ai\claude-code\bin\claude.exe"
)
foreach ($c in $candidates) {
    $found = Get-ChildItem $c -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($found) {
        $realClaude = $found.FullName
        Write-Host "  Found: $realClaude"
        break
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
Push-Location "F:\Documents\Projects\claude-proxy"
$env:GOPROXY = "https://goproxy.cn,direct"
go build -o "$proxyBin\claude.exe" .\cmd\claude-proxy\
go build -o "$proxyBin\claude-bridge.exe" .\cmd\claude-bridge\
Pop-Location
Write-Host "  Built: $proxyBin\claude.exe"
Write-Host "  Built: $proxyBin\claude-bridge.exe"

# 4. Write config
Write-Host "[4/5] Writing config..."
$config = @{
    real_bin = $realClaude.Replace('\', '\\')
    enable_ws = $false
    enable_json_parse = $true
    parse_stderr = $false
    drop_unknown_events = $true
} | ConvertTo-Json
[System.IO.File]::WriteAllText("$proxyRoot\config.json", $config)
Write-Host "  Config: $proxyRoot\config.json"

# 5. Instructions
Write-Host "[5/5] Done!"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. In VS Code: Ctrl+Shift+P -> Open User Settings (JSON)"
Write-Host "     Add: `"claudeCode.claudeProcessWrapper`": `"$($proxyBin.Replace('\','\\'))\\claude.exe`""
Write-Host "  2. Reload VS Code: Ctrl+Shift+P -> Reload Window"
Write-Host ""
Write-Host "Restore original: .\scripts\restore-claude.ps1"
