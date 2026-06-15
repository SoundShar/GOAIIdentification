$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

New-Item -ItemType Directory -Force -Path "embeddata" | Out-Null

$yoloPath = "embeddata/yolo11.onnx"
$needYoloExport = $true
if (Test-Path $yoloPath) {
    $opsetCheck = python -c "import onnx; m=onnx.load(r'$yoloPath'); print(m.opset_import[0].version)"
    if ($opsetCheck -eq "17") {
        $needYoloExport = $false
        Write-Host "YOLO11 model exists (opset 17): $yoloPath"
    } else {
        Write-Host "YOLO11 opset $opsetCheck unsupported, re-exporting..."
    }
}
if ($needYoloExport) {
    Write-Host "Exporting YOLO11n ONNX (opset 17, onnxruntime 1.19)..."
    python -c @"
from ultralytics import YOLO
model = YOLO('yolo11n.pt')
model.export(format='onnx', imgsz=640, opset=17, simplify=True)
"@
    $exported = Get-Item "yolo11n.onnx" -ErrorAction SilentlyContinue
    if (-not $exported) {
        throw "YOLO11 export failed, run: pip install ultralytics"
    }
    Copy-Item -Force $exported.FullName $yoloPath
    Remove-Item -Force "yolo11n.onnx" -ErrorAction SilentlyContinue
    Write-Host "YOLO11 model ready: $yoloPath"
}

$yunetUrl = "https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx"
$faceDetectPath = "embeddata/face_detect.onnx"
if (-not (Test-Path $faceDetectPath)) {
    Write-Host "Downloading YuNet face detector..."
    Invoke-WebRequest -Uri $yunetUrl -OutFile $faceDetectPath -UseBasicParsing
} else {
    Write-Host "Face detect model exists: $faceDetectPath"
}

$buffaloUrl = "https://github.com/deepinsight/insightface/releases/download/v0.7/buffalo_sc.zip"
$buffaloZip = "embeddata/buffalo_sc.zip"
if (-not (Test-Path "embeddata/face_rec.onnx")) {
    Write-Host "Downloading InsightFace buffalo_sc..."
    Invoke-WebRequest -Uri $buffaloUrl -OutFile $buffaloZip -UseBasicParsing
    Expand-Archive -Path $buffaloZip -DestinationPath "embeddata/buffalo_sc" -Force
    if (Test-Path "embeddata/buffalo_sc/w600k_mbf.onnx") {
        Copy-Item -Force "embeddata/buffalo_sc/w600k_mbf.onnx" "embeddata/face_rec.onnx"
    } elseif (Test-Path "embeddata/buffalo_sc/buffalo_sc/w600k_r50.onnx") {
        Copy-Item -Force "embeddata/buffalo_sc/buffalo_sc/w600k_r50.onnx" "embeddata/face_rec.onnx"
    } else {
        throw "face_rec.onnx not found in buffalo_sc package"
    }
    Remove-Item -Force $buffaloZip -ErrorAction SilentlyContinue
} else {
    Write-Host "Face rec model exists: embeddata/face_rec.onnx"
}

$ortVersion = "1.19.2"
$ortDllPath = "embeddata/onnxruntime.dll"
$ortZipUrl = "https://github.com/microsoft/onnxruntime/releases/download/v$ortVersion/onnxruntime-win-x64-$ortVersion.zip"
$ortZip = "embeddata/onnxruntime-win.zip"
if (-not (Test-Path $ortDllPath)) {
    Write-Host "Downloading ONNX Runtime $ortVersion..."
    Invoke-WebRequest -Uri $ortZipUrl -OutFile $ortZip -UseBasicParsing
    Expand-Archive -Path $ortZip -DestinationPath "embeddata/ort_tmp" -Force
    Copy-Item -Force "embeddata/ort_tmp/onnxruntime-win-x64-$ortVersion/lib/onnxruntime.dll" $ortDllPath
    Remove-Item -Recurse -Force "embeddata/ort_tmp"
    Remove-Item -Force $ortZip
} else {
    Write-Host "onnxruntime.dll exists: $ortDllPath"
}

$required = @("embeddata/yolo11.onnx", "embeddata/face_detect.onnx", "embeddata/face_rec.onnx", "embeddata/onnxruntime.dll")
foreach ($file in $required) {
    if (-not (Test-Path $file)) {
        throw "Missing required file: $file"
    }
}

Write-Host "Embed assets ready in embeddata/ (will be compiled into yks-tool.exe)"
