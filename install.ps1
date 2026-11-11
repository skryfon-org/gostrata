# gostrata installer for Windows (PowerShell).
#   irm https://raw.githubusercontent.com/skryfon-org/gostrata/main/install.ps1 | iex
# Optional: $env:VERSION = "v1.1.1"
$ErrorActionPreference = "Stop"
$repo = "skryfon-org/gostrata"

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$version = $env:VERSION
if (-not $version) {
  $version = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
}
$num = $version.TrimStart("v")
$archive = "gostrata_${num}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/$version"

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("gostrata-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Write-Host "Downloading gostrata $version (windows/$arch)..."
  Invoke-WebRequest "$base/$archive" -OutFile "$tmp\$archive"
  Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt"

  $expected = ((Get-Content "$tmp\checksums.txt" | Where-Object { $_ -match [regex]::Escape($archive) + "$" }) -split "\s+")[0]
  $actual = (Get-FileHash "$tmp\$archive" -Algorithm SHA256).Hash.ToLower()
  if (-not $expected -or $expected -ne $actual) { throw "Checksum verification failed" }

  $dir = Join-Path $env:LOCALAPPDATA "gostrata\bin"
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
  Expand-Archive "$tmp\$archive" -DestinationPath $tmp -Force
  Copy-Item "$tmp\gostrata.exe" "$dir\gostrata.exe" -Force

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (($userPath -split ";") -notcontains $dir) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$dir", "User")
    Write-Host "Added $dir to your PATH. Restart your terminal."
  }
  Write-Host "Installed gostrata to $dir\gostrata.exe"
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
