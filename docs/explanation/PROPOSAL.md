# Naughtian Edda: Declarative configuration languages to TLA+ specification translator - design note
 
| Item | Value |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Codename | Naughtian Edda |
| Writing standard | ASD-STE100 Simplified Technical English |
 
## 1. Name
 
Snorri Sturluson wrote the Prose Edda in approximately 1220. He wrote it as a handbook for young poets. The old skaldic poems were too difficult for them to read. The handbook did not replace the old poems. It helped people to read them.
 
Naughtian Edda does the same work for TLA+. It does not replace TLA+. It helps operators to use TLA+.
 
Use the repository name `naughtian-edda`. Netflix had a different project with the name "Edda". That project is stopped. The prefix prevents confusion.
 
## 2. Problem
 
We must prove that a distributed system operates correctly. For example, a log system must obey these rules:
 
- An acknowledged record must stay durable.
- The system must not lose a record.
- The system must not write a record two times.
Model checking can find errors in a design before the code exists. TLA+ is the industry standard for this work. But TLA+ uses mathematical notation. Many operators cannot read this notation easily. The notation stops them. The logic does not stop them.
 
Mathematics and TLA+ have the same problem as skaldic poetry. The reader must have expert background knowledge to understand the text. If the expert community stops, the text becomes unreadable.
 
## 3. Objective
 
1. The operator writes a specification in a configuration language that he can read easily.
2. Edda translates the specification word for word into TLA+.
3. The TLA+ tools (TLC, Apalache, TLAPS) check the TLA+ output.
4. Edda shows the result in the language of the specification.
The operator does not have to read TLA+.
 
## 4. Design principles
 
### 4.1 TLA+ is the intermediate representation
 
TLA+ is an intermediate representation (IR). It is not the source language. The Rust compiler uses LLVM IR in the same way. The user does not have to read the IR. But the user can read the IR at all times.
 
### 4.2 Declarative to declarative
 
The source language is declarative. TLA+ is declarative. The translation keeps this property. The source language has no control flow. A step has no sequence of lines.
 
### 4.3 Word for word and deterministic
 
Each source word has exactly one TLA+ equivalent. The same input always gives the same output, byte for byte. A deterministic program does the translation. An LLM does not do the translation.
 
### 4.4 Errors in the source language
 
Edda shows all errors in the source language. Edda translates SANY errors and TLC traces back to the related HCL block.
 
### 4.5 Small vocabulary
 
Add a word to the language only when a real specification needs it. Each word must have an entry in the approved vocabulary (section 6.2) before Edda accepts it.
 
### 4.6 One step is one system event
 
Each step is a meaningful system event. Examples: write, fsync, acknowledge, crash, leader election. A step is not a line of code. A specification tells what must be true. It does not tell how to do the work.
 
### 4.7 Explicit text first
 
A reader must not have to recite a specification many times to understand it.
 
1. The name of an invariant is a sentence in human language. The reader understands the rule before he reads the formula.
2. Read the formula aloud from left to right. The result must be the same sentence. If the reader must reverse or decode the formula, write it again.
3. Each symbol must have a reason. If a symbol does not help the reader, remove it.
Example. The sentence is: "Do not acknowledge a record before all nodes complete fsync."
 
```hcl
invariant "no acknowledgement before all nodes complete fsync" {
  check = all(n in nodes, acked <= length(durable[n]))
}
```
 
Read the `check` line: "For all nodes, the number of acknowledged records is not more than the number of durable records on that node."
 
The equivalent TLA+ is: `\A n \in Nodes : acked <= Len(durable[n])`
 
### 4.8 A step describes two states
 
A step describes the state before the step and the state after the step.
 
- The `when` block is a condition on the state before the step.
- The `after` block gives the values after the step.
- All names on the right side of `after` refer to the values before the step.
- The lines in `after` have no sequence. There is no "value set earlier in the same step".
Edda must show an error if a specification uses a value as if an earlier line in the same step changed it.
 
Note: The keyword `after` replaces the keyword `set` from the first draft. The word `set` made the reader think of an imperative assignment.
 
### 4.9 Limit of the tool
 
The writer must still build the state machine in his mind. This is the design work. Edda removes the load of the notation. It does not remove the design work.
 
