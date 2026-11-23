# Harness lessons

Objective: the next run of a stage does not repeat a sealed mistake, without an extra model call and without rewriting the harness skills.
Branch: main
Delivery: single repo
Test runner: `go test ./internal/learn/ ./internal/prompt/ ./internal/engine/`

## Specs

S1. Lessons live in `<workspace>/.span/lessons.json`. A missing file is an empty book. A file that cannot be parsed does not stop the run and is not overwritten.

S2. Only `frame`, `build`, and `inspect` record a `hold` or a `stop`. An `advance` does not create a lesson.

S3. A reason is kept when it is non-empty, at most 160 characters, and does not start with `open questions:`.

S4. The same stage and reason update one lesson. The book holds at most 32 lessons. A new one-off does not evict a lesson that has already repeated or has a positive score.

S5. The next prompt for that stage includes at most 5 lessons, highest score first, then most seen. A lesson with `harmful >= 2` and `helpful == 0` is removed.

S6. When the stage advances, each lesson that was in that prompt gets `helpful + 1`. When the same reason is sealed again, that lesson gets `harmful + 1`.

S7. Writing or reading the book never changes the stage verdict.

## Tasks

- T1. S1–S7. Ledger, prompt line, and engine hook. Route: inline.

## Log

L1. Aether Flow's retro writes a growing lesson file with a second model call and does not change the run. Span keeps that boundary and drops the model call: the sealed reason is the lesson, lookup is a bounded file, and a lesson that fails twice leaves the prompt.
