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
