# Signals

Signals surfaces significant architectural changes in a Git repository with minimal noise.

It reads recent commits, groups them into change sets, matches changed paths against an explicitly
declared architecture, and reports the change sets that deserve architect attention, each with the
signals and evidence that caused it.

Signals is a proof of concept. It follows the principles in `dev/principles/README.md`:
signal over completeness, evidence over inference, explicit architecture over discovered
architecture, and minimum sufficient analysis. It analyzes commit metadata and paths only: no source
code semantics, no dependency graphs, no AI.

## Build and Usage

Requirements: Go 1.27 or later and `git` on `PATH`.

```sh
go build -o bin/signals .
bin/signals scan --repo . --architecture architecture.yaml --since 7d
```

Without building:

```sh
go run main.go scan --repo . --architecture architecture.yaml --since 7d
task run -- scan --repo . --architecture architecture.yaml --since 7d
```

| Flag             | Required | Meaning                                                                    |
| ---------------- | -------- | -------------------------------------------------------------------------- |
| `--repo`         | yes      | Path inside a local Git repository.                                        |
| `--architecture` | yes      | Architecture YAML file, see `architecture.yaml`.                           |
| `--since`        | yes      | Length of the scan period ending now: whole days (`7d`) or a Go duration (`36h`). |
| `--format`       | no       | `text` (default) or `json`.                                                |

Text output:

```text
HIGH  PAY-1421

Commits:
  f686307  2026-10-21  Jane Doe  PAY-1421 add refunds table
  1404a3f  2026-10-20  Jane Doe  PAY-1421 expose refund status in customer profile

Signals:
  CROSS_CONTEXT_CHANGE          bounded contexts changed together: commerce/customer, commerce/payments
  PUBLIC_CONTRACT_CHANGED       public-contract files changed: 1
  PERSISTENCE_SCHEMA_CHANGED    persistence-schema files changed: 1

Evidence:
  CROSS_CONTEXT_CHANGE
    f686307 payments/migrations/002_refunds.sql added (in commerce/payments)
    1404a3f customer/profile/profile.go modified (in commerce/customer/customer-profile)
    1404a3f payments/api/openapi.yaml modified (in commerce/payments/payments-api)
  PUBLIC_CONTRACT_CHANGED
    1404a3f payments/api/openapi.yaml modified (matches public-contract pattern "**/openapi.yaml")
  PERSISTENCE_SCHEMA_CHANGED
    f686307 payments/migrations/002_refunds.sql added (matches persistence-schema pattern "**/migrations/**")


MEDIUM  d54b81ec24ee

Commits:
  d54b81e  2026-10-22  Jane Doe  add fraud detection rules

Signals:
  COMPONENT_CREATED_OR_REMOVED  component commerce/payments/fraud-detection created

Evidence:
  COMPONENT_CREATED_OR_REMOVED
    d54b81e payments/fraud added (declared path of component commerce/payments/fraud-detection)


Repository commerce: 3 change sets since 2026-10-16T12:00:00Z.
2 attention candidates. 1 change sets without architectural signal: impact unknown, not assessed as safe.
```

`--format json` prints the same report as one JSON object: `repository`, `since`, `changeSets`,
`withoutSignal`, and `candidates`. Each candidate holds `level`, `changeSet`, `commits`, and
`signals`; each signal holds `type`, `changeSet`, `summary`, and `evidence` with `commit`, `path`,
`status`, and `reason`.

"Without architectural signal" means the architectural impact is unknown, not that the change is safe.

## Architecture File

`architecture.yaml` is a complete, commented example. All patterns and paths are slash-separated and
relative to the repository root. In patterns, `**` matches zero or more directories; other segments
use Go `path.Match` syntax.

- **Systems and contexts.** A changed path belongs to the context whose `paths` match it. When no
  context path matches, it belongs to the context listing the repository in `repositories`. The
  repository name is the base name of the repository's top-level directory. A path that belongs to
  two contexts at the same precedence is an ambiguous declaration and fails the scan.
