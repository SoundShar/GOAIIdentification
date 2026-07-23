$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

Write-Host "Downloading embed assets..."
powershell -ExecutionPolicy Bypass -File scripts/download-deps.ps1

New-Item -ItemType Directory -Force -Path "build" | Out-Null

if (-not (Test-Path "assets/icon.ico")) {
    throw "Missing app icon: assets/icon.ico"
}

Write-Host "Generating Windows version resource..."
Remove-Item -Force resource.syso -ErrorAction SilentlyContinue
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 `
    -icon="assets/icon.ico" `
    -application-icon="assets/icon.ico" `
    -64=true
if ($LASTEXITCODE -ne 0) {
    throw "goversioninfo failed; set GOPROXY (e.g. https://goproxy.cn,direct) and retry."
}
if (-not (Test-Path "resource.syso")) {
    throw "resource.syso not generated; exe will have no icon."
}
Write-Host "Windows resource ready (icon: assets/icon.ico)"

Write-Host "Building single-file yks-tool.exe (models embedded)..."
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w -H windowsgui" -o build/yks-tool.exe .

$exeSize = (Get-Item "build/yks-tool.exe").Length / 1MB
Write-Host ("Build complete: build/yks-tool.exe ({0:N1} MB, standalone; TLS CA generated at runtime)" -f $exeSize)
