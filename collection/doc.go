// Package collection collects and normalizes development changes.
//
// It reads the commit history of a local Git repository for a time period by
// running the git command, and converts it into a common ChangeSet
// representation. Git-specific details stay inside this package.
//
// A ChangeSet represents the intent of a change rather than an individual
// commit. Commits whose subject starts with the same work-item key, such as
// PAY-1421, are correlated into one ChangeSet. Every other commit forms its own
// ChangeSet.
//
// This package owns:
//   - repository and change discovery
//   - commit metadata
//   - changed-file metadata
//   - created and removed directories
//   - work-item correlation from commit subjects
//   - normalization into ChangeSets
//
// It does not determine architectural significance or decide whether a change
// requires architect attention.
package collection
