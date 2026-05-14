# Hero CLI install script (Windows / PowerShell).
#
# Usage:
#   irm https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.ps1 | iex
#
# Environment:
#   $env:HERO_VERSION   pin a specific version (e.g. v0.9.1); default: latest
#   $env:HERO_INSTALL   install directory; default: $env:LOCALAPPDATA\Programs\hero

$ErrorActionPreference = 'Stop'

$Repo     = 'hero-engine/hero-releases'
$Releases = "https://github.com/$Repo/releases"

function Write-Info($msg)  { Write-Host $msg }
function Write-Ok($msg)    { Write-Host $msg -ForegroundColor Green }
function Write-Err($msg)   { Write-Host "error: $msg" -ForegroundColor Red }
function Fatal($msg)       { Write-Err $msg; exit 1 }

# Detect arch.
$archRaw = $env:PROCESSOR_ARCHITECTURE
switch ($archRaw) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { Fatal "unsupported architecture: $archRaw" }
}

# Resolve version.
$version = $env:HERO_VERSION
if (-not $version) {
    Write-Info "Resolving latest release..."
    try {
        $latest  = Invoke-WebRequest -UseBasicParsing -MaximumRedirection 0 -ErrorAction SilentlyContinue "$Releases/latest"
    } catch {
        $latest = $_.Exception.Response
    }
    $location = $latest.Headers['Location']
    if (-not $location) { Fatal "could not determine latest version" }
    $version  = ($location -split '/tag/')[-1]
}
if ($version -notmatch '^v') { $version = "v$version" }
$versionBare = $version.TrimStart('v')

$archive = "hero_${versionBare}_windows_${arch}.zip"
$url     = "$Releases/download/$version/$archive"
$sumsUrl = "$Releases/download/$version/checksums.txt"

# Pick install dir.
$installDir = $env:HERO_INSTALL
if (-not $installDir) { $installDir = Join-Path $env:LOCALAPPDATA 'Programs\hero' }
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Download.
$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "hero-install-$([guid]::NewGuid().Guid)") -Force
try {
    $archivePath = Join-Path $tmp.FullName $archive
    Write-Info "Downloading hero $version (windows/$arch)..."
    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $archivePath

    # Best-effort checksum verification.
    try {
        $sumsPath = Join-Path $tmp.FullName 'checksums.txt'
        Invoke-WebRequest -UseBasicParsing -Uri $sumsUrl -OutFile $sumsPath
        $expected = (Get-Content $sumsPath | Where-Object { $_ -match "\s$([regex]::Escape($archive))$" }) -split '\s+' | Select-Object -First 1
        if ($expected) {
            $actual = (Get-FileHash -Algorithm SHA256 $archivePath).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                Fatal "checksum mismatch (expected $expected, got $actual)"
            }
        }
    } catch {
        Write-Info "(checksum file not available; skipping verification)"
    }

    # Extract over the install dir (overwrite hero.exe).
    Expand-Archive -Path $archivePath -DestinationPath $installDir -Force
} finally {
    Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
}

$dest = Join-Path $installDir 'hero.exe'
if (-not (Test-Path $dest)) { Fatal "install failed: $dest not found after extract" }
Write-Ok "Installed hero $version to $dest"

# Add to user PATH if missing.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$installDir*") {
    $newPath = if ($userPath) { "$userPath;$installDir" } else { $installDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Info ""
    Write-Info "Added $installDir to your user PATH."
    Write-Info "Open a new terminal for the change to take effect."
}

Write-Info ""
Write-Info "Run 'hero --help' to get started."
