# Changelog

All notable changes to this project are in this file.
The format is [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- MVP translator in Go. `edda emit spec.hcl` writes `spec.tla` and `spec.cfg`. `edda check spec.hcl` runs TLC and shows a failure as steps of the HCL specification, with line references. The operator thus does not have to read TLA+.
- Static checks before TLC: unknown variables, `after` for a variable that does not exist, words that are not in the approved vocabulary, invariant names that are not sentences, and names that the TLA+ output uses two times. Edda shows each error at its HCL line.
- Tests: a golden test for the PROPOSAL.md section 6 specification, one test for each word of the version 0 vocabulary, one test for each static check, and a TLC test for the three steps of PROPOSAL.md section 9.
- `scripts/fetch-tla2tools.sh` downloads `tla2tools.jar` into `~/.cache/edda/` and prints its path. The script compares the SHA-256 of the file with the pin in `scripts/tla2tools.txt`. TLC thus has the same version on all computers. Homebrew has no TLA+ formula for Linux, so Edda needs this script.
- The pin is TLA+ `v1.8.0`. Version `v1.7.4` does not have the TLC option `-dumpTrace json`, which Edda uses to read traces.
