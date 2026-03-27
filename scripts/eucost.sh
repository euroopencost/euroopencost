#!/bin/bash
# eucost — terraform plan + Cloud-Kostenberechnung in einem Schritt
#
# Nutzung:
#   ./scripts/eucost.sh              # im aktuellen Verzeichnis
#   ./scripts/eucost.sh -o json      # JSON Output
#
# Oder global einbinden:
#   echo 'alias eucost="<pfad>/scripts/eucost.sh"' >> ~/.bashrc

set -e

OUTPUT_FORMAT="table"
TF_ARGS=()

# Argumente parsen: -o/--output für eucost, Rest an terraform weiterleiten
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        *)
            TF_ARGS+=("$1")
            shift
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EUCOST="${SCRIPT_DIR}/../eucost"

# Fallback: eucost aus PATH
if [ ! -f "$EUCOST" ]; then
    EUCOST="eucost"
fi

TMP_PLAN=".eucost-tmp.tfplan"

echo ">> terraform plan wird ausgefuehrt..."
terraform plan -out="$TMP_PLAN" "${TF_ARGS[@]}"

echo ""
echo ">> Kosten werden berechnet..."
echo ""

# terraform show -json in eucost plan - pipen
terraform show -json "$TMP_PLAN" | "$EUCOST" plan - -o "$OUTPUT_FORMAT"

rm -f "$TMP_PLAN"
