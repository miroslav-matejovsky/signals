// Package detection identifies architectural signals in development changes.
//
// Detection combines normalized ChangeSets with the declared architecture and
// produces evidence of potentially significant architectural change. Each rule
// produces at most one Signal per ChangeSet:
//   - CROSS_CONTEXT_CHANGE: files in two or more bounded contexts changed
//   - PUBLIC_CONTRACT_CHANGED: a public-contract artifact changed
//   - PERSISTENCE_SCHEMA_CHANGED: a persistence-schema artifact changed
//   - DEPLOYMENT_CHANGED: a deployment artifact changed
//   - SECURITY_CHANGED: a security artifact changed
//   - COMPONENT_CREATED_OR_REMOVED: a declared component directory was
//     created or removed
//
// Signals describe observable evidence. They do not assert that an architecture
// is correct, incorrect, compliant, or violated.
//
// Every detected signal retains provenance to the observations that caused it:
// each Evidence names the commit, the path, how the path changed, and the
// declaration it matched, so that its origin can be inspected without relying
// on inference or AI.
package detection
