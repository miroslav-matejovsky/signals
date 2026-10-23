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
