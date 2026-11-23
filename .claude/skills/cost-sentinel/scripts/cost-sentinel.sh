#!/usr/bin/env bash
# cost-sentinel — chequeo read-only post-deploy de spikes de costo y runaways en AWS.
#
# ponytail: esto es un spot-check on-demand, NO un daemon. La detección
#   siempre-encendida ya la da AWS nativo (Budgets + Cost Anomaly Detection,
#   desplegar una vez — ver SKILL.md). Este script reproduce las 3 firmas que
#   identificaron el incidente de $7k de junio-2026 para correrlo a mano tras un
#   deploy o feature.
#
# Uso:   ./cost-sentinel.sh --profile <p> [--region <r>] [--selftest]
# Env:   perfil: --profile > AWS_PROFILE_OVERRIDE > AWS_PROFILE > prompt (TTY)
#        AWS_REGION (def us-east-1)  SPIKE_FACTOR (def 1.5)  QUEUE_RX_HR (def 100000)
# Salida: exit 0 todo verde · exit 1 hay hallazgo ROJO (sirve para gate de CI)
#         exit 2 uso/credenciales inválidas (sin perfil o STS falló).
set -uo pipefail

PROFILE="${AWS_PROFILE_OVERRIDE:-${AWS_PROFILE:-}}"
REGION="${AWS_REGION:-us-east-1}"
SELFTEST=0
while [ $# -gt 0 ]; do
  case "$1" in
    --profile) PROFILE="${2:-}"; shift 2 ;;
    --profile=*) PROFILE="${1#*=}"; shift ;;
    --region) REGION="${2:-}"; shift 2 ;;
    --region=*) REGION="${1#*=}"; shift ;;
    --selftest) SELFTEST=1; shift ;;
    *) echo "uso: $0 --profile <p> [--region <r>] [--selftest]" >&2; exit 2 ;;
  esac
done
SPIKE_FACTOR="${SPIKE_FACTOR:-1.5}"     # día más reciente vs mediana previa
QUEUE_RX_HR="${QUEUE_RX_HR:-100000}"    # msgs/hora recibidos => sospecha de loop
RED=0
ST1="N/A"; ST2="N/A"; ST3="N/A"; D1="-"; D2="-"; D3="-"

_aws() { command aws --profile "$PROFILE" --region "$REGION" "$@"; }

# --- self-check del único algoritmo no trivial (ponytail: una verificación) ---
if [ "$SELFTEST" -eq 1 ]; then
  python3 - "$SPIKE_FACTOR" <<'PY'
import sys
f=float(sys.argv[1])
def is_spike(latest, prior):
    p=sorted(prior); med=p[len(p)//2]
    return med>0 and latest > med*f
assert is_spike(900,[100,110,108,112]) is True,  "debe detectar 900 vs ~110"
assert is_spike(120,[100,110,108,112]) is False, "120 no es spike a 1.5x"
assert is_spike(100,[0,0,0]) is False,           "mediana 0 => no spike (evita div)"
print("selftest ok")
PY
  exit $?
fi

# --- perfil: explícito, nunca asumido -----------------------------------------
if [ -z "$PROFILE" ]; then
  AVAIL="$(command aws configure list-profiles 2>/dev/null | tr '\n' ' ')"
  if [ -t 0 ]; then
    echo "Perfiles disponibles: ${AVAIL:-(ninguno)}"
    printf 'Profile to use: '; read -r PROFILE
  fi
  if [ -z "$PROFILE" ]; then
    echo "uso: $0 --profile <p> [--region <r>]  (perfiles: ${AVAIL:-ninguno})" >&2; exit 2
  fi
fi
ACCT="$(_aws sts get-caller-identity --query Account --output text 2>/dev/null)"
if [ -z "$ACCT" ]; then
  echo "credentials invalid for profile $PROFILE (aws sso login --profile $PROFILE)" >&2; exit 2
fi
TODAY="$(date -u +%Y-%m-%d)"; START="$(date -u -v-8d +%Y-%m-%d 2>/dev/null || date -u -d '8 days ago' +%Y-%m-%d)"
echo "== cost-sentinel · cuenta $ACCT · $REGION · $(date -u +%FT%TZ) =="

# --- 1. Spike de costo diario -------------------------------------------------
echo "-- 1) Costo diario (spike > ${SPIKE_FACTOR}x mediana) --"
_aws ce get-cost-and-usage --time-period Start="$START",End="$TODAY" --granularity DAILY \
  --metrics UnblendedCost --query "ResultsByTime[*].[TimePeriod.Start,Total.UnblendedCost.Amount]" \
  --output text 2>/dev/null | python3 - "$SPIKE_FACTOR" <<'PY'
import sys
f=float(sys.argv[1]); rows=[l.split() for l in sys.stdin if l.strip()]
vals=[(d,float(c)) for d,c in rows]
for d,c in vals: print(f"   {d}  ${c:,.2f}")
if len(vals)>=3:
    *prior,(ld,lc)=vals
    p=sorted(v for _,v in prior); med=p[len(p)//2]
    if med>0 and lc>med*f:
        print(f"   \033[31mROJO\033[0m: {ld} ${lc:,.0f} > {f}x mediana ${med:,.0f}"); sys.exit(7)
    print(f"   \033[32mOK\033[0m: último día ${lc:,.0f} ~ mediana ${med:,.0f}")
PY
case $? in
  7) RED=1; ST1="ROJO"; D1="spike > ${SPIKE_FACTOR}x mediana" ;;
  0) ST1="VERDE"; D1="sin spike" ;;
  *) ST1="N/A";  D1="Cost Explorer inaccesible"; echo "   (Cost Explorer no accesible)" ;;
