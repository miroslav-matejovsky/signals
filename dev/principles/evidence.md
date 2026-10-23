# Evidence Before Assumption

> **Observe. Measure. Then conclude.**

Evidence is operational knowledge about behavior. An explanation that is not measured is a
hypothesis, however plausible it sounds. Replace assumptions with observations: when
uncertainty matters, instrument the system and measure it. Good evidence makes behavior
visible, reproducible, and explainable.

## Principle

**Do not assume what can be measured.**

- When behavior is uncertain, instrument the system and observe it.
- Unexpected behavior is an evidence gap, not an invitation to speculate.
- Diagnostics come before speculative fixes.
- Invariants are stated explicitly and verified at runtime where practical.
- Observed facts, verified facts, hypotheses, and unknowns are kept apart and labeled.
- Reproducible measurements outrank anecdotal observations.
- A fix is not understood until its effect has been observed and verified.

## Development Rule

**No unexplained behavior. No unmarked assumptions.**

If behavior is uncertain, measure it.

If behavior cannot be measured, instrument it.

If an assumption cannot be verified yet, label it as a hypothesis or an unknown.

If a defect cannot be explained from available evidence, improve observability before
changing behavior.

## Reasoning Loop

    Observation
      -> Measurement
      -> Hypothesis
      -> Experiment / Reproduction
      -> Verification Against Invariants
      -> Conclusion
      -> Change
      -> Measurement Of Resulting Behavior

Avoid:

    Symptom -> Plausible Explanation -> Code Change

Prefer:

    Symptom -> Observation -> Measurement -> Hypothesis -> Verification -> Change -> Measurement

## Guiding Rule

Hypotheses are necessary for investigation. The objective is not to eliminate them.

The objective is to prevent unverified hypotheses from becoming engineering facts.