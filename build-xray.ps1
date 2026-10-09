# Build Xray-core v1.8.3 for macOS 10.13 (darwin/amd64) using Go 1.20.
#
# Why v1.8.3: it is the newest Xray-core tag whose go.mod still says "go 1.20"
# AND that contains REALITY. Building with Go 1.20 is what keeps the produced
# binary runnable on macOS 10.13 (High Sierra); Go 1.21+ drops 10.13.
#
# Result: resources\xray  (Mach-O x86_64, LC_BUILD_VERSION minOS=10.13)
#
# Usage: powershell -ExecutionPolicy Bypass -File .\build-xray.ps1

$ErrorActionPreference = 'Stop'
$Root = $PSScriptRoot
$Go   = 'C:\Users\stepa\go\bin\go1.20.exe'   # golang.org/dl/go1.20
$Ver  = 'v1.8.3'
$Work = Join-Path $env:TEMP 'vlessbar-xray'

if (-not (Test-Path $Go)) { throw "Go 1.20 not found at $Go (install: 'go install golang.org/dl/go1.20@latest; go1.20 download')" }

if (Test-Path $Work) { Remove-Item -Recurse -Force $Work }
Write-Host "cloning Xray-core $Ver ..."
git clone --depth 1 --branch $Ver https://github.com/XTLS/Xray-core.git $Work

New-Item -ItemType Directory -Force -Path (Join-Path $Root 'resources') | Out-Null
$env:GOOS = 'darwin'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
Write-Host "building ..."
Push-Location (Join-Path $Work 'main')
try {
	& $Go build -trimpath -ldflags "-s -w" -o (Join-Path $Root 'resources\xray') .
	if ($LASTEXITCODE -ne 0) { throw "xray build failed" }
} finally {
	Pop-Location
}
Write-Host "built resources\xray (Xray-core $Ver)"