### 4.10 Relation to ASD-STE100
 
ASD-STE100 started in the European aerospace industry for aircraft maintenance documents. The readers are maintenance technicians in many countries. Many of them do not speak English as a first language. They read the documents under pressure. If a technician does not understand a sentence correctly, the result can be an accident. Thus, the industry did not select the most expressive English. It selected an English that is difficult to understand incorrectly.
 
Edda uses the same principle for logic:
 
| ASD-STE100 | Edda |
|---|---|
| A dictionary of approved words. Each word has one meaning. | An approved vocabulary. Each word has one TLA+ equivalent. |
| A limit on sentence length. One instruction in each sentence. | One `step` for each event. One sentence for each invariant. |
| Written for the technician, not for the design engineer. | Written for the operator, not for the notation expert. |
| An incorrect reading can cause an aircraft accident. | An incorrect reading can cause the loss of an acknowledged record. |
 
The maintenance technician is the operator of the aviation industry. ASD-STE100 shows that a controlled language can operate at the scale of an industry where errors kill people.
 
## 5. Architecture
 
```
HCL (default) ──┐
TOML (possible) ┴─► Spec AST ─► static checks ─► .tla + .cfg ─► TLC / Apalache
                                                                    │
           traces and errors as text, with references to HCL lines ◄┘
```
 
### 5.1 Frontend
 
Version 0 uses HCL only. Reasons:
 
- HCL has a real expression syntax. Expressions are not in strings.
- The HCL parser gives the position of an error.
- Operators know HCL from Terraform.
- The Go module `github.com/hashicorp/hcl/v2` parses HCL. It is the reference parser, and Terraform uses it.
TOML is possible in a later version. YAML is rejected, because YAML guesses data types.
 
### 5.2 Spec AST
 
The Spec AST is one internal model for all frontends. A new frontend needs only a new parser.
 
### 5.3 Static checks
 
Edda does these checks before it starts TLC:
 
- It finds incorrect variable names.
- It finds `after` entries for variables that do not exist.
- It finds invariants that use variables that are not declared.
- It finds words that are not in the approved vocabulary (section 6.2).
### 5.4 Backends
 
Version 0 uses TLC. TLC examines all states. Later versions can use Apalache (symbolic checks) and TLAPS (proofs).
 
### 5.5 Reverse translation
 
Edda shows a TLC trace as a sequence of steps in text. Each step refers to its HCL block.
 
## 6. Language version 0
 
```hcl
state {
  log    = []
  buffer = []
  acked  = 0
}
 
step "Write" {
  after = { buffer = append(buffer, "msg") }
}
 
step "Fsync" {
  after = {
    log    = concat(log, buffer)
    buffer = []
  }
}
 
step "Acknowledge" {
  when  = length(log) > acked
  after = { acked = acked + 1 }
}
 
step "Crash" {
  after = { buffer = [] }
}
 
invariant "an acknowledged record is durable" {
  check = acked <= length(log)
}
 
constraint {
  check = length(log) + length(buffer) <= 3
}
```
 
### 6.1 Translation table
 
