# RFC: Gooo bootstrap ceremony and root-of-trust boundary

Status: proposed bootstrap protocol  
Repository: `kimjooyoon/gooo-authority-bootstrap`  
Evaluator release binding: supplied by `immutable_evaluator_release_digest`

## Abstract

Gooo is a deterministic metaprogramming protocol for the self-improvement
root-of-trust problem. A candidate may propose an authority delta, but it may
not authorize its own ownership map, route, protection snapshot, or credential.
Missing core authority does not stop unrelated improvements: the evaluator
returns an explicit frontier and separately carries independently bound
activities that remain continuable.

The implementation is a bootstrap-ceremony evaluator and artifact generator.
It emits a proposal, receipt, semantic IR, generated evaluator source, machine
manifest, and human dossier. It has no API client for GitHub mutation and no
path that creates apps, secrets, branch protection, commits, pull requests,
merges, or releases in a target input repository.

## Pipeline

```text
.gooo source
    -> parser
    -> semantic IR with digest and one-to-one bindings
    -> generated Go evaluator artifact
    -> proposal / receipt / manifest
    -> human dossier
```

The `.gooo` grammar is line-oriented. The first non-comment line is `gooo 1`;
each subsequent line binds one field name to one JSON value. Required bindings
include the repository and base/head/PR tuple, protected paths, ownership
policy digest, immutable evaluator digest, branch-protection or ruleset
snapshot, credential authority identity, proposed delta, signer and approver
identities, timestamps, and observed artifacts.

The semantic IR sorts path and identity lists before calculating its digest.
Each protected-path cell has exactly one `observe-protected-path` activity. The
IR carries cell count, activity count, binding count, and an explicit
`one_to_one` boolean. There are no percentages, scores, inferred priorities, or
activity closure inferred from a green check or cache hit.

## Authority roles

Four roles are explicit:

1. Policy authority establishes the ownership and route policy.
2. Observation authority reports the protection snapshot and path set.
3. Execution authority represents the credential that could carry out an
   already-authorized operation.
4. Human decision authority makes the explicit approve decision.

Identity and artifact identifiers are compared across roles. Reuse of one actor
or artifact across different canonical roles is `REFUTED`, including silent
reuse through a credential, path observation, protection observation, signer,
or evaluator artifact. The candidate identity, branch, and edit digest are
checked against the proposed delta. Any authority-bearing change proposed by
the candidate is `REFUTED`.

## Decision semantics

The total order is `REFUTED > UNKNOWN > CLOSED`.

`REFUTED` is terminal for the submitted ceremony. It is returned for a stale
receipt whose repository/base/head/PR tuple differs, even when the branch name
matches; for self-authorization; for cross-role actor or artifact reuse; for a
complete claim over a known truncated observation; and for a regression that
tries to authorize a new identity or head.

The truncated-list rule preserves evidence. An observation reporting 3,000 of
3,044 paths is valid as a partial observation, but it is `REFUTED` as complete
path authority. Partial validity never supplies semantic closure.

`UNKNOWN` means that no contradiction was observed, but a frontier is missing.
Every unknown proposal has exactly these six `unknown_frontier` fields:

```text
credential_authority_identity
guardian_app_client_id
guardian_app_private_key
required_status_checks
independent_observation
human_decision
```

`blocked_by` contains only the minimal currently unmet frontier: both missing
credential fields are returned together when both are absent; downstream
observation and decision requirements are not added until the earlier frontier
is satisfied. A missing GitHub App client ID/private-key digest or a
`required_status_checks: null` snapshot is therefore `UNKNOWN`, never
`CLOSED`.

`CLOSED` requires all of the following:

- the current repository/base/head/PR tuple matches exactly;
- protection and path evidence is complete and independently observed;
- an explicit human/admin approve decision is bound to the tuple;
- the receipt is bound to the immutable evaluator release identity;
- replay evidence has exact tuple and receipt-digest equality;
- role identity and artifact bindings remain distinct.

## Münchhausen choice

The input declares one of three choices:

- `FOUNDATION`: a pre-declared out-of-band or threshold authority must be
  bound before the candidate. Its binding digest, identity, threshold, members,
  and timestamp are input data.
- `COHERENCE`: independently held authorities may validate consistency among
  themselves. Coherence cannot create an authority that is absent.
- `REGRESSION`: an exact previously valid baseline may be preserved. It cannot
  authorize a new identity or head, and tuple mismatch is refuted.

The choice is reported in the proposal and dossier; it is not inferred from
the presence of checks or a cached result.

## Case corpus

The static digest-bound corpus contains exactly eight cases: two `CLOSED`,
three `UNKNOWN`, and three `REFUTED`.

| Case | Expected result | Binding exercised |
| --- | --- | --- |
| valid Foundation | `CLOSED` | pre-declared threshold bound before candidate |
| missing Guardian credentials | `UNKNOWN` | client ID and private-key frontier |
| null required checks | `UNKNOWN` | `required_status_checks` frontier |
| stale #609 receipt | `REFUTED` | new head defeats branch-name reuse |
| candidate self-ownership | `REFUTED` | candidate cannot register its own map |
| REST truncation | `REFUTED` | 3,000/3,044 complete claim plus partial evidence |
| independent improvements | `UNKNOWN` | unrelated activities remain continuable |
| exact replay and human decision | `CLOSED` | exact receipt replay and explicit approval |

## Provenance and related primary specifications

The protocol uses no external research at runtime. The following primary
official specifications informed terminology and the receipt boundary:

- [The Update Framework specification](https://theupdateframework.github.io/specification/latest/)
  motivates threshold, delegated, and immutable metadata boundaries.
- [in-toto specification](https://in-toto.io/in-toto-spec/) motivates binding
  statements to a subject and authenticated step materials.
- [Sigstore documentation](https://docs.sigstore.dev/) motivates transparent
  signing and identity-bound provenance without treating a signature as a
  substitute for policy.
- [GitHub branch protection REST documentation](https://docs.github.com/en/rest/branches/branch-protection)
  and [GitHub repository rules documentation](https://docs.github.com/en/rest/repos/rules)
  motivate preserving the distinction between a snapshot observation and the
  authority to change the target.

## Product boundary and metrics

The product reports `repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`. Development lifecycle operations such as
creating this repository, pushing a branch, opening the implementation pull
request, and collecting GitHub Actions results are reported separately from
those product metrics.

The inventory records exact files, directories, Go physical lines, Gooo
physical lines, build wall, test wall, peak RSS, and executed/reused/skipped/
not-observed activity counts. When the caller does not provide an observed
runtime measurement, wall and RSS fields are `not-observed`; the evaluator does
not fabricate them.