esac

# servicios top del día más reciente completo (contexto del spike)
echo "   top servicios (ayer):"
Y="$(date -u -v-1d +%Y-%m-%d 2>/dev/null || date -u -d '1 day ago' +%Y-%m-%d)"
_aws ce get-cost-and-usage --time-period Start="$Y",End="$TODAY" --granularity DAILY --metrics UnblendedCost \
  --group-by Type=DIMENSION,Key=SERVICE --query "ResultsByTime[0].Groups[*].[Keys[0],Metrics.UnblendedCost.Amount]" \
  --output text 2>/dev/null | sort -t$'\t' -k2 -rn | head -5 | awk -F'\t' '{printf "     %-30s $%.2f\n",$1,$2}'

# --- 2. Colas SQS runaway (firma del loop: alta tasa de recepción) -----------
echo "-- 2) Colas SQS con recepción > ${QUEUE_RX_HR}/hora (firma runaway) --"
S=$(date -u -v-1H +%s 2>/dev/null || date -u -d '1 hour ago' +%s); E=$(date -u +%s)
ST=$(date -u -r "$S" +%FT%TZ 2>/dev/null || date -u -d "@$S" +%FT%TZ); ET=$(date -u -r "$E" +%FT%TZ 2>/dev/null || date -u -d "@$E" +%FT%TZ)
found=0
for url in $(_aws sqs list-queues --query "QueueUrls[]" --output text 2>/dev/null | tr '\t' '\n'); do
  q=$(basename "$url")
  rx=$(_aws cloudwatch get-metric-statistics --namespace AWS/SQS --metric-name NumberOfMessagesReceived \
       --dimensions Name=QueueName,Value="$q" --start-time "$ST" --end-time "$ET" --period 3600 \
       --statistics Sum --query "Datapoints[0].Sum" --output text 2>/dev/null)
  [ "$rx" = "None" ] || [ -z "$rx" ] && continue
  if python3 -c "import sys; sys.exit(0 if float('$rx')>$QUEUE_RX_HR else 1)" 2>/dev/null; then
    printf "   \033[31mROJO\033[0m: %-40s %.0f msgs/h\n" "$q" "$rx"; found=$((found+1)); RED=1
  fi
done
if [ "$found" -eq 0 ]; then ST2="VERDE"; D2="ninguna cola sobre el umbral"; echo "   \033[32mOK\033[0m: ninguna cola sobre el umbral"
else ST2="ROJO"; D2="$found cola(s) > ${QUEUE_RX_HR}/h"; fi

# --- 3. Guardrails nativos presentes? (la prevención real) -------------------
echo "-- 3) Guardrails de costo (AWS Budgets + Cost Anomaly Detection) --"
nb=$(_aws budgets describe-budgets --account-id "$ACCT" --query "length(Budgets)" --output text 2>/dev/null || echo 0)
nm=$(_aws ce get-anomaly-monitors --query "length(AnomalyMonitors)" --output text 2>/dev/null || echo 0)
if [ "${nb:-0}" -gt 0 ] 2>/dev/null; then echo "   \033[32mOK\033[0m: $nb budget(s)"; else echo "   \033[31mROJO\033[0m: 0 AWS Budgets — sin alerta de presupuesto"; RED=1; fi
if [ "${nm:-0}" -gt 0 ] 2>/dev/null; then echo "   \033[32mOK\033[0m: $nm anomaly monitor(s)"; else echo "   \033[31mROJO\033[0m: 0 Cost Anomaly Monitors — sin detección ML de spikes"; RED=1; fi
if [ "${nb:-0}" -gt 0 ] 2>/dev/null && [ "${nm:-0}" -gt 0 ] 2>/dev/null; then ST3="VERDE"; D3="budgets+anomaly OK"; else ST3="ROJO"; D3="faltan: ${nb:-0} budget(s), ${nm:-0} monitor(s)"; fi

# --- Tabla summary ------------------------------------------------------------
col() { case "$1" in VERDE) printf "\033[32m%-5s\033[0m" VERDE;; ROJO) printf "\033[31m%-5s\033[0m" ROJO;; *) printf "%-5s" "$1";; esac; }
echo ""
echo "== SUMMARY · cuenta $ACCT · $(date -u +%FT%TZ) =="
printf "  %-2s %-22s %-5s  %s\n" "#" "Chequeo" "Estado" "Detalle"
printf "  %-2s %-22s %s  %s\n" "1" "Spike de costo diario" "$(col "$ST1")" "$D1"
printf "  %-2s %-22s %s  %s\n" "2" "Cola SQS runaway"      "$(col "$ST2")" "$D2"
printf "  %-2s %-22s %s  %s\n" "3" "Guardrails (Budgets+Anomaly)" "$(col "$ST3")" "$D3"
echo "  ───"
echo "  VEREDICTO: $([ $RED -eq 0 ] && col VERDE || col ROJO)  (exit $RED)"
exit $RED
