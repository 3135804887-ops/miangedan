# LiveKit 本地开发启动脚本（Windows）
# 追踪：docs/operations/LIVEKIT-RUNBOOK.md；infra/modules/sfu/README.md
# 用法：powershell -ExecutionPolicy Bypass -File infra/modules/sfu/start-local.ps1
param(
    [string]$Version = "v1.13.5"
)

$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$workDir = Join-Path $repoRoot 'work\livekit'
$exe = Join-Path $workDir 'livekit-server.exe'
$configPath = Join-Path $workDir 'livekit.dev.yaml'
$envPath = Join-Path $workDir '.env.local'
$logPath = Join-Path $workDir 'livekit.log'
$errPath = Join-Path $workDir 'livekit.err.log'

New-Item -ItemType Directory -Path $workDir -Force | Out-Null

$running = Get-Process -Name 'livekit-server' -ErrorAction SilentlyContinue
if ($running) {
    Write-Output "livekit-server already running (PID $($running.Id)); URL: ws://localhost:7880"
    if (Test-Path $envPath) { Get-Content $envPath }
    exit 0
}

if (-not (Test-Path $exe)) {
    $version = $Version.TrimStart('v')
    $zip = Join-Path $workDir "livekit_${version}_windows_amd64.zip"
    $url = "https://github.com/livekit/livekit/releases/download/$Version/livekit_${version}_windows_amd64.zip"
    Write-Output "downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $zip
    Expand-Archive -Path $zip -DestinationPath $workDir -Force
    Remove-Item -LiteralPath $zip -Force
}
if (-not (Test-Path $exe)) {
    throw "livekit-server.exe not found after download: $exe"
}

$apiKey = $null
$apiSecret = $null
if (Test-Path $envPath) {
    $envLines = Get-Content $envPath
    $apiKey = (($envLines | Where-Object { $_ -like 'WEBRTC_API_KEY=*' }) -replace '^WEBRTC_API_KEY=', '')
    $apiSecret = (($envLines | Where-Object { $_ -like 'WEBRTC_API_SECRET=*' }) -replace '^WEBRTC_API_SECRET=', '')
}
if (-not $apiKey -or -not $apiSecret) {
    $chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'.ToCharArray()
    $apiKey = 'devkey-' + ((1..24 | ForEach-Object { $chars | Get-Random }) -join '')
    $apiSecret = ((1..48 | ForEach-Object { $chars | Get-Random }) -join '')
}

$config = @"
port: 7880
bind_addresses:
  - "127.0.0.1"
rtc:
  tcp_port: 7881
  port_range_start: 50000
  port_range_end: 50100
  use_external_ip: false
  enable_loopback_candidate: true
keys:
  $apiKey: $apiSecret
logging:
  level: info
"@
Set-Content -Path $configPath -Value $config -Encoding UTF8
$envContent = "WEBRTC_SFU_URL=ws://localhost:7880`nWEBRTC_API_KEY=$apiKey`nWEBRTC_API_SECRET=$apiSecret`n"
Set-Content -Path $envPath -Value $envContent -Encoding UTF8

$proc = Start-Process -FilePath $exe -ArgumentList @('--config', $configPath) -WindowStyle Hidden -RedirectStandardOutput $logPath -RedirectStandardError $errPath -PassThru
Start-Sleep -Seconds 3
try {
    $resp = Invoke-WebRequest -Uri 'http://localhost:7880/' -UseBasicParsing -TimeoutSec 5
    Write-Output "livekit-server started (PID $($proc.Id)); HTTP $($resp.StatusCode) at http://localhost:7880/"
} catch {
    Write-Output "livekit-server process started (PID $($proc.Id)) but health check failed; see $logPath and $errPath"
}
Write-Output "env file: $envPath"
Get-Content $envPath