- **Components.** A component declares one directory `path` and owns everything below it.
  Components of a context must not nest.
- **Artifacts.** Each of `public-contract`, `persistence-schema`, `deployment`, and `security` must be
  declared with a list of patterns. An empty list (`[]`) detects nothing. Unknown kinds and unknown
  fields are rejected.

## Signals

Each rule produces at most one signal per change set. Every signal lists all changes that satisfied
it as evidence.

| Signal                         | Raised when the change set                                       |
| ------------------------------ | ---------------------------------------------------------------- |
| `CROSS_CONTEXT_CHANGE`         | changes files in two or more bounded contexts.                   |
| `PUBLIC_CONTRACT_CHANGED`      | adds, modifies, or deletes a `public-contract` artifact.         |
| `PERSISTENCE_SCHEMA_CHANGED`   | adds, modifies, or deletes a `persistence-schema` artifact.      |
| `DEPLOYMENT_CHANGED`           | adds, modifies, or deletes a `deployment` artifact.              |
| `SECURITY_CHANGED`             | adds, modifies, or deletes a `security` artifact.                |
| `COMPONENT_CREATED_OR_REMOVED` | creates or removes the directory declared as a component `path`. |

Signals are facts, not judgments: a signal never claims an architectural violation.

## Attention

Change sets without signals are suppressed. Every other change set becomes an attention candidate:

- `HIGH`: two or more distinct signal types.
- `MEDIUM`: one signal type.

Candidates are ordered HIGH first, then by newest commit, then by change set ID.

## Domain Model

| Concept             | Go type                   | Meaning                                                                                       |
| ------------------- | ------------------------- | --------------------------------------------------------------------------------------------- |
| ChangeSet           | `collection.ChangeSet`    | Commits implementing one change: same work-item key at the start of the subject, else one commit. |
| Change              | `collection.Change`       | One path added, modified, or deleted by one commit.                                           |
| Commit              | `collection.Commit`       | Hash, author, author date, and subject.                                                       |
| ArchitectureScope   | `architecture.Scope`      | Declared system, bounded context, and component of a path.                                    |
| Artifact            | `architecture.Artifact`   | Artifact kind of a path and the declared pattern it matched.                                  |
| Signal              | `detection.Signal`        | Factual observation of one rule in one change set.                                            |
| Evidence            | `detection.Evidence`      | Commit, path, change status, and matched declaration behind a signal.                         |
| AttentionCandidate  | `attention.Candidate`     | Change set with its level, commits, and signals.                                              |

Provenance runs Candidate -> Signal -> Evidence -> ChangeSet -> commit and path.

## Package Boundaries

```text
main (CLI, composition root, rendering)
  |
  +-- collection    local Git -> ChangeSets          (no architecture knowledge)
  +-- architecture  YAML -> declared architecture    (no Git knowledge)
  +-- detection     ChangeSets + architecture -> Signals
  +-- attention     ChangeSets + Signals -> Candidates
```

- `collection` is the only package that knows Git. It runs `git log` and parses its output.
- `architecture` is the only package that knows the YAML format and pattern matching.
- `detection` depends on `collection` and `architecture`.
- `attention` depends on `collection` and `detection`.
- `main.go` wires the pipeline and renders text or JSON. It stays one file so `go run main.go` works.

`.go-arch-lint.yml` enforces these dependencies. Each package's `doc.go` describes its responsibility.

## Collection Scope

- Commits reachable from `HEAD`, committed within the scan period. Merge commits are skipped.
- Renames are reported as a deletion and an addition.
- Work-item keys (`PAY-1421`, `[PAY-1421]`) are read from the start of the commit subject only, to
  avoid grouping on look-alikes such as `UTF-8`.

## Development

```sh
task all
```

`task all` formats, vets, checks dead code and architecture rules, lints, and runs all tests. The tests
build small fixture Git repositories in temporary directories.
