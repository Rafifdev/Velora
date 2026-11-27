param(
    [string]$Version = "1.3.0.0",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$exePath = Join-Path $root "build\bin\Velora.exe"
if (-not $SkipBuild -or -not (Test-Path $exePath)) {
    Stop-Process -Name "Velora" -Force -ErrorAction SilentlyContinue
    Write-Host "Building Velora.exe..."
    wails build -clean
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$makeappx = Get-ChildItem -Path "C:\Program Files (x86)\Windows Kits\10\bin" -Recurse -Filter "makeappx.exe" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match "\\x64\\" } |
    Sort-Object FullName -Descending |
    Select-Object -First 1 -ExpandProperty FullName

if (-not $makeappx) {
    throw "makeappx.exe (Windows 10/11 SDK x64) not found."
}

$py = "C:\Program Files\PostgreSQL\18\pgAdmin 4\python\python.exe"
if (-not (Test-Path $py)) {
    $py = "python"
}

$staging = Join-Path $root "build\msix-staging"
$assetsDir = Join-Path $staging "Assets"
if (Test-Path $staging) { Remove-Item $staging -Recurse -Force }
New-Item -ItemType Directory -Force -Path $assetsDir | Out-Null

try {
    Copy-Item -Force $exePath (Join-Path $staging "Velora.exe")
    Copy-Item -Force (Join-Path $root "build\windows\icon.ico") (Join-Path $assetsDir "icon.ico")

    $iconSrc = Join-Path $root "build\appicon.png"
    & $py -c @"
from PIL import Image
import os

src = r'$iconSrc'
out_dir = r'$assetsDir'
im = Image.open(src).convert('RGBA')

for name, size in [
    ('Square44x44Logo.png', (44, 44)),
    ('Square44x44Logo.targetsize-24_altform-unplated.png', (24, 24)),
    ('Square44x44Logo.targetsize-48_altform-unplated.png', (48, 48)),
    ('Square44x44Logo.targetsize-256_altform-unplated.png', (256, 256)),
    ('Square150x150Logo.png', (150, 150)),
    ('StoreLogo.png', (50, 50)),
]:
    im.resize(size, Image.Resampling.LANCZOS).save(os.path.join(out_dir, name), format='PNG', optimize=True)

wide = Image.new('RGBA', (310, 150), (16, 17, 23, 255))
logo_w = im.resize((110, 110), Image.Resampling.LANCZOS)
wide.alpha_composite(logo_w, ((310 - 110) // 2, (150 - 110) // 2))
wide.save(os.path.join(out_dir, 'Wide310x150Logo.png'), format='PNG', optimize=True)
"@
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    $manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  xmlns:desktop7="http://schemas.microsoft.com/appx/manifest/desktop/windows10/7"
  IgnorableNamespaces="uap rescap desktop7">

  <Identity
    Name="RafifDev.VeloraPlayer-DynamicIsland"
    Publisher="CN=2DFCB639-2FE8-4980-8F7A-030D3E544B21"
    Version="$Version"
    ProcessorArchitecture="x64" />

  <Properties>
    <DisplayName>Velora Player - Dynamic Island</DisplayName>
    <PublisherDisplayName>Rafif Dev</PublisherDisplayName>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>

  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.19041.0" MaxVersionTested="10.0.26100.0" />
  </Dependencies>

  <Resources>
    <Resource Language="en-us" />
  </Resources>

  <Applications>
    <Application Id="Velora" Executable="Velora.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements
        DisplayName="Velora Player - Dynamic Island"
        Description="Modern desktop music player for Windows with native Dynamic Island overlay, synced lyrics, and local-first privacy."
        BackgroundColor="transparent"
        Square150x150Logo="Assets\Square150x150Logo.png"
        Square44x44Logo="Assets\Square44x44Logo.png">
        <uap:DefaultTile Wide310x150Logo="Assets\Wide310x150Logo.png" />
      </uap:VisualElements>
      <Extensions>
        <desktop7:Extension Category="windows.shortcut">
          <desktop7:Shortcut File="[{Desktop}]\Velora.lnk" Icon="[{Package}]\Assets\icon.ico" />
        </desktop7:Extension>
      </Extensions>
    </Application>
  </Applications>

  <Capabilities>
    <rescap:Capability Name="runFullTrust" />
  </Capabilities>
</Package>
"@
    [System.IO.File]::WriteAllText((Join-Path $staging "AppxManifest.xml"), $manifest, [System.Text.UTF8Encoding]::new($false))

    $shortVer = $Version.Split('.')[0..2] -join '.'
    $outMsix = Join-Path $root "build\bin\Velora-$shortVer-x64.msix"
    & $makeappx pack /d $staging /p $outMsix /o
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "MSIX created: $outMsix"
} finally {
    if (Test-Path $staging) { Remove-Item $staging -Recurse -Force -ErrorAction SilentlyContinue }
}
