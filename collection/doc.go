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
