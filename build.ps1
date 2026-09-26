# Construit deckpad.exe (appli tablette incluse). Usage : .\build.ps1
$ErrorActionPreference = 'Stop'

Push-Location web
npm ci
npm run build
Pop-Location

Push-Location agent
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -o ..\bin\deckpad.exe .
Pop-Location

Write-Host 'OK : bin\deckpad.exe'
