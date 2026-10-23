// Package attention determines which detected signals deserve architect
// attention.
//
// Its purpose is reduction rather than additional analysis. It transforms a
// potentially large collection of signals into a small set of explainable
// attention candidates.
//
// This package owns:
//   - signal aggregation per ChangeSet
//   - suppression of ChangeSets without signals
//   - prioritization into HIGH and MEDIUM
//   - attention candidates
//
// Prioritization is deterministic: a candidate with two or more distinct
// signal types is HIGH, a candidate with one is MEDIUM. There are no numerical
// scores. A suppressed ChangeSet has unknown architectural impact; it is not
// assessed as safe.
//
// Every attention candidate carries the signals that caused it, with their
// evidence, and the commits of its ChangeSet, preserving provenance back to
// the underlying change.
//
// The package should favor precision over completeness: increasing the number
// of reported candidates at the cost of substantial noise is considered a
// degradation.
package attention
