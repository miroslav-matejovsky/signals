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
