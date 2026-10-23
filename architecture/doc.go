// Package architecture represents the intended architecture used to interpret
// development changes.
//
// The model is deliberately small. It is loaded from a declarative YAML file
// that maps repositories and paths to systems, bounded contexts, and
// components. The same file classifies paths as artifacts with architectural
// significance: public contracts, persistence schemas, deployment
// configuration, and security configuration.
//
// Architecture is declared explicitly rather than inferred from source code.
// Paths and patterns are slash-separated and relative to the repository root.
// In patterns, a "**" segment matches zero or more path segments.
//
// This package owns:
//   - systems
//   - bounded contexts
//   - components
//   - repository and path mappings (Architecture.Resolve)
//   - architectural artifact classifications (Architecture.Classify)
//
// It is not intended to become a complete architecture repository, dependency
// model, or CMDB. It contains only the architectural knowledge required by
// Signals to interpret changes.
package architecture
