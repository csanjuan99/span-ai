---
name: cost-sentinel
description: "Trigger: 'revisar costos', 'validar el deploy no disparó gasto', 'hay un runaway', 'cost-sentinel', 'check costs', after a prod/qa deploy or infra merge. Read-only AWS cost spike and runaway spot-check; exit code gates CI."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "2.0.0"
  category: "ops"
---

## Activation Contract

Load right after a deploy to prod/qa or after merging an infra feature, when someone suspects a runaway (Lambda loop, queue not draining, odd spend), or as a quick weekly check. Trigger phrases: "revisar costos", "validar el deploy no disparó gasto", "hay un runaway", "cost-sentinel", "check costs".

## Hard Rules

- Read-only. The script only queries STS, Cost Explorer, CloudWatch, SQS, Budgets, and Cost Anomaly Detection; never mutate infrastructure from this skill.
- This is an on-demand spot-check, never a daemon. Always-on detection is AWS Budgets + Cost Anomaly Detection (see `references/prevention.md`); never build a custom monitor.
- Never assume an AWS profile: use the one the user names; if none was named, ask which profile (one question) before running.
- Requires valid AWS credentials for that profile; if expired, run `aws sso login --profile <profile>` first.
- Exit `0` means all green; exit `1` means at least one RED finding; exit `2` means usage or credential error. Use `0`/`1` as a CI gate.
- Report only actionable findings; do not paste the full raw output.

## Decision Gates

| Finding | Action |
| --- | --- |
| Daily cost spike (check 1) | Inspect the top service of the day, cross with check 2 (queues) and with invocations of the suspected Lambda (`aws cloudwatch get-metric-statistics ... Invocations`). |
| Runaway SQS queue (check 2) | Find the consumer re-publishing to its own queue (`Sent ≈ Received`). Mitigate: lower `reservedConcurrentExecutions`; fix the re-enqueue. |
| Missing guardrails (check 3) | Deploy the native prevention in `references/prevention.md`. This is the root fix for "seen 6 days late". |
| Exit `2`, check 1 `N/A`, or check 3 RED right after a credential error | Verify profile/credentials/region (`aws sso login --profile <profile>`), rerun. |

Parameters:

| Variable | Default | Meaning |
| --- | --- | --- |
| `--profile <p>` / `AWS_PROFILE_OVERRIDE` / `AWS_PROFILE` | none, required | AWS CLI profile (flag wins; script prompts on a TTY, exits `2` otherwise) |
| `--region <r>` / `AWS_REGION` | `us-east-1` | Region for SQS/CloudWatch queries |
| `SPIKE_FACTOR` | `1.5` | Latest day vs prior median multiplier that flags RED |
| `QUEUE_RX_HR` | `100000` | Messages received per hour that flags a queue RED |

## Execution Steps

1. Resolve the skill directory (the directory containing this `SKILL.md`); all paths below are relative to it.
2. Determine the profile: use the one the user provided; otherwise ask (the harness may run `aws configure list-profiles` to offer choices). Do not proceed without one.
3. Optionally run `scripts/cost-sentinel.sh --selftest` to verify the spike algorithm (no profile needed).
4. Run `scripts/cost-sentinel.sh --profile <p> [--region <r>]`. It prints three checks, a summary table, and exits `0`/`1`; exit `2` is a usage or credential error: report it and stop.
5. Read the summary table and exit code (script prints VERDE/ROJO for GREEN/RED); apply Decision Gates to every RED row.
6. Load `references/incident-2026-06.md` only if you need the original incident signatures to explain a finding.
7. Return the Output Contract.

## Output Contract

- The 3-row summary (daily cost spike, runaway SQS queue, guardrails) with GREEN/RED/N/A per row.
- Overall verdict and exit code.
- For each RED row: the concrete next action from Decision Gates.
- If guardrails are missing: pointer to `references/prevention.md`.

## References

- `scripts/cost-sentinel.sh` — the read-only checker; run it, do not rewrite it.
- `references/incident-2026-06.md` — the $7k DispatchWorker incident and the three signatures the script reproduces; load when explaining a finding.
- `references/prevention.md` — always-on AWS guardrails to deploy once, plus harness-specific recurring-run notes; load when check 3 is RED or setting up monitoring.
