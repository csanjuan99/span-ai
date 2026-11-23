# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

A developer who launches a Jira story through Span and opens the Span page to move that run along: approve stages, resolve merge conflicts, review the result. They already live in a terminal and often run several stories in parallel, in one repository or across a folder of several repositories.

## Product Purpose

Span runs a story through gated stages (frame, build, prove, qa and more) using the person's own coding agent CLI, and keeps the human in control of each decision. The page (the Desk and the run page) is where that control happens. Success is a story that reaches a merged PR with the developer understanding every decision along the way.

## Positioning

Span uses the agent CLIs the developer already has (cursor, claude, codex, pi) with Gentle AI, rather than shipping its own agent. It gates each stage with evidence (tests, QA criteria, conflict choices) instead of trusting agent output.

## Operating Context

- Launch from a folder with several repositories, from a folder choosing one, or directly inside a repository. All three must work.
- Policy lives in `span.yaml` (agent, per-CLI models); the Desk writes it.
- Stories come from Jira; setting up the Jira account is a first-run step that is not guided today.
- Runs execute in parallel, each in its own worktree when the checkout is busy.

## Capabilities and Constraints

- Existing page: Desk, run page with stage rail, timeline, ledger, conflict panel, rewind, per-stage agent and model picker.
- Missing today and requested: a guided first-run / project-setup wizard, Jira account setup guidance, and a metrics dashboard.
- Dashboard must answer four questions: time and stages (duration per run and per stage, where runs stall); result quality (share reaching PR/merge, holds, QA failures, retries); cost and tokens (per run, agent, model, agents out of quota); agents and models (which performs best per stage and which fails or blocks).
- Open product fact: which of these metrics the engine already records is not yet confirmed. Cost and token figures in particular must not be invented; the dashboard shows only what runs actually recorded.
- Page copy is in English.

## Brand Commitments

The existing identity is binding: a terminal session, not a marketing dashboard (see DESIGN.md). New surfaces extend it.

## Evidence on Hand

Real run state in `.span/runs` and `agents.json` per project; no external benchmarks or customer claims exist, and none may be fabricated.

## Product Principles

- The human decides; the page makes each decision legible and cheap to make.
- Show what actually happened, from recorded run data, never an estimate presented as fact.
- A first-time developer reaches a working run without reading docs.
- Works the same whether launched from a folder of repositories or inside one repository.
- Dense and scannable over decorative: the user is mid-task.
