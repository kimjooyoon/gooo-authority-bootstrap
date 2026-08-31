# Gooo Authority Bootstrap

Deterministic bootstrap-ceremony evaluation for a self-improvement root of
trust. The product emits a bootstrap proposal, receipt, machine artifacts, and
human dossier. It does not mutate a target repository and never creates apps,
secrets, branch protection, commits, pull requests, merges, or releases in a
target input repository.

## Bootstrap main

The bootstrap main was intentionally limited to these three paths:

```text
.gitignore
LICENSE
README.md
```

The initial commit SHA is recorded in the final lifecycle dossier after the
bootstrap commit is created. Substantive implementation is introduced in one
pull request from the implementation branch.

## Development contract

The evaluator consumes a `.gooo` ceremony source, lowers it into a semantic IR,
generates a Go evaluator, and emits deterministic machine artifacts plus a
human-readable dossier. It models policy, observation, execution, and human
decision authority as distinct roles. Results are ordered `REFUTED > UNKNOWN >
CLOSED`; no score, percentage, inferred priority, cache hit, or green check
alone can close a ceremony.

All local Go build, test, vet, format, compiler, harness, and conformance runs
are prohibited by process policy. GitHub Actions is the executable validation
environment. Development-only operations are reported separately from product
metrics.

## Run the evaluator

Use a caller-owned output directory:

```text
go run ./cmd/gooo-bootstrap \
  -source .gooo/authority.gooo \
  -output /caller-owned/output \
  -inventory-root .
```

The command emits `bootstrap-proposal.json`, `bootstrap-receipt.json`,
`semantic-ir.json`, `generated/evaluator.go`, `human-dossier.md`, and a
`manifest.json`. A `REFUTED` or `UNKNOWN` decision is a deterministic product
result; only malformed source or an unwritable caller-owned output is a command
error. See [the RFC](docs/RFC.md) and [the lifecycle record](docs/PROCESS.md)
for the protocol and release process.