| HCL | TLA+ |
|---|---|
| `state { x = v }` | `VARIABLES x` and `Init == x = v` |
| `step "X" { ... }` | `X == ...` |
| `when = e` | `/\ e` |
| `after = { x = e }` | `/\ x' = e` |
| `after = { x = one_of(S) }` | `/\ x' \in S` (nondeterministic) |
| variable not in `after` | `UNCHANGED <<...>>` (Edda makes this automatically) |
| `for_each = S` in a step, with the name `each` | `\E each \in S : ...` |
| `all(n in S, e)` | `\A n \in S : e` |
| `any(n in S, e)` | `\E n \in S : e` |
| `&&` / `\|\|` / `!` | `/\` / `\/` / `~` |
| `length` / `append` / `concat` | `Len` / `Append` / `\o` |
| all `step` blocks | `Next == X \/ Y \/ ...` and `Spec == Init /\ [][Next]_vars` |
| `invariant` | a definition, and `INVARIANT` in `.cfg` |
| `constraint` | `CONSTRAINT` in `.cfg` |
 
Note on `for_each`: The step exists for each element. Any element can do the next step. This is the meaning of `\E` in TLA+.
 
### 6.2 Approved vocabulary
 
This vocabulary uses the format of the ASD-STE100 dictionary. Each word has:
 
- One meaning only.
- One TLA+ equivalent only.
- One approved example.
- One example that is not approved, with the reason.
Section 6.1 is a short summary of this vocabulary. If the two sections do not agree, this section is correct.
 
Each entry must have a golden test. The test translates the approved example and compares the output with the expected TLA+. The test also makes sure that Edda shows an error for the example that is not approved, if Edda can find that error.
 
#### Blocks
 
| Word | Meaning | TLA+ | Approved | Not approved |
|---|---|---|---|---|
| `state` | The variables of the system and their initial values. | `VARIABLES`, `Init` | `state { acked = 0 }` | Two `state` blocks in one file. There is only one initial state. |
| `step` | One event that the system can do. | `Name == ...`, part of `Next` | `step "Fsync" { ... }` | `step "UpdateBufferThenLog"`. One step is one event, not a procedure. |
| `when` | The condition on the state before the step. The step can occur only if the condition is true. | `/\ e` | `when = length(log) > acked` | `when = acked' > 0`. A condition cannot refer to the state after the step. |
| `after` | The values of the variables after the step. All names on the right side refer to the state before the step. The lines have no sequence. | `/\ x' = e` | `after = { log = concat(log, buffer), buffer = [] }` | Use of `after` as a sequence of assignments, with the expectation that `log` gets the empty `buffer`. |
| `invariant` | A rule that must be true in all reachable states. The name is a sentence in human language. | definition and `INVARIANT` | `invariant "an acknowledged record is durable" { ... }` | `invariant "Inv1" { ... }`. The name must be a sentence. |
| `constraint` | A limit on the states that TLC examines. It is not a rule of the system. | `CONSTRAINT` | `check = length(log) + length(buffer) <= 3` | Use of `constraint` to hide a failure of an invariant. |
 
#### Values and quantifiers
 
| Word | Meaning | TLA+ | Approved | Not approved |
|---|---|---|---|---|
| `one_of(S)` | Any one element of the set `S`. TLC examines all elements. | `x' \in S` | `after = { leader = one_of(replicas) }` | `when = one_of(replicas)`. `one_of` gives a value for the state after the step only. |
| `for_each` | The step exists for each element of the set. Any element can do the next step. | `\E each \in S : ...` | `for_each = nodes` | Use of `for_each` as a loop that does the step for all elements one after the other. |
| `each` | The element that does the step, inside a step with `for_each`. | the bound name | `after = { durable = ... durable[each] ... }` | Use of `each` outside a step with `for_each`. |
| `all(n in S, e)` | The condition `e` is true for all elements `n` of `S`. | `\A n \in S : e` | `all(n in nodes, acked <= length(durable[n]))` | Nested `all` without a sentence that explains it. |
| `any(n in S, e)` | The condition `e` is true for one or more elements `n` of `S`. | `\E n \in S : e` | `any(n in nodes, n == leader)` | Use of `any` to select a value. Use `one_of` for that. |
 
#### Operators and functions
 
