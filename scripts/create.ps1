$packages = @{
    "collection" = @'
// Package collection collects and normalizes development changes.
//
// It integrates with source-control and work-tracking systems and converts
// provider-specific data into a common ChangeSet representation.
//
// A ChangeSet represents the intent of a change rather than an individual
// commit. Where possible, commits and pull requests are correlated with their
// originating work item.
//
// This package owns:
//   - repository and change discovery
//   - commit and pull-request metadata
//   - changed-file metadata
//   - work-item correlation
//   - normalization into ChangeSets
//
// It does not determine architectural significance or decide whether a change
// requires architect attention.
package collection
'@

    "architecture" = @'
// Package architecture represents the intended architecture used to interpret
// development changes.
//
// The model is deliberately small. It maps repositories and paths to
// architectural concepts such as systems, bounded contexts, and components.
// It can also identify artifacts with architectural significance, including
// public contracts, persistence schemas, deployment configuration, and
// security configuration.
//
// Architecture is declared explicitly rather than inferred from source code.
//
// This package owns:
//   - systems
//   - bounded contexts
//   - components
//   - repository and path mappings
//   - architectural artifact classifications
//
// It is not intended to become a complete architecture repository, dependency
// model, or CMDB. It contains only the architectural knowledge required by
// Signals to interpret changes.
package architecture
'@

    "detection" = @'
// Package detection identifies architectural signals in development changes.
//
// Detection combines normalized ChangeSets with the declared architecture and
// produces evidence of potentially significant architectural change.
//
// Example signals include:
//   - changes spanning multiple bounded contexts
//   - public contract changes
//   - persistence schema changes
//   - deployment changes
//   - security-boundary changes
//   - creation or removal of architectural components
//
// Signals describe observable evidence. They do not assert that an architecture
// is correct, incorrect, compliant, or violated.
//
// Every detected signal must retain provenance to the observations that caused
// it so that its origin can be inspected without relying on inference or AI.
package detection
'@

    "attention" = @'
// Package attention determines which detected signals deserve architect
// attention.
//
// Its purpose is reduction rather than additional analysis. It transforms a
// potentially large collection of signals into a small set of explainable
// attention candidates.
//
// This package owns:
//   - signal aggregation
//   - suppression
//   - deduplication
//   - prioritization
//   - attention candidates
//   - architect disposition and feedback
//
// Every attention candidate must explain which signals caused it and preserve
// provenance back to the underlying change.
//
// The package should favor precision over completeness: increasing the number
// of reported candidates at the cost of substantial noise is considered a
// degradation.
package attention
'@
}

foreach ($name in $packages.Keys) {
    New-Item -ItemType Directory -Path $name -Force | Out-Null
    Set-Content -Path "$name/doc.go" -Value $packages[$name] -Encoding utf8
}