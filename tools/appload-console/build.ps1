# Build the GMS Console AppLoad app on Windows.
# Produces build\gms-console\ ready to copy to the tablet.
#
# Requires:
#   - Go (for the backend). If not installed: winget install GoLang.Go
#   - Qt6 'rcc' to pack the QML into resources.rcc. Options:
#       * Windows Qt install: set $env:RCC to the full path of rcc.exe, OR
#       * WSL/Linux: run build.sh instead (apt install qt6-base-dev-tools)
#
# Usage:  powershell -ExecutionPolicy Bypass -File .\build.ps1

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
$dest = Join-Path $root "build\gms-console"

Remove-Item -Recurse -Force $dest -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $dest "backend") | Out-Null

# 1. Backend (reMarkable 2 = linux/arm/v7)
Write-Host "Building backend..."
Push-Location (Join-Path $root "backend")
$env:GOOS = "linux"; $env:GOARCH = "arm"; $env:GOARM = "7"; $env:CGO_ENABLED = "0"
go build -o (Join-Path $dest "backend\entry") .
Pop-Location

# 2. Frontend resources (resources.rcc)
$rcc = $env:RCC
if (-not $rcc) {
    $rcc = (Get-Command rcc -ErrorAction SilentlyContinue).Source
}
if (-not $rcc) {
    Write-Warning "rcc not found. Set `$env:RCC to your Qt6 rcc.exe, or build resources.rcc via WSL:"
    Write-Warning "  wsl bash -c 'rcc --binary -o build/gms-console/resources.rcc application.qrc'"
} else {
    Write-Host "Packing resources with $rcc ..."
    & $rcc --binary -o (Join-Path $dest "resources.rcc") (Join-Path $root "application.qrc")
}

# 3. Manifest + icon
Copy-Item (Join-Path $root "manifest.json") $dest
if (Test-Path (Join-Path $root "icon.png")) { Copy-Item (Join-Path $root "icon.png") $dest }

Write-Host "Done -> $dest"
Write-Host "Copy it to the tablet: scp -r `"$dest`" root@10.11.99.1:/home/root/xovi/exthome/appload/"