| Word | Meaning | TLA+ | Approved | Not approved |
|---|---|---|---|---|
| `&&` | Both conditions are true. | `/\` | `when = a && b` | — |
| `\|\|` | One or both conditions are true. | `\/` | `when = a \|\| b` | — |
| `!` | The condition is not true. | `~` | `when = !crashed` | — |
| `length(s)` | The number of elements in the sequence `s`. | `Len(s)` | `length(log)` | `length(nodes)` on a set. Use a set function for sets. |
| `append(s, v)` | The sequence `s` with the value `v` added at the end. | `Append(s, v)` | `append(buffer, "msg")` | Use of `append` with the expectation that it changes `s`. It gives a new value. |
| `concat(a, b)` | The sequence `a` followed by the sequence `b`. | `a \o b` | `concat(log, buffer)` | — |
 
## 7. Requirements for the TLA+ output
 
The TLA+ output must be correct without the source file. Experts must be able to examine it. A position paper must be able to include it.
 
1. Keep the names. `step "Acknowledge"` becomes `Acknowledge ==`.
2. Put a reference comment before each action. Example: `\* spec.hcl:18 step "Acknowledge"`.
3. One HCL block gives one TLA+ block. Keep the sequence of the blocks. Do not merge blocks. Do not optimize.
4. Use a deterministic format. The output must be easy to compare with `git diff`.
## 8. Command-line interface
 
| Command | Function |
|---|---|
| `edda check spec.hcl` | Do the static checks. Make the IR. Start TLC. Show the result, or show the trace as text. |
| `edda emit spec.hcl` | Write the `.tla` file and the `.cfg` file. |
 
Example of the trace output:
 
```
Step 1 — Write       (spec.hcl:7):  buffer [] → ["msg"]
Step 2 — Acknowledge (spec.hcl:18): acked 0 → 1
FAILURE: "an acknowledged record is durable" (spec.hcl:27): acked = 1, log = []
```
 
## 9. Minimum viable product
 
Use the specification in section 6 (one node, with crash). Do these steps:
 
1. Run `edda check` on the correct specification. Make sure that Edda finds no failure.
2. Change the `Acknowledge` step. Use `when = length(log) + length(buffer) > acked`. With this change, the system acknowledges before fsync.
3. Run `edda check` again. Make sure that Edda shows a trace like the example in section 8.
If Edda completes these three steps, the project has a reason to continue. Add `for_each`, `one_of` and quantifiers only when a real specification needs them.
 
## 10. Risks and limits
 
### 10.1 The lesson of Snorri
 
Snorri did not only copy the old myths. He arranged them, he made them logical, and in some places he added to them. His book then became the standard text. The handbook changed the knowledge that it kept.
 
Edda can make TLA+ easier to read. Edda must not change the meaning of TLA+. To prevent this:
 
- The IR must always be readable.
- Each word in the approved vocabulary must have a golden test.
### 10.2 Java runtime
 
TLC and Apalache use the Java virtual machine. They operate only during a check. They are not permanent services. Thus, they do not break the First Law. But the dependency must be in the documentation.
 
### 10.3 Parser
 
Edda is written in Go and uses `github.com/hashicorp/hcl/v2`. Its package `hclsyntax` gives a full expression AST and the source position of each expression. Speed is not a reason for the selection, because TLC uses almost all of the time of a check.
 
### 10.4 Trust in the translator
 
An error in the translator makes the proof useless. Keep the translator small. Use golden tests (HCL input, expected TLA+ output). Keep the IR readable.
 
### 10.5 Growth of the vocabulary
 
If a word needs a translation that changes with the context, the design is incorrect. If a new word cannot have an entry with one meaning in section 6.2, do not add it.
 
### 10.6 The Lamport trap
 
It is easy to write a specification as a program. Each `step` must be a system event, not a line of code.
 
### 10.7 Properties that TLA+ cannot state directly
 
- "Same input, same output" compares two executions. This is a hyperproperty. One TLA+ specification describes one execution at a time. Use golden tests, or use self-composition (two copies of the specification in parallel).
- "The IR has the same meaning as the source" is not a property that TLC can check. Use golden tests. Use TLAPS for a proof if necessary.
### 10.8 Related tools
 
- PlusCal: pseudocode that translates to TLA+.
- FizzBee: Python-like syntax, with its own model checker.
- Quint: TypeScript-like syntax, with the same logic as TLA+.
Differences of Edda: the frontend is a configuration language that operators know, standard TLA+ is the IR, the vocabulary is controlled in the style of ASD-STE100, and traces come back as text.
 
## 11. Scope
 
- Edda is a standalone project.
- Edda supports specifications of distributed systems. Examples: one-node log, replication, fencing.
- Edda can be a thesis subject for the formal verification course of the master's degree.
## 12. Open items
 
1. Decide on the counterexample discipline. Proposal: each invariant has three layers. Layer 1 is the sentence. Layer 2 is one counterexample story, kept in the repository. Layer 3 is the formula and the TLA+. Reason: if a generation loses the notation, layers 1 and 2 let the next generation build layer 3 again.
2. Select the method for determinism: golden tests or self-composition (see 10.7).
3. Add set functions to the vocabulary (for example, the number of elements in a set) when a replication specification needs them.