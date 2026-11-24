# naughtian-edda

Edda translates a specification in HCL into TLA+, and checks it with TLC.
You do not have to read TLA+. Edda shows each failure as steps of your HCL file.

Refer to [PROPOSAL.md](docs/explanation/PROPOSAL.md) for the design.

## Install

You must have Go and Java.

```sh
make install
```

This command installs `edda` into `$(go env GOPATH)/bin` and TLC into `~/.cache/edda/`.
Make sure that `$(go env GOPATH)/bin` is in your `PATH`.

## Use

Write a specification, for example `counter.hcl`. The file name must be a TLA+ name: use letters, digits and `_` only.

```hcl
state {
  count = 0
}

step "Increment" {
  when  = count < 2
  after = { count = count + 1 }
}

step "Reset" {
  when  = count > 1
  after = { count = 0 }
}

invariant "the count is not more than 2" {
  check = count <= 2
}
```

| Command | Function |
|---|---|
| `edda check counter.hcl` | Do the static checks, run TLC, and show the result. |
| `edda emit counter.hcl` | Write `counter.tla` and `counter.cfg` next to `counter.hcl`. |

If TLC finds a failure, Edda shows the steps to the failure:

```
Step 1 — Write       (log_bug.hcl:7):  buffer [] → ["msg"]
Step 2 — Acknowledge (log_bug.hcl:18): acked 0 → 1
FAILURE: "an acknowledged record is durable" (log_bug.hcl:27): acked = 1, log = []
```

## Rules for a correct specification

- Each invariant must be true in all states.
- In each state, one or more steps must be able to occur. If no step can occur, TLC shows a deadlock.
- The number of states must be finite. Use `constraint` to set a limit, for example `check = length(log) <= 3`.

## Vocabulary (version 0)

Edda accepts only these words. Each word has one TLA+ equivalent.

### Blocks

| Word | Meaning | TLA+ |
|---|---|---|
| `state { x = v }` | The variables and their initial values. Use one `state` block only. | `VARIABLES x`, `Init` |
| `step "X" { ... }` | One event that the system can do. | `X == ...`, part of `Next` |
| `when = e` | The step can occur only if `e` is true before the step. | `/\ e` |
| `after = { x = e }` | The value of `x` after the step. Names on the right side refer to the values before the step. | `/\ x' = e` |
| `invariant "sentence" { check = e }` | A rule that must be true in all states. The name is a sentence. | `INVARIANT` |
| `constraint { check = e }` | A limit on the states that TLC examines. Use one `constraint` block only. | `CONSTRAINT` |

### Values, operators and functions

| Word | Meaning | TLA+ | Example |
|---|---|---|---|
| `0`, `1`, ... | A number that is not negative. | `0`, `1`, ... | `acked = 0` |
| `"text"` | A string. It cannot contain `"` or `\`. | `"text"` | `"msg"` |
| `[a, b]` | A sequence. | `<<a, b>>` | `log = []` |
| `( )` | Parentheses. | `( )` | `(acked + 1) <= 3` |
| `+` | The sum of two numbers. | `+` | `acked + 1` |
| `<` | The left number is less than the right number. | `<` | `count < 2` |
| `>` | The left number is more than the right number. | `>` | `length(log) > acked` |
| `<=` | The left number is not more than the right number. | `<=` | `acked <= length(log)` |
| `>=` | The left number is not less than the right number. | `>=` | `length(log) >= acked` |
| `length(s)` | The number of elements in the sequence `s`. | `Len(s)` | `length(log)` |
| `append(s, v)` | The sequence `s` with `v` added at the end. | `Append(s, v)` | `append(buffer, "msg")` |
| `concat(a, b)` | The sequence `a` followed by the sequence `b`. | `a \o b` | `concat(log, buffer)` |

## Development

| Command | Function |
|---|---|
| `make build` | Build `edda` into `bin/`. |
| `make test` | Run the tests. The TLC test runs only if TLC is installed. |
