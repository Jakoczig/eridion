# Whole-app idle sample for the native ERIDION binary (Windows).
# Starts the app with an isolated synthetic profile and journal, waits for
# warm-up, then samples CPU time and working set of the app process and every
# descendant process (WebView2 children) at a fixed interval.
# Usage:
#   powershell -File scripts/testing/native-idle-sample.ps1 -Exe go\bin\daedalus.exe `
#     -JournalDir <synthetic journals> -OutFile <result.json> [-WarmupSeconds 60] [-Seconds 300] [-Interval 5]
param(
  [Parameter(Mandatory = $true)][string]$Exe,
  [Parameter(Mandatory = $true)][string]$JournalDir,
  [Parameter(Mandatory = $true)][string]$OutFile,
  [int]$WarmupSeconds = 60,
  [int]$Seconds = 300,
  [int]$Interval = 5
)
$ErrorActionPreference = 'Stop'
$profileDir = Join-Path ([System.IO.Path]::GetTempPath()) ('eridion-perf-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $profileDir | Out-Null

function Get-Tree([int]$rootId) {
  $all = Get-CimInstance Win32_Process | Select-Object ProcessId, ParentProcessId, Name
  $ids = @($rootId); $changed = $true
  while ($changed) {
    $changed = $false
    foreach ($p in $all) { if ($ids -contains $p.ParentProcessId -and -not ($ids -contains $p.ProcessId)) { $ids += $p.ProcessId; $changed = $true } }
  }
  return $ids
}
function Get-Sample([int]$rootId) {
  $cpu = 0.0; $ws = 0; $count = 0; $webview = 0
  foreach ($id in (Get-Tree $rootId)) {
    $p = Get-Process -Id $id -ErrorAction SilentlyContinue
    if ($null -eq $p) { continue }
    $count++
    if ($p.ProcessName -like 'msedgewebview2*') { $webview++ }
    $cpu += $p.TotalProcessorTime.TotalSeconds
    $ws += $p.WorkingSet64
  }
  return [pscustomobject]@{ cpu = $cpu; ws = $ws; processes = $count; webview = $webview }
}

$proc = Start-Process -FilePath $Exe -ArgumentList @('--data-dir', $profileDir, '--journal-dir', $JournalDir) -PassThru
try {
  Start-Sleep -Seconds $WarmupSeconds
  $first = Get-Sample $proc.Id
  $samples = @()
  $begin = Get-Date
  while (((Get-Date) - $begin).TotalSeconds -lt $Seconds) {
    Start-Sleep -Seconds $Interval
    $s = Get-Sample $proc.Id
    $samples += [pscustomobject]@{ t = [math]::Round(((Get-Date) - $begin).TotalSeconds, 1); cpu = [math]::Round($s.cpu, 3); wsMB = [math]::Round($s.ws / 1MB, 1); processes = $s.processes; webview = $s.webview }
  }
  $last = $samples[-1]
  $elapsed = ((Get-Date) - $begin).TotalSeconds
  $ws = $samples | ForEach-Object { $_.wsMB }
  $result = [pscustomobject]@{
    exe = (Split-Path -Leaf $Exe); warmupSeconds = $WarmupSeconds; seconds = [math]::Round($elapsed, 1); logicalProcessors = [Environment]::ProcessorCount
    cpuSecondsDuringSample = [math]::Round($last.cpu - $first.cpu, 3)
    averageCpuPercentOneCore = [math]::Round(100 * ($last.cpu - $first.cpu) / $elapsed, 2)
    workingSetStartMB = [math]::Round($first.ws / 1MB, 1); workingSetEndMB = $last.wsMB
    workingSetMaxMB = ($ws | Measure-Object -Maximum).Maximum
    processes = $last.processes; webviewProcesses = $last.webview
    samples = $samples
  }
  $result | ConvertTo-Json -Depth 4 | Set-Content -Encoding utf8 $OutFile
} finally {
  foreach ($id in (Get-Tree $proc.Id)) { Stop-Process -Id $id -Force -ErrorAction SilentlyContinue }
  Start-Sleep -Seconds 2
  Remove-Item -Recurse -Force $profileDir -ErrorAction SilentlyContinue
}
