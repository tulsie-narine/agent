# Windows Inventory Agent Load Testing Runner
# Requires k6 to be installed: https://k6.io/docs/get-started/installation/

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet('standard', 'stress', 'spike', 'endurance', 'all')]
    [string]$TestType = 'standard',

    [Parameter(Mandatory=$false)]
    [string]$BaseUrl = 'http://localhost:8080/api/v1',

    [Parameter(Mandatory=$false)]
    [string]$OutputPath = '.\results',

    [Parameter(Mandatory=$false)]
    [switch]$DryRun,

    [Parameter(Mandatory=$false)]
    [switch]$Verbose
)

# Configuration
$ScriptRoot = $PSScriptRoot
$TestScript = Join-Path $ScriptRoot 'load-test.js'
$ConfigFile = Join-Path $ScriptRoot 'k6-config.yaml'
$ResultsDir = Join-Path $ScriptRoot $OutputPath

# Ensure results directory exists
if (!(Test-Path $ResultsDir)) {
    New-Item -ItemType Directory -Path $ResultsDir -Force | Out-Null
}

# Check if k6 is installed
try {
    $k6Version = & k6 version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "k6 not found"
    }
} catch {
    Write-Error "k6 is not installed or not in PATH. Please install k6 from https://k6.io/docs/get-started/installation/"
    Write-Host "You can install k6 using: choco install k6" -ForegroundColor Yellow
    exit 1
}

Write-Host "Windows Inventory Agent Load Testing" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green
Write-Host "k6 version: $k6Version" -ForegroundColor Cyan
Write-Host "Test Type: $TestType" -ForegroundColor Cyan
Write-Host "Base URL: $BaseUrl" -ForegroundColor Cyan
Write-Host "Results Path: $ResultsDir" -ForegroundColor Cyan
Write-Host ""

# Function to run a specific test scenario
function Run-TestScenario {
    param(
        [string]$Scenario,
        [string]$Description
    )

    Write-Host "Running $Description..." -ForegroundColor Yellow

    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $resultFile = Join-Path $ResultsDir "load-test-${Scenario}-${timestamp}.json"
    $summaryFile = Join-Path $ResultsDir "summary-${Scenario}-${timestamp}.txt"

    $k6Args = @(
        'run',
        '--config', $ConfigFile,
        '--tag', "scenario=$Scenario",
        '--out', "json=$resultFile",
        '--summary-export', $summaryFile
    )

    if ($Verbose) {
        $k6Args += '--verbose'
    }

    if ($DryRun) {
        $k6Args = @('inspect') + $k6Args
    }

    $k6Args += @(
        '--env', "BASE_URL=$BaseUrl",
        $TestScript
    )

    # Set the scenario environment variable
    $env:K6_SCENARIO = $Scenario

    try {
        if ($DryRun) {
            Write-Host "Dry run mode - inspecting test script..." -ForegroundColor Magenta
            & k6 $k6Args
        } else {
            Write-Host "Starting load test..." -ForegroundColor Green
            $startTime = Get-Date

            & k6 $k6Args

            $endTime = Get-Date
            $duration = $endTime - $startTime

            Write-Host "Test completed in $($duration.TotalSeconds) seconds" -ForegroundColor Green
            Write-Host "Results saved to: $resultFile" -ForegroundColor Cyan
            Write-Host "Summary saved to: $summaryFile" -ForegroundColor Cyan
        }
    } catch {
        Write-Error "Test execution failed: $_"
        return $false
    } finally {
        Remove-Item Env:\K6_SCENARIO -ErrorAction SilentlyContinue
    }

    return $true
}

# Run tests based on selected type
$testResults = @()

switch ($TestType) {
    'standard' {
        $testResults += Run-TestScenario -Scenario 'standard_load' -Description 'Standard Load Test'
    }
    'stress' {
        $testResults += Run-TestScenario -Scenario 'stress_test' -Description 'Stress Test'
    }
    'spike' {
        $testResults += Run-TestScenario -Scenario 'spike_test' -Description 'Spike Test'
    }
    'endurance' {
        $testResults += Run-TestScenario -Scenario 'endurance_test' -Description 'Endurance Test'
    }
    'all' {
        $testResults += Run-TestScenario -Scenario 'standard_load' -Description 'Standard Load Test'
        $testResults += Run-TestScenario -Scenario 'stress_test' -Description 'Stress Test'
        $testResults += Run-TestScenario -Scenario 'spike_test' -Description 'Spike Test'
        $testResults += Run-TestScenario -Scenario 'endurance_test' -Description 'Endurance Test'
    }
}

# Summary
$passedTests = ($testResults | Where-Object { $_ -eq $true }).Count
$totalTests = $testResults.Count

Write-Host ""
Write-Host "Test Summary" -ForegroundColor Green
Write-Host "============" -ForegroundColor Green
Write-Host "Tests run: $totalTests" -ForegroundColor White
Write-Host "Tests passed: $passedTests" -ForegroundColor Green
Write-Host "Tests failed: $($totalTests - $passedTests)" -ForegroundColor Red

if ($passedTests -eq $totalTests) {
    Write-Host "All tests completed successfully!" -ForegroundColor Green
    exit 0
} else {
    Write-Error "Some tests failed. Check the output above for details."
    exit 1
}