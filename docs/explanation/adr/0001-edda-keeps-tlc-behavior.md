# ADR-0001: Edda keeps the default behavior of TLC

| Item | Value |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |

## Context

By default, TLC finds a deadlock: a state in which no step can occur. A specification can stop on purpose, for example a counter that stops at 2. TLC shows a failure for this specification, but its invariant is true.

Edda can start TLC with the option `-deadlock`. This option stops the deadlock check. Then TLC accepts the counter.

## Decision

Edda is a translator. It does not change the default behavior of TLC.

- Edda does not add TLC options that stop a check or change a check.
- The writer of the specification must satisfy TLC, as a writer of TLA+ must.
- If a system stops on purpose, the specification must say so with an explicit word. Add this word to the vocabulary only when a real specification needs it, and only after the Founder approves it.

## Consequences

- No surprise: `edda check` gives the same result as TLC on the generated `.tla` and `.cfg` files.
- An expert can examine the generated TLA+ without knowledge of Edda.
- A specification that stops on purpose shows a deadlock failure. To prevent it, the writer adds a step that can always occur, for example a `Reset` step. `README.md` gives this rule.
