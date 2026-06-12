# Instalador para Windows do monitor do Mancer Mystic G1.
#
# Compila o binario, instala em %LOCALAPPDATA%\MancerCooler e cria DUAS tarefas
# agendadas que sobem no logon:
#   1. "LibreHardwareMonitor (admin)" - o LHM elevado, que expoe o servidor web
#      com a temperatura da CPU (ele so le a CPU rodando como administrador).
#   2. "MancerCoolerMonitor"          - o monitor que le a temperatura e a
#      escreve no display do cooler; reinicia sozinho se cair.
#
# Uso (PowerShell normal, NAO precisa abrir como admin - o script pede UAC so
# na hora de registrar as tarefas):
#   .\install-windows.ps1
#
# Pre-requisitos:
#   - Go instalado (https://go.dev/dl/)
#   - LibreHardwareMonitor instalado (winget install LibreHardwareMonitor.LibreHardwareMonitor)
#     e com "Options > Remote Web Server > Run" ja ativado uma vez (fica salvo).

param([switch]$Elevated)

$ErrorActionPreference = 'Stop'

$taskMonitor = 'MancerCoolerMonitor'
$taskLHM     = 'LibreHardwareMonitor-admin'
$installDir  = Join-Path $env:LOCALAPPDATA 'MancerCooler'
$monitorExe  = Join-Path $installDir 'mancer-cooler-monitor.exe'

function Find-LHM {
    $base = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages'
    $pkg = Get-ChildItem $base -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -like 'LibreHardwareMonitor*' } | Select-Object -First 1
    if ($pkg) {
        $exe = Get-ChildItem $pkg.FullName -Recurse -Filter 'LibreHardwareMonitor.exe' -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($exe) { return $exe.FullName }
    }
    return $null
}

if (-not $Elevated) {
    # ===== Fase 1 (usuario normal): apenas compilar (go precisa do PATH do usuario) =====
    # -H=windowsgui: gera um app GUI, sem abrir janela de console (roda oculto).
    # Os logs vao para %LOCALAPPDATA%\MancerCooler\monitor.log (ver log_windows.go).
    Write-Host '==> Compilando o binario (sem console)...'
    go build -ldflags '-H=windowsgui' -o (Join-Path $PSScriptRoot 'mancer-cooler-monitor.exe') $PSScriptRoot
    if ($LASTEXITCODE -ne 0) { throw "Falha ao compilar (go build retornou $LASTEXITCODE)." }

    # ===== Fase 2: instalar e registrar (precisa de admin p/ encerrar a tarefa em execucao) =====
    Write-Host '==> Instalando e registrando as tarefas (vai aparecer o prompt do UAC - clique em Sim)...'
    Start-Process powershell -Verb RunAs -Wait -ArgumentList @(
        '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', "`"$PSCommandPath`"", '-Elevated'
    )
    Write-Host ''
    Write-Host 'Pronto! As tarefas sobem automaticamente a cada logon.'
    Write-Host 'Para desinstalar:'
    Write-Host "  Unregister-ScheduledTask -TaskName '$taskMonitor' -Confirm:`$false"
    Write-Host "  Unregister-ScheduledTask -TaskName '$taskLHM' -Confirm:`$false"
    return
}

# ===== Daqui pra baixo roda ELEVADO (admin) =====
$lhmExe = Find-LHM
if (-not $lhmExe) {
    throw 'LibreHardwareMonitor nao encontrado. Instale com: winget install LibreHardwareMonitor.LibreHardwareMonitor'
}

# Evita instancias duplicadas: encerra qualquer LHM/monitor aberto e remove a
# entrada de auto-inicio propria do LHM (a tarefa agendada passa a ser a unica fonte).
Get-Process LibreHardwareMonitor, mancer-cooler-monitor -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500   # garante que o arquivo .exe foi liberado

# Agora que o monitor em execucao foi encerrado, instala o binario recem-compilado.
Write-Host "==> Instalando em $monitorExe"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Copy-Item -Force (Join-Path $PSScriptRoot 'mancer-cooler-monitor.exe') $monitorExe

$runKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
if (Get-ItemProperty -Path $runKey -Name 'LibreHardwareMonitor' -ErrorAction SilentlyContinue) {
    Remove-ItemProperty -Path $runKey -Name 'LibreHardwareMonitor' -ErrorAction SilentlyContinue
}

# Grava a config do LHM com o servidor web JA LIGADO, para que ele suba sozinho
# (sem precisar clicar em "Run") independentemente de como tenha sido fechado.
# Feito com o LHM encerrado acima, senao ele sobrescreveria ao sair.
$lhmCfg = Join-Path (Split-Path $lhmExe) 'LibreHardwareMonitor.config'
@'
<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <appSettings>
    <add key="listenerPort" value="8085" />
    <add key="runWebServerMenuItem" value="true" />
    <add key="minTrayMenuItem" value="true" />
    <add key="minCloseMenuItem" value="true" />
    <add key="startMinMenuItem" value="true" />
  </appSettings>
</configuration>
'@ | Set-Content -Path $lhmCfg -Encoding utf8

$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -RestartCount 999 `
    -RestartInterval (New-TimeSpan -Minutes 1) `
    -ExecutionTimeLimit ([TimeSpan]::Zero)

# 1) LibreHardwareMonitor elevado (expoe o servidor web com a temperatura).
Register-ScheduledTask -TaskName $taskLHM -Force `
    -Action (New-ScheduledTaskAction -Execute $lhmExe) `
    -Trigger $trigger -Settings $settings `
    -User $env:USERNAME -RunLevel Highest | Out-Null

# 2) Monitor do cooler (le a temp e escreve no display).
Register-ScheduledTask -TaskName $taskMonitor -Force `
    -Action (New-ScheduledTaskAction -Execute $monitorExe) `
    -Trigger $trigger -Settings $settings `
    -User $env:USERNAME -RunLevel Highest | Out-Null

Write-Host '==> Tarefas registradas. Iniciando agora...'
Start-ScheduledTask -TaskName $taskLHM
Start-Sleep -Seconds 3   # da tempo do servidor web subir antes do monitor
Start-ScheduledTask -TaskName $taskMonitor
Write-Host '==> OK.'
