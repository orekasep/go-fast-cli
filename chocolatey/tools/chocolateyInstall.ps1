$ErrorActionPreference = 'Stop'

$packageName = 'go-fast-cli'
$toolsDir    = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
$version     = '1.1.0'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url            = "https://github.com/orekasep/go-fast-cli/releases/download/v${version}/go-fast-cli_v${version}_windows-amd64.zip"
  checksum       = 'ec0d370740006a0ad258eec4822c54fa8358d6e9e474a03fd73f67a5fcf919d6'
  checksumType   = 'sha256'
  url64bit       = "https://github.com/orekasep/go-fast-cli/releases/download/v${version}/go-fast-cli_v${version}_windows-amd64.zip"
  checksum64     = 'ec0d370740006a0ad258eec4822c54fa8358d6e9e474a03fd73f67a5fcf919d6'
  checksumType64 = 'sha256'
}

if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
  $packageArgs.url64bit       = "https://github.com/orekasep/go-fast-cli/releases/download/v${version}/go-fast-cli_v${version}_windows-arm64.zip"
  $packageArgs.checksum64     = '8c448936a891b0e7165589ad8c8b483ccc39786b0223a2db55da0efc65698e33'
  $packageArgs.checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# The ZIP unpacks into a subfolder 'go-fast-cli_v<version>_windows-<arch>'
# Move fast.exe to $toolsDir so Chocolatey automatically creates shims
$extractedDir = Get-ChildItem -Path $toolsDir -Directory -Filter "go-fast-cli_*" | Select-Object -First 1
if ($extractedDir) {
  $extractedFastExe = Join-Path $extractedDir.FullName "fast.exe"
  if (Test-Path $extractedFastExe) {
    Copy-Item -Path $extractedFastExe -Destination (Join-Path $toolsDir "fast.exe") -Force
    Copy-Item -Path $extractedFastExe -Destination (Join-Path $toolsDir "go-fast-cli.exe") -Force
  }
}
