# Install WiX Toolset using Chocolatey
# This script assumes Chocolatey is installed. If not, run the Chocolatey installer first.

# Check if Chocolatey is installed
if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Host "Chocolatey is not installed. Installing Chocolatey..." -ForegroundColor Yellow
    Set-ExecutionPolicy Bypass -Scope Process -Force
    iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
    Write-Host "Chocolatey installed. Please restart PowerShell and run this script again." -ForegroundColor Green
    exit
}

# Install WiX Toolset
Write-Host "Installing WiX Toolset..." -ForegroundColor Green
choco install wixtoolset -y

# Verify installation
if (Get-Command candle.exe -ErrorAction SilentlyContinue) {
    Write-Host "WiX Toolset installed successfully. candle.exe and light.exe are now available." -ForegroundColor Green
} else {
    Write-Host "WiX Toolset installation may have failed. Please check the output above." -ForegroundColor Red
}