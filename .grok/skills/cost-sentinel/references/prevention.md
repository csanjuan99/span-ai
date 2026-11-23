# Always-on prevention (deploy ONCE, do not rebuild)

Load this when check 3 (guardrails) is RED, or when setting up cost monitoring for an account.

Continuous detection is already provided by AWS natively. Do not build a custom monitor. Deploy once per account (CDK or console):

- **AWS Budgets** — monthly/daily budget per environment with alerts at 80/100/120% → SNS/email.
- **AWS Cost Anomaly Detection** — `CfnAnomalyMonitor` (dimension `SERVICE`) + `CfnAnomalySubscription`. ML-based, alerts within hours, catches spikes inside the normal range that a fixed budget misses.
- **Lambda recursive-loop detection** — account setting = `Terminate` (verify). Not sufficient alone (a re-enqueue with delay breaks lineage tracking), but it is free.
- **Invocation alarm** on high-risk SQS workers, plus `reservedConcurrentExecutions` as a blast-radius cap.

`scripts/cost-sentinel.sh` flags RED while Budgets/Anomaly monitors are missing; once deployed, it stops flagging check 3.

## Harness-specific: recurring runs (Claude Code)

These notes apply only to harnesses that provide `/loop` and `/goal`:

- **Recurring watch:** `/loop 30m correr cost-sentinel y avisar si hay ROJO` → runs the script every 30 minutes; the summary table is what you see each iteration.
- **Until resolved:** `/goal vigila costos con cost-sentinel hasta que el summary esté en VERDE` → self-paces and stops when the script exits `0`.
- In both, the agent executes `scripts/cost-sentinel.sh` (relative to the skill directory), reads the table and exit code, and reports only what is actionable.
