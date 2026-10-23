# Plans

This directory contains implementation plans. A plan is an executable specification, not a discussion document.

Each plan has:
- `README.md` describing goal, scope, architecture impact, deliverables, and success criteria
- `alternatives.md` describing possible alternative approaches and their trade-offs
- `assessment.md` describing feasibility, risks, dependencies, validation, and rollback
- `progress.md` describing the current status by explicitly listing completed and pending steps in simple table format
- numbered implementation steps (`01-*.md`, `02-*.md`, ...)

Each step contains YAML front matter:

```yaml
title: "01 - Step title"
dependencies: []
effort: "S|M|L|XL"
complexity: "low|medium|high|very_high"
```

Each step must contain:
- **Objective**: exact outcome of the step
- **Target Artifacts**: files, modules, APIs, schemas, configurations, and tests to modify
- **Implementation Tasks**: ordered, concrete development actions
- **Technical Details**: classes, methods, interfaces, contracts, migrations, and configuration changes
- **Verification**: exact commands and validation activities
- **Acceptance Criteria**: binary, objectively testable conditions
- **Non-Goals**: what is explicitly not implemented

Step order:
- Order steps outside-in: start at the user-facing outcome and move toward the supporting detail (UI -> API -> domain -> storage).
- Step `01` defines the contracts the user-facing behavior needs: interfaces, request and response types, errors, and configuration keys. Interfaces are declared next to their consumer.
- Each following step replaces one contract with a real implementation, working from the outermost layer inward.
- No step introduces code that nothing calls. New code is wired into a caller reachable from an entry point in the same step, so dead code checks stay green.
- Until a layer is implemented, its contract is satisfied by the simplest failing implementation, for example a stub returning a "not implemented" error. Stubs are replaced, never accumulated.
- Every step is verifiable on its own. A step that can only be verified once a later step exists is ordered incorrectly; split it or move the verification into the step that completes the behavior.

Requirements:
- Plan must be understandable and actionable even by a medium-skilled developer.
- Plan must be evaluated against the principles defined in the `dev/principles` folder.
- Every task must describe **what to change, where to change it, and how it will be verified**
- Reference specific files, APIs, classes, methods, database objects, and tests whenever possible
- Define implementation order explicitly through dependencies
- Include required unit, integration, and regression tests or justify their absence
- Acceptance criteria must be measurable and independently verifiable

Forbidden:
- Bottom-up step order that builds storage or infrastructure before the behavior that consumes it
- Steps that leave code unreferenced until a later step
- Vague work items such as "improve", "enhance", "optimize", "refactor", or "investigate"
- Open-ended research tasks
- Future considerations, ideas, or alternatives
- Statements without implementation details
- Acceptance criteria that require subjective judgment

A plan is complete only when all implementation tasks are finished, all tests pass, all acceptance criteria are verified, and the deliverables exist in the codebase.

## Active plan sets

| Name | Description |
|------|-------------|
| Example Plan | Demonstrates the structure and content of a typical implementation plan |