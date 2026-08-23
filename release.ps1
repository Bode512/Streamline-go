$ErrorActionPreference = 'Stop'

$node = 'C:\Users\User\Downloads\node-v24.19.0-win-x64'
$env:Path = "$node;$(go env GOPATH)\bin;$env:Path"

Write-Host 'Building embedded Core...'
Remove-Item 'desktop\embedded\streamline-core.exe' -Force -ErrorAction SilentlyContinue
go build -o 'desktop\embedded\streamline-core.exe' .

if (-not (Test-Path 'bin\HandBrakeCLI.exe')) {
    throw 'Missing bin\HandBrakeCLI.exe'
}
Copy-Item 'bin\HandBrakeCLI.exe' 'desktop\embedded\HandBrakeCLI.exe' -Force

Push-Location desktop
try {
    wails build
} finally {
    Pop-Location
}

Write-Host 'Release ready: desktop\build\bin\desktop.exe'