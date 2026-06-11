$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

New-Item -ItemType Directory -Force -Path "build" | Out-Null

go build -ldflags="-s -w -H windowsgui" -o build/myapp.exe .

Write-Host "Build complete: build/myapp.exe"
