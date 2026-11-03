# Verificacion LOCAL: lo mismo que el CI (las partes que no necesitan
# herramientas externas). Ejecutar antes de cada commit:
#
#   .\scripts\verify.ps1          # formato, vet y pruebas con cobertura
#   .\scripts\verify.ps1 -Race    # ademas -race, dentro de Docker (Linux trae gcc)
#
# Nota: este archivo usa solo caracteres ASCII (sin tildes) a proposito.
# Windows PowerShell 5.1 lee los .ps1 sin BOM con otra codificacion y una
# tilde puede romper el script.

param([switch]$Race)

$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

function Step($name, [scriptblock]$action) {
    Write-Host "`n==> $name" -ForegroundColor Cyan
    & $action
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FALLO: $name" -ForegroundColor Red
        exit 1
    }
}

Step "Formato (gofmt)" {
    $files = gofmt -l .
    if ($files) { Write-Host "Archivos sin formato:`n$files"; $global:LASTEXITCODE = 1 }
}
Step "go vet" { go vet ./... }
Step "Pruebas + cobertura" {
    # Comillas obligatorias: sin ellas PowerShell parte el argumento en el "."
    go test "-coverprofile=coverage.out" ./...
    if ($LASTEXITCODE -eq 0) { go tool cover "-func=coverage.out" | Select-Object -Last 1 }
}

if ($Race) {
    # En Windows -race necesita gcc. En el contenedor golang (Linux) ya viene.
    Step "Pruebas con -race (Docker)" {
        docker run --rm -v "${PWD}:/src" -w /src golang:1.27 go test -race ./...
    }
}

Write-Host "`nTodo OK" -ForegroundColor Green
