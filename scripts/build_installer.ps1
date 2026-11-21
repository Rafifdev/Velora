$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Stop-Process -Name "Velora" -Force -ErrorAction SilentlyContinue

Write-Host "Building Velora.exe..."
wails build -clean
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$assetsDir = Join-Path $root "cmd\installer\assets"
New-Item -ItemType Directory -Force -Path $assetsDir | Out-Null
Copy-Item -Force (Join-Path $root "build\bin\Velora.exe") (Join-Path $assetsDir "Velora.exe")
Copy-Item -Force (Join-Path $root "build\windows\icon.ico") (Join-Path $assetsDir "icon.ico")

Write-Host "Embedding Windows icon & manifest resource..."
go run github.com/akavel/rsrc@latest -ico (Join-Path $root "build\windows\icon.ico") -manifest (Join-Path $root "cmd\installer\installer.manifest") -o (Join-Path $root "cmd\installer\rsrc_windows_amd64.syso")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

try {
    Write-Host "Building Velora-Setup.exe..."
    go build -tags installer -ldflags="-H=windowsgui -s -w" -o (Join-Path $root "build\bin\Velora-Setup.exe") ./cmd/installer
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "Installer created: build\bin\Velora-Setup.exe"
} finally {
    Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $assetsDir "Velora.exe")
}
