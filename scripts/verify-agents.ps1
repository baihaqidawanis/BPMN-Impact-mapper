param([switch]$Browser)

$ErrorActionPreference = 'Stop'
$mapperRoot = Split-Path -Parent $PSScriptRoot
$checks = [System.Collections.Generic.List[object]]::new()
$previousCache = $env:GOCACHE
$previousToolchain = $env:GOTOOLCHAIN
$started = [DateTime]::UtcNow
$successful = $false

function Invoke-AgentCheck {
    param([string]$Name, [string]$Program, [string[]]$Arguments)
    Write-Host "Checking $Name"
    & $Program @Arguments
    $passed = $LASTEXITCODE -eq 0
    $checks.Add([pscustomobject]@{ name = $Name; passed = $passed })
    if (-not $passed) { throw "$Name failed. See the command output above." }
}

Push-Location -LiteralPath $mapperRoot
try {
    $env:GOCACHE = Join-Path $mapperRoot '.gocache'
    $env:GOTOOLCHAIN = 'local'
    $packages = @('./internal/planning', './internal/llm', './internal/agenttools', './internal/agentjobs', './internal/api', './internal/filestore')
    Invoke-AgentCheck 'Offline agent contracts and HTTP pipeline' 'go' (@('test', '-count=1') + $packages)
    Invoke-AgentCheck 'Go vet' 'go' (@('vet') + $packages)
    Invoke-AgentCheck 'UI text encoding' 'node' @('scripts/check-ui-encoding.mjs')
    Push-Location -LiteralPath (Join-Path $mapperRoot 'web')
    try {
        Invoke-AgentCheck 'Frontend types' 'npm.cmd' @('run', 'typecheck')
        if ($Browser) {
            Invoke-AgentCheck 'Planning browser flow' 'npm.cmd' @('run', 'test:e2e', '--', 'qwen-review.spec.ts', '--workers=1')
        }
    } finally { Pop-Location }
    $successful = $true
} finally {
    $reportDirectory = Join-Path $mapperRoot 'outputs/agent-readiness'
    New-Item -ItemType Directory -Force -Path $reportDirectory | Out-Null
    [pscustomobject]@{
        scope = 'Offline integration contracts; no lab inference or model-quality claim'
        started_utc = $started.ToString('o')
        finished_utc = [DateTime]::UtcNow.ToString('o')
        passed = $successful
        browser_requested = [bool]$Browser
        checks = @($checks.ToArray())
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $reportDirectory 'result.json') -Encoding UTF8
    $env:GOCACHE = $previousCache
    $env:GOTOOLCHAIN = $previousToolchain
    Pop-Location
}
Write-Host 'Offline checks passed. Report: outputs/agent-readiness/result.json'
