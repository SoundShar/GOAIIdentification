$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

Write-Host "Downloading embed assets..."
powershell -ExecutionPolicy Bypass -File scripts/download-deps.ps1

$sslCert = Join-Path $projectRoot "ssl/local.sharas.cn_bundle.crt"
$sslKey = Join-Path $projectRoot "ssl/local.sharas.cn.key"
if (-not (Test-Path $sslCert)) {
    throw "Missing TLS cert: $sslCert"
}
if (-not (Test-Path $sslKey)) {
    throw "Missing TLS key: $sslKey"
}

$embedSslDir = Join-Path $projectRoot "embeddata/ssl"
New-Item -ItemType Directory -Force -Path $embedSslDir | Out-Null
Copy-Item -Force $sslCert (Join-Path $embedSslDir "local.sharas.cn_bundle.crt")
Copy-Item -Force $sslKey (Join-Path $embedSslDir "local.sharas.cn.key")
Write-Host "TLS assets copied to embeddata/ssl/"

New-Item -ItemType Directory -Force -Path "build" | Out-Null

Write-Host "Generating Windows version resource..."
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

Write-Host "Building single-file yks-tool.exe (models embedded)..."
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w -H windowsgui" -o build/yks-tool.exe .

$exeSize = (Get-Item "build/yks-tool.exe").Length / 1MB
Write-Host ("Build complete: build/yks-tool.exe ({0:N1} MB, standalone)" -f $exeSize)
