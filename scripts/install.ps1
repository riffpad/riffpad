$ErrorActionPreference = "Stop"

$installDir = Join-Path $env:LOCALAPPDATA "riffpad"
$exe = Join-Path $installDir "riffpad.exe"

Write-Host "Installing Riffpad to $installDir ..."
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$release = Invoke-RestMethod "https://api.github.com/repos/riffpad/riffpad/releases/latest"
$asset = $release.assets | Where-Object { $_.name -eq "riffpad-windows-amd64.exe" }
if (-not $asset) {
  throw "Latest release has no riffpad-windows-amd64.exe asset"
}
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $exe

# Add the install dir to the user PATH if missing.
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
  Write-Host "Added $installDir to your user PATH (reopen the terminal to use riffpad)."
}

# Autostart the daemon at logon via a hidden scheduled task.
schtasks /Create /TN "RiffpadDaemon" /TR "\"$exe\" _daemon" /SC ONLOGON /RL LIMITED /F | Out-Null

Write-Host ""
Write-Host "Done. Run 'riffpad version' in a new terminal."
