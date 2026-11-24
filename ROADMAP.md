# Roadmap

## Phase 0: MVP (v0.1.0)

- Translator for language version 0. Source: [PROPOSAL.md](docs/explanation/PROPOSAL.md) sections 6 to 9.
  Edda gives the commands `edda check` and `edda emit`. Each word in the vocabulary has a golden test.
  Goal: Edda completes the three steps in PROPOSAL.md section 9.

## Phase 1: Distribution (v0.2.0)

Start this phase after Phase 0 is complete.

- Homebrew formula in the Naughtian tap. One `brew install` must install Edda and TLC.
  - The formula builds the `edda` binary from the source.
  - The formula installs `tla2tools.jar` as a resource. Use the version and SHA-256 in `scripts/tla2tools.txt`.
  - The formula depends on `openjdk`, because TLC needs Java.
  - Edda must find the jar from the formula without the variable `EDDA_TLA2TOOLS`.
  Goal: On a clean macOS and on a clean Linux, `brew install` followed by `edda check` on the PROPOSAL.md section 6 specification gives the correct result.
