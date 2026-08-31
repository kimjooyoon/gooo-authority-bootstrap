# Repository lifecycle and evidence ledger

This repository separates the product boundary from development operations.
The evaluator's product metrics are always `repository_writes=0`,
`local_test_executions=0`, and `cross_project_required_gates=0`. GitHub
repository administration and Actions are lifecycle evidence, not evaluator
authority.

## Required lifecycle

1. Bootstrap `main` contains exactly `.gitignore`, `LICENSE`, and `README.md`.
   Record the bootstrap commit SHA and all three paths.
2. Put all substantive implementation in one implementation pull request.
3. The required Action blocks on gofmt, build, test, vet, conformance, replay,
   and artifact audit.
4. Merge only after every required check is green.
5. Run post-main Actions from the merged commit and upload generated artifacts
   to a caller-owned temporary directory.
6. Enable immutable releases before publishing immutable `v0.1.0`.
7. Preserve failed or mutable releases as historical evidence; never mutate a
   published artifact. Use a patch release for corrections.

## Evidence record

The final lifecycle record must contain exact values, not summaries:

- repository URL and visibility;
- bootstrap SHA and path inventory;
- implementation branch SHA, pull request number, merge SHA, and post-main
  SHA;
- each required and post-main Action run ID, job name, conclusion, and URL;
- artifact names, paths, byte counts, physical lines, and SHA-256 digests;
- tag, release ID, release URL, release assets, and asset digests;
- case, cell, and activity denominator counts;
- files, directories, Go physical lines, Gooo physical lines, build wall, test
  wall, peak RSS, and executed/reused/skipped/not-observed values;
- deviations from this process, including every failed attempt.

No local Go build, test, vet, format, fix, compiler, harness, or conformance
execution is allowed. Failed GitHub Actions attempts remain visible in the
run history and are not deleted or rewritten.
