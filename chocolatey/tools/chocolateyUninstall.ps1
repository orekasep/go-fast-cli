$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$fastExe = Join-Path $toolsDir "fast.exe"
$goFastCliExe = Join-Path $toolsDir "go-fast-cli.exe"

if (Test-Path $fastExe) {
  Remove-Item $fastExe -Force -ErrorAction SilentlyContinue
}
if (Test-Path $goFastCliExe) {
  Remove-Item $goFastCliExe -Force -ErrorAction SilentlyContinue
}

Get-ChildItem -Path $toolsDir -Directory -Filter "go-fast-cli_*" | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
