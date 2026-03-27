# eucost.ps1 — terraform plan + Cloud-Kostenberechnung in einem Schritt
#
# Nutzung (PowerShell):
#   .\scripts\eucost.ps1                    # Table Output
#   .\scripts\eucost.ps1 -Output json       # JSON Output
#   .\scripts\eucost.ps1 -- -var "env=prod" # Extra terraform Argumente
#
# Tipp: Als Alias einrichten:
#   Set-Alias eucost "C:\pfad\zu\scripts\eucost.ps1"

param(
    [string]$Output = "table",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$TerraformArgs
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Eucost = Join-Path $ScriptDir "..\eucost.exe"

# Fallback: eucost.exe aus PATH
if (-not (Test-Path $Eucost)) {
    $Eucost = "eucost.exe"
}

$TmpPlan = ".eucost-tmp.tfplan"

Write-Host ">> terraform plan wird ausgefuehrt..." -ForegroundColor Cyan

try {
    # terraform plan ausführen
    & terraform plan -out=$TmpPlan @TerraformArgs
    if ($LASTEXITCODE -ne 0) {
        Write-Error "terraform plan fehlgeschlagen (Exit Code: $LASTEXITCODE)"
        exit $LASTEXITCODE
    }

    Write-Host ""
    Write-Host ">> Kosten werden berechnet..." -ForegroundColor Cyan
    Write-Host ""

    # terraform show -json in eucost pipen
    $PlanJson = & terraform show -json $TmpPlan
    if ($LASTEXITCODE -ne 0) {
        Write-Error "terraform show fehlgeschlagen (Exit Code: $LASTEXITCODE)"
        exit $LASTEXITCODE
    }

    $PlanJson | & $Eucost plan - -o $Output

} finally {
    # Temp-Datei immer löschen
    if (Test-Path $TmpPlan) {
        Remove-Item $TmpPlan -Force
    }
}
