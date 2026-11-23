# Incident 2026-06 — the $7k DispatchWorker loop

Load this when you need to explain why a check exists or what a RED finding looked like in the original incident.

## What happened

A recursive loop in the `DispatchWorker` Lambda (a consumer re-publishing to its own SQS queue) cost ~$7k in June 2026. It was detected 6 days late, from the invoice, because there was no automatic cost alert.

Full diagnostic: `mu/docs/evidencia-spike-costos-junio-2026.md` (external, in the mu repo).

## The three signatures the script reproduces

1. **Daily cost spike** — Cost Explorer, last 8 days. RED if the latest day exceeds `SPIKE_FACTOR` × the prior median. The script also lists the top services of the day (where the spend lands: SSM, Lambda, DynamoDB…).
2. **Runaway SQS queue** — any queue with `NumberOfMessagesReceived > QUEUE_RX_HR` per hour. This is the self-sustaining loop signature; `DispatchWorker` was receiving ~417k messages/hour at the peak.
3. **Guardrails present** — are there AWS Budgets and Cost Anomaly Monitors? If **0**, RED: there is no automatic alert and the next spike will again be seen late.

## Purpose of the skill

A per-developer resource so a deploy or feature cannot silently inflate costs for days. The script is the manual reproduction of the signatures above; the real fix is the always-on prevention in `prevention.md`.
