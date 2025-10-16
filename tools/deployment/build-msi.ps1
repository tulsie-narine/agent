# Windows Inventory Agent MSI Build Script
# Requires WiX Toolset to be installed

param(
    [string]$Configuration = "Release",
    [string]$Platform = "x64",
    [string]$OutputPath = ".\bin",
    [string]$Version = "1.0.0.0"
)

# Configuration
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$InstallerSource = Join-Path $PSScriptRoot "installer.wxs"
$OutputDir = Join-Path $ProjectRoot $OutputPath
$BuildDir = Join-Path $PSScriptRoot "build"

# Validate required paths
if ([string]::IsNullOrEmpty($ProjectRoot)) {
    throw "ProjectRoot is empty or null"
}
if ([string]::IsNullOrEmpty($OutputDir)) {
    throw "OutputDir is empty or null"
}
if ([string]::IsNullOrEmpty($BuildDir)) {
    throw "BuildDir is empty or null"
}
if ([string]::IsNullOrEmpty($Version)) {
    throw "Version is empty or null"
}

# Ensure output directories exist
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

if (!(Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
}

# Check for WiX Toolset
$wixPath = "${env:ProgramFiles(x86)}\WiX Toolset v3.14\bin"
if (!(Test-Path $wixPath)) {
    $wixPath = "${env:ProgramFiles}\WiX Toolset v3.14\bin"
}

if (!(Test-Path $wixPath)) {
    Write-Error "WiX Toolset v3.14 not found. Please install WiX Toolset from https://wixtoolset.org/"
    exit 1
}

# Add WiX to PATH
$env:PATH = "$wixPath;$env:PATH"

# Build the agent executable first
Write-Host "Building Windows Inventory Agent executable..." -ForegroundColor Green

Push-Location $ProjectRoot
try {
    # Build Go executable
    $goCmd = "go build -o `"$OutputDir\WindowsInventoryAgent.exe`" -ldflags `"-X main.version=$Version`" .\agent"
    Invoke-Expression $goCmd

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build agent executable"
        exit 1
    }

    Write-Host "Agent executable built successfully" -ForegroundColor Green
} finally {
    Pop-Location
}

# Generate WiX object file
Write-Host "Compiling WiX source..." -ForegroundColor Green

$candleCmd = 'candle.exe -dConfiguration=' + $Configuration + ' -dPlatform=' + $Platform + ' -dOutputPath=' + $OutputDir + ' -dProjectRoot=' + $ProjectRoot + ' -out "' + $BuildDir + '\installer.wixobj" "' + $InstallerSource + '"'
Invoke-Expression $candleCmd

if ($LASTEXITCODE -ne 0) {
    Write-Error "WiX compilation failed"
    exit 1
}

Write-Host "WiX compilation completed" -ForegroundColor Green

# Link MSI
Write-Host "Linking MSI installer..." -ForegroundColor Green

$lightCmd = "light.exe -out `"$OutputDir\WindowsInventoryAgent-$Version.msi`" `"$BuildDir\installer.wixobj`" -ext WixUIExtension -ext WixNetFxExtension"
Invoke-Expression $lightCmd

if ($LASTEXITCODE -ne 0) {
    Write-Error "MSI linking failed"
    exit 1
}

Write-Host "MSI installer created successfully: $OutputDir\WindowsInventoryAgent-$Version.msi" -ForegroundColor Green

# Clean up build files
Write-Host "Cleaning up build files..." -ForegroundColor Yellow
Remove-Item $BuildDir -Recurse -Force

Write-Host "Build completed successfully!" -ForegroundColor Green