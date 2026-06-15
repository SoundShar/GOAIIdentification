$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

Write-Host "Downloading embed assets..."
powershell -ExecutionPolicy Bypass -File scripts/download-deps.ps1

New-Item -ItemType Directory -Force -Path "build" | Out-Null

Write-Host "Generating Windows version resource..."
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

Write-Host "Building single-file yks-tool.exe (models embedded)..."
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w -H windowsgui" -o build/yks-tool.exe .

$exeSize = (Get-Item "build/yks-tool.exe").Length / 1MB
Write-Host ("Build complete: build/yks-tool.exe ({0:N1} MB, standalone)" -f $exeSize)
