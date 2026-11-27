# Agent process lifecycle

Objective: no agent process outlives the run that started it. Branch `fix/agent-process-lifecycle`.
Delivery: ask-on-risk. Tests: `go test ./...`. Route: inline, one verifier at the end (Risk items 3 and 4).

## Specs

- S1: "`span stop`, `span pause`, `span rewind` and relaunch terminate every agent process the run started, including descendants: start each agent in its own process group (setpgid) and signal the group (SIGTERM, wait a grace period, then SIGKILL). Release the run lock only after every process exited."
- S2: "No agent process survives its run: on `span serve` start, reap or kill agent process groups recorded for runs that are not active."
- S3: "Record an event per termination (stage, agent, pid, signal, exit) in events.jsonl so the stop is auditable."
- S4: "Choosing an agent on relaunch applies to every agent-calling stage of that run (frame, build, inspect), or the CLI says explicitly which stages keep the default."
- S5 (acceptance): "Start a run, let the build agent start, run `span stop <run>`: `ps` shows no process from that run's agent tree within the grace period, and no process with ppid 1 from that agent remains." "Same for pause, rewind and relaunch."
- S6 (acceptance): "Relaunch with `claude`: `agents.json` lists claude for frame, build and inspect, and the next `agent.run` event of each stage reports claude."
- S7 (acceptance): "Tests cover a fake agent that ignores SIGTERM and spawns a child that ignores SIGTERM."

Assumptions: pause interrupts only a stage that is calling an agent (the stage is not completed and re-runs on resume); other stages keep holding at the next boundary. Relaunch gets a `span relaunch` command (UI/CLI parity). Windows has no process groups: only the started process is killed.

## Tasks

- T1 (S1, S3, S7) supervise the agent process tree; inline; done; commit b863b83
- T2 (S1, S3, S5) pause interrupts the agent; rewind stops a running run first; inline; done; commit b863b83
- T3 (S2) reap on serve start and on stop of an undriven run; inline; done; commit b863b83
- T4 (S1, S4, S6) relaunch stops, sets the agent for all stages; CLI command; docs; inline; done; commit b863b83 (docs b28bc72)

## Log

- L1 (2026-11-27): "Bug: stopping, pausing, rewinding or relaunching a run does not terminate the agent sessions that run spawned. Observed (runs 2026-11-27-1514-8e60 and 2026-11-27-1514-836f, `span serve` pid 6320): after a rewind + relaunch, `span serve` kept or started `cursor-agent ... -p --trust --force --model auto` children for the build stage. Each cursor-agent forks a `node .../index.js worker-server` child. One of them (pid 18197) was left orphaned with ppid 1 after its parent ended. SIGTERM on the cursor-agent process did not end it; it took SIGKILL on the parent and on every descendant. Relaunching a run with a different agent only changed the `frame` stage (`agents.json` = {\"frame\":\"claude\"}); `build` and `inspect` silently fell back to `span.yaml` `agent: cursor`." (Expected and acceptance quoted in Specs.)
- Evidence: `proc.Tame` already used setpgid but only signalled the group on context cancel, never after a normal exit. cursor spawns `worker-server` with `detached:false`.
- Evidence (2026-11-27): RED observed for tree supervision (agent tests did not build) and for pause mid-build (agent never stopped); GREEN: `go test ./...` all packages ok, `go vet` on darwin and GOOS=windows ok, web `vitest run` 827 passed, `tsc` clean.
- Decision: pause now interrupts an agent call (changes the former boundary-only behaviour, per S1). Rewind/relaunch stop a driven run instead of refusing; a driver that does not end is still a busy/conflict error.
- Limit: a child that moves to its own group AND loses its parent within 500 ms of starting is not found.
- Verifier (2026-11-27): S1, S3-S7 met; S2 partly. Found B1 (reaper in another TZ dropped the record), B2 (relaunch refused for --max-parallel still stopped the run), G1 (delete lost the record). Fixed, each with a test seen RED first; pinned TZ=UTC, check room before halting, reap on delete and on relaunch.
- Native review (2026-11-27): granted. Blockers R2-001 (running card still promised a stage-boundary pause) and R4-001 (log flood where processes cannot be listed) fixed; successor review-89377cf8039983a2 approved and acknowledged. Advisory only, left for later: conflict.go pause path, orphan.go record handling, serve.go relaunch ordering.
- Native review (2026-11-27) of the committed range (d1640c9..399cd78 trees): granted; review-56963460d46f7f72 approved and acknowledged, no blockers. Open warnings: agent record not written atomically (proc/orphan.go); USAGE.md describes a conflict-proposal pause the code does not do; relaunch saves the agent before halting; conflict pause and abort-without-HEAD behaviour undocumented; in-flight pause test does not assert the persisted cursor.
