# MinIO 本地开发启动脚本（Windows；OD-01 自建矩阵：媒体存储自建）
# 用法：powershell -ExecutionPolicy Bypass -File infra/modules/object-storage/start-local.ps1
$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$workDir = Join-Path $repoRoot 'work\minio'
$exe = Join-Path $workDir 'minio.exe'
$dataDir = Join-Path $workDir 'data'
$envPath = Join-Path $workDir '.env.local'

New-Item -ItemType Directory -Path $dataDir -Force | Out-Null

$running = Get-Process -Name 'minio' -ErrorAction SilentlyContinue
if ($running) {
    Write-Output "minio already running (PID $($running.Id)); http://127.0.0.1:9000"
    exit 0
}

if (-not (Test-Path $exe) -or (Get-Item $exe).Length -lt 50MB) {
    $url = 'https://dl.min.io/server/minio/release/windows-amd64/minio.exe'
    Write-Output "downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $exe -TimeoutSec 120
    if ((Get-Item $exe).Length -lt 50MB) {
        throw 'minio.exe 下载不完整，请检查网络（dl.min.io 需要可达或走代理）'
    }
}

$rootUser = 'minioadmin'
$rootPassword = 'minioadmin' # 仅本地开发；云上部署必须由 KMS 注入
$env:MINIO_ROOT_USER = $rootUser
$env:MINIO_ROOT_PASSWORD = $rootPassword
Set-Content -Path $envPath -Value "MINIO_ROOT_USER=$rootUser`nMINIO_ROOT_PASSWORD=$rootPassword`n" -Encoding UTF8

$proc = Start-Process -FilePath $exe -ArgumentList @('server', $dataDir, '--address', '127.0.0.1:9000', '--console-address', '127.0.0.1:9001') -WindowStyle Hidden -RedirectStandardOutput (Join-Path $workDir 'minio.log') -RedirectStandardError (Join-Path $workDir 'minio.err.log') -PassThru
Start-Sleep -Seconds 3
Write-Output "minio started (PID $($proc.Id)); API http://127.0.0.1:9000 console http://127.0.0.1:9001"
Write-Output "env file: $envPath"
